package consola

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0017 §1 (MEJ-010): un pedido retirado solo cambiaba de estado en la cola, y la sesión que lo pidió no se enteraba.

func pedidoRetirado(t *testing.T, sesion string) (string, Pedido) {
	t.Helper()
	root := proyectoConTareas(t)
	t.Setenv(sesionClaude, sesion)
	p, err := PedirEnOrden(root, "git push", "Publicar", "", Orden{Tarea: "barra"}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	if err := AnotarHuellas(root, enCola); err != nil {
		t.Fatal(err)
	}
	cambiarTarea(t, root, "barra", "commit a2")
	if got := pedidoDe(t, root, p.ID, enCola).Estado; got != EstadoRetirado {
		t.Fatalf("debía retirarse: %s", got)
	}
	return root, p
}

func TestContextoCiclo_LeAvisaUnaVezALaSesionQuePidioElPedidoRetirado(t *testing.T) {
	root, _ := pedidoRetirado(t, "sesion-a")
	texto := ContextoCiclo(root, "sesion-a", enCola)
	for _, debe := range []string{"retiró", "«Publicar»", "barra", "pídelo de nuevo"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q: %q", debe, texto)
		}
	}
	if otra := ContextoCiclo(root, "sesion-a", enCola); strings.Contains(otra, "retiró") {
		t.Fatalf("se avisa una sola vez: %q", otra)
	}
}

func TestContextoCiclo_OtraSesionNoConsumeElAvisoDelRetirado(t *testing.T) {
	root, _ := pedidoRetirado(t, "sesion-a")
	if texto := ContextoCiclo(root, "sesion-b", enCola); strings.Contains(texto, "retiró") {
		t.Fatalf("no es su pedido: %q", texto)
	}
	if texto := ContextoCiclo(root, "sesion-a", enCola); !strings.Contains(texto, "retiró") {
		t.Fatalf("sigue pendiente para quien lo pidió: %q", texto)
	}
}

func TestContextoCiclo_SinSesionNoAvisaDeNingunRetirado(t *testing.T) {
	root, _ := pedidoRetirado(t, "")
	for _, sesion := range []string{"", "sesion-a"} {
		if texto := ContextoCiclo(root, sesion, enCola); strings.Contains(texto, "retiró") {
			t.Fatalf("no se sabe quién lo pidió (%q): %q", sesion, texto)
		}
	}
	if pedidos := PedidosA(root, enCola); len(pedidos) != 1 || pedidos[0].Estado != EstadoRetirado {
		t.Fatalf("sigue visible en la cola: %+v", pedidos)
	}
}

func TestContextoCiclo_ElAvisoDelRetiradoSaleAunqueLasEsperasNoCambien(t *testing.T) {
	root, _ := pedidoRetirado(t, "sesion-a")
	ContextoCiclo(root, "sesion-b", enCola)
	if texto := ContextoCiclo(root, "sesion-a", enCola); !strings.Contains(texto, "retiró") {
		t.Fatalf("las esperas repetidas no lo callan: %q", texto)
	}
}
