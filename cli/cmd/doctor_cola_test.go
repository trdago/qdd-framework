package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'

// FND-081: doctor revisa que la cola de pedidos esté en el repositorio que corresponde y --fix la junta, sin
// reactivar la consola por eso.
func TestDoctor_RevisaLaColaDePedidosYLaRepara_FND081(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	g := buildConsolaChecks(root)
	if len(g.Items) != 0 {
		t.Fatalf("sin consola activa ni pedidos fuera de lugar, el grupo no aparece: %+v", g.Items)
	}
	otro := t.TempDir()
	os.MkdirAll(filepath.Join(otro, ".qdd"), 0755)
	ruta, _ := consola.RutaLocal(root, "pedidos")
	os.MkdirAll(filepath.Dir(ruta), 0700)
	os.WriteFile(ruta, []byte(`[{"id":"1","comando":"exit 0","motivo":"m","carpeta":"`+otro+`","tecla":"a","estado":"pendiente"}]`), 0600)
	g = buildConsolaChecks(root)
	if len(g.Items) != 1 || g.Items[0].Success || g.Items[0].Name != "Cola de pedidos del panel (una por repositorio)" {
		t.Fatalf("un pedido de otro proyecto aparece como problema: %+v", g.Items)
	}
	repairConsola(root, g)
	if len(consola.Pedidos(root)) != 0 || len(consola.Pedidos(otro)) != 1 {
		t.Fatalf("--fix lo pasa a la cola de su proyecto: %+v %+v", consola.Pedidos(root), consola.Pedidos(otro))
	}
}
