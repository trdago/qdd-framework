package estado

import (
	"os"
	"testing"

	"github.com/qdd-framework/qdd/pkg/aislar"
	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// Las pruebas corren git en carpetas temporales: sin el GIT_DIR de un hook, que las haría escribir en el repositorio
// real (FND-117).
func init() {
	gitenv.LimpiarProceso()
}

// TestMain corre las pruebas con un QDD_HOME temporal: sin él, dejaban carpetas en el de la persona (FND-246).
func TestMain(m *testing.M) {
	os.Exit(aislar.Correr(m))
}
