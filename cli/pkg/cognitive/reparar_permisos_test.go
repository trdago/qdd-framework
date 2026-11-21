package cognitive

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Reparar con permisos de escritura es una puerta aparte: solo AttemptRepair de un backend RepairCapable lanza el agente
// con --dangerously-skip-permissions; Ask, el camino de siempre, nunca.

// claudeQueAnota deja en el PATH un claude falso que escribe cada argumento en su línea y devuelve dónde.
func claudeQueAnota(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("el claude falso es un script de sh")
	}
	dir := t.TempDir()
	args := filepath.Join(dir, "args")
	guion := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ARGS_DE_CLAUDE\"\necho respuesta\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(guion), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARGS_DE_CLAUDE", args)
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("CLAUDE_CODE_EXECPATH", "")
	return args
}

func argumentosDe(t *testing.T, ruta string) string {
	t.Helper()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("el claude falso no corrió: %v", err)
	}
	os.Remove(ruta)
	return string(b)
}

// soloPregunta es un backend disponible que no es RepairCapable y anota si alguien le preguntó.
type soloPregunta struct{ preguntado bool }

func (s *soloPregunta) Name() string    { return "solo-pregunta" }
func (s *soloPregunta) Available() bool { return true }
func (s *soloPregunta) Ask(context.Context, string) (string, error) {
	s.preguntado = true
	return "ok", nil
}

// contrato: cognitive#8c7f35f8
func TestAsk_NuncaPasaPermisosDeEscrituraYReparar_Si(t *testing.T) {
	args := claudeQueAnota(t)
	b := &ClaudeBackend{}
	if _, err := b.Ask(context.Background(), "¿está bien?"); err != nil {
		t.Fatal(err)
	}
	if a := argumentosDe(t, args); strings.Contains(a, "--dangerously-skip-permissions") || !strings.Contains(a, "¿está bien?") {
		t.Fatalf("Ask corre claude -p sin permisos de escritura: %q", a)
	}
	if _, err := b.AttemptRepair(context.Background(), t.TempDir(), "arréglalo"); err != nil {
		t.Fatal(err)
	}
	if a := argumentosDe(t, args); !strings.Contains(a, "--dangerously-skip-permissions") {
		t.Fatalf("reparar es lo único que pide permisos de escritura: %q", a)
	}
}

// contrato: cognitive#8c7f35f8
func TestAttemptRepair_SaltaLosBackendsQueSoloPreguntan(t *testing.T) {
	s := &soloPregunta{}
	if _, _, err := attemptRepairUsing(context.Background(), t.TempDir(), "arréglalo", []Backend{s}); err == nil {
		t.Fatal("sin un backend RepairCapable no hay reparación")
	}
	if s.preguntado {
		t.Fatal("reparar nunca cae en Ask de un backend que no es RepairCapable")
	}
}
