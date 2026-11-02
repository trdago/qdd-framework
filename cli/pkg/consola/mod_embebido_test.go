package consola

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-107: rc.11 instaló el mod sin hooks/materializar.js (el go:embed nombraba solo register.js) y Claude Code no
// cargó el panel: «hooks module did not load: …/materializar.js: no such file». Todo lo que el mod trae fuera de sus
// pruebas tiene que ir en el binario.
func TestModEmbebido_LlevaTodosLosArchivosDelMod_FND107(t *testing.T) {
	err := fs.WalkDir(os.DirFS("mod/qdd-tareas"), ".", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (ruta == "tests" || ruta == ".claude-plugin/types") {
			return fs.SkipDir
		}
		// Lo que Claude Code o git dejan en la carpeta (.gitignore) no es del mod.
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if _, err := modFS.ReadFile("mod/qdd-tareas/" + ruta); err != nil {
			t.Errorf("el binario no lleva %s: el panel instalado no carga", ruta)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
