package contexto

import "github.com/qdd-framework/qdd/pkg/gitenv"

// Las pruebas corren git en carpetas temporales: sin el GIT_DIR de un hook, que las haría escribir en el repositorio
// real (FND-117).
func init() {
	gitenv.LimpiarProceso()
}
