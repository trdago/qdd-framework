package audit

import (
	"io/fs"

	"github.com/qdd-framework/qdd/pkg/contexto"
)

// walkProject recorre el proyecto saltando lo que no pertenece a su código
// versionado (repos anidados, worktrees y carpetas que git ignora). Es el
// mismo recorrido que usa el Contexto (contexto.Recorrer), así la auditoría y
// el Contexto ven el mismo repositorio.
func walkProject(root string, fn fs.WalkDirFunc) error {
	return contexto.Recorrer(root, fn)
}
