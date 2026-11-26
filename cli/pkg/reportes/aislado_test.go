package reportes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/aislar"
	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'

// TestMain corre las pruebas con un QDD_HOME temporal: una prueba que olvide fijarlo no deja su llave ni sus reportes
// en la carpeta de la persona.
func TestMain(m *testing.M) {
	os.Exit(aislar.Correr(m))
}

// La llave, los pendientes y los enviados de las pruebas quedan en una carpeta temporal, nunca en el QDD_HOME real.
// contrato: reportes#8036988e
func TestPruebas_NoTocanElQDDHomeReal(t *testing.T) {
	home, err := blueprints.Home()
	if err != nil {
		t.Fatal(err)
	}
	tmp, _ := filepath.EvalSymlinks(os.TempDir())
	real, _ := filepath.EvalSymlinks(home)
	if !strings.HasPrefix(real, tmp+string(filepath.Separator)) {
		t.Fatalf("QDD_HOME de las pruebas fuera de la carpeta temporal: %s", home)
	}
	if _, err := Llave(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(real, "reportes", "llave")); err != nil {
		t.Errorf("la llave de prueba no quedó en el QDD_HOME temporal: %v", err)
	}
}
