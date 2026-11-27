package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// FND-334: lo local que quedó con la clave de más de 80 caracteres aparece en doctor y --fix lo renombra, sin
// reactivar la consola por eso.
func TestDoctor_RenombraLoLocalConLaClaveLarga_FND334(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	base, _ := filepath.EvalSymlinks(t.TempDir())
	root := filepath.Join(base, strings.Repeat("carpeta-larga-", 7))
	os.MkdirAll(filepath.Join(root, ".qdd"), 0o755)
	nueva, _ := estado.RutaLocal(root, "telefono")
	larga := strings.Repeat("carpeta-larga-", 7)[:97] + "-" + strings.TrimSuffix(filepath.Base(nueva), ".json")[len(filepath.Base(nueva))-5-12:]
	vieja := filepath.Join(home, "claude", "telefono", larga+".json")
	os.MkdirAll(filepath.Dir(vieja), 0o700)
	os.WriteFile(vieja, []byte(`{}`), 0o600)

	g := buildConsolaChecks(root)
	if len(g.Items) != 1 || g.Items[0].Success || g.Items[0].Name != nombreClaveLarga {
		t.Fatalf("doctor ve lo local con la clave larga: %+v", g.Items)
	}
	repairConsola(root, g)
	if _, err := os.Stat(nueva); err != nil {
		t.Fatalf("--fix lo renombra a la clave nueva: %v", err)
	}
	if g := buildConsolaChecks(root); len(g.Items) != 0 {
		t.Fatalf("después de --fix no queda nada que revisar: %+v", g.Items)
	}
}
