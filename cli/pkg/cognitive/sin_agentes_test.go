package cognitive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FND-172: cuatro pruebas lanzaban un agente real (claude -p) en cada go test: unos 127 s, tokens gastados, un resultado
// que dependía de la máquina y un agente corriendo dentro del árbol que se estaba probando.

// agentesDeMentira deja en el PATH un claude, un agentapi y un cursor-agent que solo anotan que alguien los lanzó.
func agentesDeMentira(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	marca := filepath.Join(dir, "lanzado")
	for _, nombre := range []string{"claude", "agentapi", "cursor-agent", "cursor"} {
		guion := "#!/bin/sh\n: > \"$MARCA_AGENTE\"\necho respuesta\n"
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(guion), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MARCA_AGENTE", marca)
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("CLAUDE_CODE_EXECPATH", "")
	return marca
}

func sinLanzar(t *testing.T, marca string) {
	t.Helper()
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("una prueba lanzó un agente real")
	}
}

func TestEnUnaPruebaNoHayMotorAunqueElAgenteEsteInstalado_FND172(t *testing.T) {
	marca := agentesDeMentira(t)
	if info := DetectLocalLLM(); info.Detected || info.Status != "OFFLINE" {
		t.Fatalf("en una prueba no hay motor: %+v", info)
	}
	sinLanzar(t, marca)
}

func TestExecuteLocalIntent_EnUnaPruebaNoLanzaNingunAgente_FND172(t *testing.T) {
	marca := agentesDeMentira(t)
	res, err := ExecuteLocalIntent(context.Background(), ".", "Crea una función de suma")
	if err != nil || res.Status != "OFFLINE_GUIDANCE" {
		t.Fatalf("%+v %v", res, err)
	}
	sinLanzar(t, marca)
}

func TestAsk_EnUnaPruebaNoLanzaNingunAgente_FND172(t *testing.T) {
	marca := agentesDeMentira(t)
	if _, _, err := Ask(context.Background(), "hola"); err == nil {
		t.Fatal("no hay agente que responda")
	}
	sinLanzar(t, marca)
}

// AttemptRepair corre con permisos de escritura: es la que menos puede salir de una prueba.
func TestAttemptRepair_EnUnaPruebaNoLanzaNingunAgente_FND172(t *testing.T) {
	marca := agentesDeMentira(t)
	if _, _, err := AttemptRepair(context.Background(), t.TempDir(), "repara"); err == nil {
		t.Fatal("no hay agente que repare")
	}
	sinLanzar(t, marca)
}

// Lo que antes solo se probaba lanzando un agente de verdad, y sin mirar el resultado.
func TestEjecutarIntento_ConUnAgenteDevuelveSuRespuesta_FND172(t *testing.T) {
	var recibido string
	preguntar := func(_ context.Context, prompt string) (string, string, error) {
		recibido = prompt
		return "respuesta", "claude", nil
	}
	info := LocalLLMInfo{Provider: "Claude Code", SelectedModel: "m"}
	res, err := ejecutarIntento(context.Background(), info, preguntar, "Crea una función")
	if err != nil || res.Status != "SUCCESS" || res.Output != "respuesta" || res.Provider != "claude" || res.Model != "m" {
		t.Fatalf("%+v %v", res, err)
	}
	if !strings.Contains(recibido, "Crea una función") {
		t.Fatalf("el pedido llega con sus reglas: %q", recibido)
	}
	if len(res.GovernanceChecks) == 0 {
		t.Fatal("dice qué reglas aplicó")
	}
}

func TestEjecutarIntento_SiElAgenteFallaDevuelveElError_FND172(t *testing.T) {
	falla := errors.New("sin cuota")
	preguntar := func(context.Context, string) (string, string, error) { return "", "", falla }
	res, err := ejecutarIntento(context.Background(), LocalLLMInfo{Provider: "Antigravity"}, preguntar, "x")
	if !errors.Is(err, falla) || res.Status != "ERROR" || !strings.Contains(res.Output, "Antigravity") {
		t.Fatalf("%+v %v", res, err)
	}
}
