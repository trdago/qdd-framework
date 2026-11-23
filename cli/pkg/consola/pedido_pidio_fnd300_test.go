package consola

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-300 (qdd-movile, 2026-11-22): el pedido 8 de qdd-movile lo dejó una sesión el 15/11. Pedido de nuevo hoy por otra
// sesión viva, la cola reusó el abierto y conservó pidio de la vieja: el baúl buscó el latido de una sesión cerrada
// («no late en este proyecto») aunque quien lo pedía estaba viva. Quien lo pide de nuevo pasa a ser quien lo pidió: a
// ella le llegan el 🔑 del baúl y el aviso si se retira.

const (
	sesionVieja = "b88803e4-1291-4935-b6b7-83f4164ea7ba"
	sesionNueva = "d8344360-0000-4000-8000-000000000001"
)

func pidioDe(t *testing.T, root, id string) string {
	t.Helper()
	for _, p := range PedidosA(root, time.Now()) {
		if p.ID == id {
			return p.Pidio
		}
	}
	t.Fatalf("no está el pedido %s", id)
	return ""
}

func TestPedido_PedirloDeNuevoLoDejaALaSesionQueLoPideAhora_FND300(t *testing.T) {
	root := proyectoConCola(t)
	t.Setenv("CLAUDE_CODE_SESSION_ID", sesionVieja)
	p := mustPedirEnOrden(t, root, "./desplegar.sh", Orden{})
	t.Setenv("CLAUDE_CODE_SESSION_ID", sesionNueva)
	q := mustPedirEnOrden(t, root, "./desplegar.sh", Orden{DeNuevo: true})
	if q.ID != p.ID {
		t.Fatalf("es el mismo pedido abierto: %s y %s", p.ID, q.ID)
	}
	if s := pidioDe(t, root, p.ID); s != sesionNueva {
		t.Fatalf("lo pidió de nuevo una sesión viva: es la suya: %q", s)
	}
}

func TestPedido_PedirloSinSesionNoLeBorraQuienLoPidio_FND300(t *testing.T) {
	root := proyectoConCola(t)
	t.Setenv("CLAUDE_CODE_SESSION_ID", sesionVieja)
	p := mustPedirEnOrden(t, root, "./desplegar.sh", Orden{})
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	mustPedirEnOrden(t, root, "./desplegar.sh", Orden{})
	if s := pidioDe(t, root, p.ID); s != sesionVieja {
		t.Fatalf("desde la terminal, sin sesión, se queda la que había: %q", s)
	}
}
