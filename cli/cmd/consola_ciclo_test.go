package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
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

// Sincronía panel–chat (2026-11-01): con --contexto, el ciclo imprime qué espera el panel de la persona, una vez.
func TestConsolaCiclo_ConContextoDiceLoQueEsperaElPanelUnaVez(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".qdd"), 0o755)
	if _, err := consola.Pedir(root, "gh pr merge 9", "Merge #9", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	estadoDir, consolaModo, consolaContexto = root, "auto", true
	defer func() { estadoDir, consolaModo, consolaContexto = "", "", false }()
	var primero, otra bytes.Buffer
	correrCiclo(root, &primero)
	if !strings.Contains(primero.String(), "«Merge #9»") {
		t.Fatalf("dice el pedido: %q", primero.String())
	}
	if correrCiclo(root, &otra); otra.Len() != 0 {
		t.Fatalf("sin cambios no imprime: %q", otra.String())
	}
}
