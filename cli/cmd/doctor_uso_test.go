package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/uso"
)

// Rule: Prohibido usar 'else'
// Uso medido (2026-10-30): doctor muestra lo medido de la versión que corre y aparta un registro ilegible, que si no
// dejaría de sumar sin que nadie lo note.
func TestDoctor_UnRegistroDeUsoIlegibleSeAparta(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".qdd"), 0755)
	dir, _ := uso.Carpeta()
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "2.8.0-rc.1.json"), []byte("{roto"), 0600)
	g := buildConsolaChecks(root)
	var item *CheckItem
	for _, it := range g.Items {
		if it.Name == nombreUso {
			item = it
		}
	}
	if item == nil || item.Success || !strings.Contains(item.Error, "ilegible") {
		t.Fatalf("doctor ve el registro ilegible: %+v", g.Items)
	}
	repairConsola(root, g)
	if _, ilegibles := uso.Todas(); len(ilegibles) != 0 {
		t.Fatalf("--fix lo aparta: %v", ilegibles)
	}
	if _, err := os.Stat(filepath.Join(dir, "2.8.0-rc.1.json.ilegible")); err != nil {
		t.Fatal("queda apartado, no borrado")
	}
}
