package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/integration"
)

// Rule: Prohibido usar 'else'
// doctor --fix quita el bloque de QDD de los AGENTS.md de las carpetas padre hasta integration.TopePadres. Durante las
// pruebas ese tope es la carpeta temporal de TestMain: ni el HOME de la persona ni una carpeta de la que cuelgue el
// repositorio, así que ninguna prueba alcanza sus AGENTS.md reales.

// contrato: integracion#92b2bc8e
func TestPruebas_DoctorNoAlcanzaLosAGENTSDeLaPersona(t *testing.T) {
	tope := integration.TopePadres()
	if home, err := os.UserHomeDir(); err == nil && tope == home {
		t.Fatalf("durante las pruebas el tope de las carpetas padre es el HOME de la persona: %s", tope)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(tope, cwd)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("el repositorio (%s) cuelga del tope %s: doctor --fix tocaría sus AGENTS.md", cwd, tope)
	}
	if copias := integration.BloquesEnPadres(cwd); len(copias) != 0 {
		t.Fatalf("desde el repositorio no se alcanza ningún AGENTS.md de afuera: %v", copias)
	}
}
