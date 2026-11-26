package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

// La IA mueve el panel con qdd consola accion <id>, que solo ejecuta lo que el panel ofrece: una herramienta nunca está
// entre esas acciones, así que la IA no la corre desde el panel ni con --confirmado; la corre el clic de la persona en
// su pestaña (qdd herramienta correr).
// contrato: herramientas#24e60e63
func TestHerramienta_NoEsUnaAccionDelPanelQueLaIAPuedaCorrer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("la herramienta de prueba es un script de sh")
	}
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	marca := filepath.Join(t.TempDir(), "corrio")
	os.MkdirAll(filepath.Join(root, ".qdd", "herramientas"), 0o755)
	os.MkdirAll(filepath.Join(root, "bin"), 0o755)
	os.WriteFile(filepath.Join(root, "bin", "sim"), []byte("#!/bin/sh\necho \"$@\" >> '"+marca+"'\n"), 0o755)
	def := "id: sim\nnombre: Sim\ncomando: [./bin/sim]\nacciones:\n  - {id: ver, nombre: Ver, args: [ver]}\nprueba: {args: [--help]}\n"
	os.WriteFile(filepath.Join(root, ".qdd", "herramientas", "sim.yaml"), []byte(def), 0o644)

	datos := datosPanel(root, time.Now(), []string{"herramientas", "atender", "pregunta", "telefono", "flujos"})
	if !strings.Contains(string(datos.Herramientas), `"sim"`) {
		t.Fatalf("el panel la lista para dibujar su pestaña: %s", datos.Herramientas)
	}
	for _, a := range datos.Acciones {
		if strings.Contains(a.ID, "sim") || strings.Contains(a.ID, "herramienta") {
			t.Fatalf("una herramienta no es una acción del panel: %+v", a)
		}
	}
	for _, id := range []string{"herramienta:sim", "herramienta:sim:ver", "sim", "sim:ver"} {
		var salida, errores bytes.Buffer
		if c := consolaAccion(root, id, true, "sesion-ia", &salida, &errores); c == 0 {
			t.Fatalf("qdd consola accion %s no corre una herramienta: %s", id, salida.String())
		}
	}
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("la herramienta corrió sin el clic de la persona")
	}
}
