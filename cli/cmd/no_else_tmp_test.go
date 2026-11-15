package cmd

import (
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// Regresión FND-219: el método manda los scripts de un solo uso a ./tmp (git lo ignora), y TestNoElseInCode los
// revisaba igual: un script local con un else hacía fallar el gate de un cambio que no lo tocaba. Solo se salta el tmp
// de la raíz; una carpeta tmp dentro del código se sigue revisando.
func TestNoElse_SaltaSoloElTmpDeLaRaiz_FND219(t *testing.T) {
	root := filepath.Join("..", "..")
	if checkSkipDir(root, filepath.Join(root, "tmp"), "tmp") != filepath.SkipDir {
		t.Fatal("el tmp de la raíz es temporal y no se revisa")
	}
	if checkSkipDir(root, filepath.Join(root, "cli", "pkg", "tmp"), "tmp") != nil {
		t.Fatal("una carpeta tmp dentro del código se revisa")
	}
	if checkSkipDir(root, filepath.Join(root, "node_modules"), "node_modules") != filepath.SkipDir {
		t.Fatal("las carpetas que ya se saltaban se siguen saltando")
	}
}
