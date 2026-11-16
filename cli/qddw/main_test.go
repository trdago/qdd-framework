package main

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238): en Windows la tarea de QDD Móvil lanzaba qdd.exe, un programa de consola, y Windows abría una
// ventana que había que dejar abierta. qddw.exe es un programa de ventanas (nunca abre consola) que corre el qdd.exe de
// su carpeta sin ventana y sale con su código.

func TestLanzador_CorreElQDDDeSuCarpetaConLosArgumentosYSaleConSuCodigo_FND238(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("el qdd de prueba es un guion de sh; en Windows lo prueba TestSinVentana")
	}
	dir := t.TempDir()
	salida := filepath.Join(dir, "salida")
	guion := "#!/bin/sh\necho \"$@ $" + envSinConsola + "\" > " + salida + "\nexit 7\n"
	if err := os.WriteFile(filepath.Join(dir, "qdd"), []byte(guion), 0o755); err != nil {
		t.Fatal(err)
	}
	if c := correr(filepath.Join(dir, "qddw"), []string{"remoto", "correr"}); c != 7 {
		t.Fatalf("sale con el código de qdd (la tarea lo relanza si es un error): %d", c)
	}
	b, _ := os.ReadFile(salida)
	if string(b) != "remoto correr 1\n" {
		t.Fatalf("corre qdd con los mismos argumentos y le dice que no tiene consola: %q", b)
	}
}

func TestLanzador_SinQDDEnSuCarpetaSaleConDos_FND238(t *testing.T) {
	if c := correr(filepath.Join(t.TempDir(), "qddw"), []string{"remoto", "correr"}); c != codigoSinQDD {
		t.Fatalf("sin qdd al lado no hay nada que lanzar: %d", c)
	}
}

func TestLanzador_LaVariableEsLaQueQDDLee_FND238(t *testing.T) {
	if envSinConsola != remoto.EnvSinConsola {
		t.Fatalf("qddw y qdd usan la misma variable: %q, %q", envSinConsola, remoto.EnvSinConsola)
	}
}

// Las banderas son las del release (publish.yml): un programa de ventanas, sin símbolos. Se compila para Windows desde
// cualquier sistema y se mira el encabezado PE.
func TestLanzador_SeCompilaComoProgramaDeVentanasYChico_FND238(t *testing.T) {
	if testing.Short() {
		t.Skip("compila qddw.exe")
	}
	exe := filepath.Join(t.TempDir(), "qddw.exe")
	c := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-buildvcs=false", "-ldflags", banderasRelease, "-o", exe, ".")
	c.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("compila: %v\n%s", err, out)
	}
	f, err := pe.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	h, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || h.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
		t.Fatalf("es un programa de ventanas: Windows no le abre consola (%+v)", f.OptionalHeader)
	}
	info, _ := os.Stat(exe)
	if info.Size() > 4<<20 {
		t.Fatalf("es chico: va dentro de qdd.exe (%d bytes)", info.Size())
	}
	if !strings.Contains(banderasRelease, "-H=windowsgui") {
		t.Fatal("las banderas del release lo hacen de ventanas")
	}
}
