package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// «Para ti» de todos los proyectos: con --dir, un comando de la consola actúa en otro proyecto, no en la carpeta actual.
func TestConsola_DirEligeElProyecto(t *testing.T) {
	otro := t.TempDir()
	if err := os.MkdirAll(filepath.Join(otro, ".qdd"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { estadoDir = "" })
	if err := consolaCmd.PersistentFlags().Set("dir", otro); err != nil {
		t.Fatalf("la consola acepta --dir: %v", err)
	}
	if got := estadoRaiz(); got != otro {
		t.Fatalf("con --dir actúa en %s, no en %s", otro, got)
	}
}
