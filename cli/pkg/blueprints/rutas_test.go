package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'

// FND-084: el estado por tareas vive en .qdd/project/estado.yaml (lo escribe qdd estado), pero ninguna ruta dentro
// de .qdd se aceptaba: cada proyecto inventaba una ruta distinta para su ficha (docs, .agents/rules/…).
func TestValidateRutas_SoloElEstadoDelProyectoPuedeVivirEnQdd_FND084(t *testing.T) {
	if problems := validateRutas([]string{".qdd/project/estado.yaml"}); len(problems) != 0 {
		t.Fatalf("el archivo de estado de QDD es una ruta válida para su ficha: %v", problems)
	}
	for _, r := range []string{".qdd", ".qdd/project/modules", ".qdd/project/estado.yaml/../x.go", ".qdd/config.yaml"} {
		if problems := validateRutas([]string{r}); len(problems) == 0 || !strings.Contains(problems[0], ".qdd") {
			t.Fatalf("%s sigue prohibida: el código nunca vive en .qdd (%v)", r, problems)
		}
	}
}

// FND-084: el esqueleto trataba toda ruta como carpeta: creaba .qdd/project/estado.yaml/README.md y el estado ya no
// se podía escribir.
func TestWriteSkeletons_UnaRutaDeArchivoNoSeVuelveCarpeta_FND084(t *testing.T) {
	root := t.TempDir()
	m := modules.Module{ID: "estado-por-tareas", Rutas: []string{".qdd/project/estado.yaml", "docs/guia.md", "internal/estado"}, Pruebas: modules.Pruebas{Goldensets: []string{"g.yaml"}}}
	creados, err := writeSkeletons(root, mustParse(t, authBlueprint), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(creados) != 1 || creados[0] != "internal/estado" {
		t.Fatalf("solo una carpeta lleva esqueleto: %v", creados)
	}
	for _, archivo := range []string{".qdd/project/estado.yaml", "docs/guia.md"} {
		if _, err := os.Stat(filepath.Join(root, archivo)); !os.IsNotExist(err) {
			t.Fatalf("%s no se crea: lo escribe su propio comando (%v)", archivo, err)
		}
	}
}
