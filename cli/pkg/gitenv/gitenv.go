// Package gitenv quita del entorno las variables con las que git elige repositorio. Un hook de git (pre-push desde un
// worktree, pre-commit) exporta GIT_DIR y GIT_INDEX_FILE: un git que corre en otra carpeta las hereda y escribe en el
// repositorio del hook en vez de en el suyo (FND-117).
package gitenv

import (
	"os"
	"strings"
)

// Rule: Prohibido usar 'else'

// Variables son las que hacen que git ignore la carpeta en que corre.
var Variables = []string{
	"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE", "GIT_PREFIX", "GIT_CEILING_DIRECTORIES",
}

// Limpio devuelve env sin Variables.
func Limpio(env []string) []string {
	var out []string
	for _, kv := range env {
		if eligeRepositorio(kv) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// LimpiarProceso quita Variables del entorno de este proceso: para las pruebas, que corren git en carpetas temporales.
func LimpiarProceso() {
	for _, v := range Variables {
		os.Unsetenv(v)
	}
}

func eligeRepositorio(kv string) bool {
	nombre, _, _ := strings.Cut(kv, "=")
	for _, v := range Variables {
		if nombre == v {
			return true
		}
	}
	return false
}
