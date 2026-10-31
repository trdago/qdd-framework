package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// qdd consola ciclo anota el modo y qdd modules context lo usa para la regla de autorización.
func TestConsolaCiclo_ElContextoDiceElModo(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".qdd"), 0o755)
	estadoDir, consolaModo = root, "auto"
	defer func() { estadoDir, consolaModo = "", "" }()
	consolaCicloCmd.Run(consolaCicloCmd, nil)
	modo, visto := modoDeLaIA(root)
	if modo != "auto" || visto == "" {
		t.Fatalf("el ciclo anota el modo: %q %q", modo, visto)
	}
	ctx := modules.BuildContext(modules.ContextInput{Root: root, Modo: modo, ModoVisto: visto})
	if !strings.Contains(ctx, "«auto»") {
		t.Fatalf("el contexto lo dice: %s", ctx)
	}
}
