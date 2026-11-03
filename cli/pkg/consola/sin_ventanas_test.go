package consola

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-122 (issue #83, 2026-11-03): en Windows, «qdd consola prs» abría una ventana de consola visible cada vez que
// llamaba a gh. El proceso de fondo se lanzaba con DETACHED_PROCESS, sin consola, y Windows le da una ventana nueva a
// cada programa de consola que ese proceso arranca. Con CREATE_NO_WINDOW el proceso tiene una consola oculta que sus
// hijos heredan. La CI no corre en Windows, así que la prueba revisa el código de todos los lanzadores de fondo.

func TestSinVentanas_NingunProcesoDeFondoSeLanzaSinConsola_FND122(t *testing.T) {
	archivos, err := filepath.Glob(filepath.Join("..", "*", "*_windows.go"))
	if err != nil || len(archivos) == 0 {
		t.Fatalf("no se encontraron archivos de Windows en pkg: %v", err)
	}
	for _, a := range archivos {
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "DETACHED_PROCESS") {
			t.Errorf("%s usa DETACHED_PROCESS: cada gh o git que lance abre una ventana visible (FND-122)", a)
		}
	}
}

func TestSinVentanas_LosLanzadoresDeFondoOcultanLaConsola_FND122(t *testing.T) {
	for _, a := range []string{"separar_windows.go", filepath.Join("..", "incidentes", "separar_windows.go")} {
		b, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "CREATE_NO_WINDOW") || !strings.Contains(string(b), "CREATE_NEW_PROCESS_GROUP") {
			t.Errorf("%s: un proceso de fondo va con CREATE_NO_WINDOW y en su propio grupo (FND-122)", a)
		}
	}
}

func TestPRs_SinRemoteDeGitHubNoLlamaAGH_FND122(t *testing.T) {
	root := proyectoConCola(t)
	llamadas := 0
	correr := func(dir string, args ...string) ([]byte, error) {
		llamadas++
		return []byte("[]"), nil
	}
	for _, remote := range []string{"", "https://dev.azure.com/org/proyecto/_git/repo", "git@gitlab.com:a/b.git"} {
		if err := traerPRsCon(correr, func(string) string { return remote }, root, enCola); err != nil {
			t.Fatalf("remote %q: sin GitHub no hay PRs que traer y no es un error: %v", remote, err)
		}
	}
	if llamadas != 0 {
		t.Fatalf("sin un remote de GitHub, gh no se llama (se llamó %d veces)", llamadas)
	}
}

func TestPRs_ConRemoteDeGitHubPreguntaAGH_FND122(t *testing.T) {
	root := proyectoConCola(t)
	llamadas := 0
	correr := func(dir string, args ...string) ([]byte, error) {
		llamadas++
		return []byte(`[{"number":7,"title":"x","isDraft":false}]`), nil
	}
	for _, remote := range []string{"git@github.com:trdago/qdd-framework.git", "https://github.com/trdago/qdd-framework"} {
		if err := traerPRsCon(correr, func(string) string { return remote }, root, enCola); err != nil {
			t.Fatal(err)
		}
	}
	if llamadas != 2 {
		t.Fatalf("con remote de GitHub se pregunta a gh una vez por ciclo (se llamó %d veces)", llamadas)
	}
	if _, ok := pedidosDePR(t, root)[7]; !ok {
		t.Fatal("el PR #7 tenía que quedar como pedido")
	}
}
