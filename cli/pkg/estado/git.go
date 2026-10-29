package estado

import (
	"os/exec"
	"path/filepath"
)

// Anterior es la versión del archivo en git HEAD (para EPF-C05: un frente no desaparece en silencio); nil si no hay
// git, no hay commit o el archivo no estaba.
func Anterior(root string) *Estado {
	out, err := exec.Command("git", "-C", root, "show", "HEAD:"+filepath.ToSlash(Ruta)).Output()
	if err != nil {
		return nil
	}
	e, err := Parse(out)
	if err != nil {
		return nil
	}
	return &e
}
