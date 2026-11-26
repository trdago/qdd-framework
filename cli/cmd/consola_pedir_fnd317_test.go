package cmd

import (
	"bytes"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'

func TestConsolaPedir_DeNuevoUnoQueFalloVuelveAPendiente_FND317(t *testing.T) {
	root := t.TempDir()
	t.Setenv("QDD_HOME", t.TempDir())
	var salida bytes.Buffer
	pedir := func() {
		t.Helper()
		if codigo := consolaPedir(root, "falla a propósito", "", consola.Orden{}, []string{"exit 3"}, &salida, &salida); codigo != 0 {
			t.Fatalf("no quedó pedido: %s", salida.String())
		}
	}
	pedir()
	a, _ := consola.BuscarAccion(consola.Acciones(root, false), "pedido:"+consola.Pedidos(root)[0].ID)
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 3 {
		t.Fatalf("el pedido tenía que fallar con 3: %d", codigo)
	}
	pedir()
	if cola := consola.Pedidos(root); len(cola) != 1 || cola[0].Estado != consola.EstadoPendiente {
		t.Errorf("el que falló sigue sin poder correr: %+v", cola)
	}
}
