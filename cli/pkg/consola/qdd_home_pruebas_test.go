package consola

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// FND-246: las pruebas de consola dejaban un archivo de turnos en el QDD_HOME de la persona por corrida (525 en este
// PC). Todo lo que escriben va a una carpeta temporal.
// contrato: gitenv#83f46fc1
func TestPruebas_NoEscribenEnElQDDHomeDeLaPersona_FND246(t *testing.T) {
	home, err := blueprints.Home()
	if err != nil {
		t.Fatal(err)
	}
	tmp, _ := filepath.EvalSymlinks(os.TempDir())
	real, _ := filepath.EvalSymlinks(home)
	if !strings.HasPrefix(real, tmp+string(filepath.Separator)) {
		t.Fatalf("QDD_HOME de las pruebas fuera de la carpeta temporal: %s", home)
	}
}
