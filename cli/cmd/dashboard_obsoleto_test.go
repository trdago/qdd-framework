package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RFC 0018: qdd dashboard queda obsoleto. Sigue funcionando, avisa al arrancar y nombra el panel que sigue.

func TestDashboard_AvisaQueEstaObsoletoAlArrancar(t *testing.T) {
	var salida, errores bytes.Buffer
	dashboardCmd.SetOut(&salida)
	dashboardCmd.SetErr(&errores)
	defer dashboardCmd.SetOut(nil)
	defer dashboardCmd.SetErr(nil)

	if dashboardCmd.PreRun == nil {
		t.Fatal("el aviso debe salir antes de levantar el servidor")
	}
	dashboardCmd.PreRun(dashboardCmd, nil)

	aviso := errores.String()
	for _, parte := range []string{"obsoleto", "sigue funcionando", "qdd-tareas", "Proyectos", "qdd consola activar"} {
		if !strings.Contains(aviso, parte) {
			t.Errorf("el aviso debe decir %q: %q", parte, aviso)
		}
	}
	if salida.Len() != 0 {
		t.Errorf("el aviso va por la salida de errores, no por la normal: %q", salida.String())
	}
}

func TestDashboard_ObsoletoSigueVisibleEnLaAyuda(t *testing.T) {
	if !dashboardCmd.IsAvailableCommand() {
		t.Fatal("sigue funcionando, así que no debe desaparecer de la ayuda")
	}
	if !strings.Contains(strings.ToLower(dashboardCmd.Short), "obsoleto") {
		t.Errorf("la ayuda debe decir que está obsoleto: %q", dashboardCmd.Short)
	}
}

func TestDocs_DicenQueDashboardEstaObsoleto(t *testing.T) {
	for _, doc := range []string{"docs/command-reference.md", "docs/getting-started.md", "README.md", "npm/README.md"} {
		contenido, err := os.ReadFile(filepath.Join("..", "..", doc))
		if err != nil {
			t.Fatalf("leer %s: %v", doc, err)
		}
		for _, linea := range strings.Split(string(contenido), "\n") {
			if !strings.Contains(linea, "qdd dashboard") {
				continue
			}
			if !strings.Contains(strings.ToLower(linea), "obsoleto") {
				t.Errorf("%s nombra qdd dashboard sin decir que está obsoleto: %q", doc, linea)
			}
		}
	}
}
