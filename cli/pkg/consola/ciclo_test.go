package consola

import (
	"io"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9 (pedidos del usuario, 2026-10-30): cada mensaje de la persona es un ciclo. Un pedido hecho que la
// persona ya vio sale del panel en el ciclo siguiente (no queda ensuciando el tablero), y el ciclo anota en qué modo
// de permisos está la IA principal para que qdd_project_context le diga cuándo pedir autorización.

func TestCiclo_UnPedidoHechoSaleEnElCicloSiguiente(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "Mergear #38", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	if codigo := a.Ejecutar(root, "qdd", io.Discard); codigo != 0 {
		t.Fatalf("corre: %d", codigo)
	}
	if cola := Pedidos(root); len(cola) != 1 || cola[0].Estado != EstadoHecho {
		t.Fatalf("recién hecho, se ve en gris: %+v", cola)
	}
	if err := MarcarCiclo(root, "auto", time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if cola := Pedidos(root); len(cola) != 0 {
		t.Fatalf("en el ciclo siguiente sale del tablero: %+v", cola)
	}
}

func TestCiclo_LoAbiertoNoSaleYUnFalloTampoco(t *testing.T) {
	root := proyectoConCola(t)
	mustPedir(t, root, "exit 0 # pendiente", "m", "")
	falla := mustPedir(t, root, "exit 3", "m", "")
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+falla.ID)
	a.Ejecutar(root, "qdd", io.Discard)
	MarcarCiclo(root, "default", time.Now().Add(time.Second))
	if cola := Pedidos(root); len(cola) != 2 {
		t.Fatalf("lo pendiente y lo que falló siguen: %+v", cola)
	}
}

func TestCiclo_AnotaElModoDeLaIA(t *testing.T) {
	root := proyectoConCola(t)
	ahora := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	if _, ok := ModoVigente(root, ahora); ok {
		t.Fatal("sin ciclos, el modo está sin medir")
	}
	MarcarCiclo(root, "auto", ahora)
	if c, ok := ModoVigente(root, ahora.Add(time.Hour)); !ok || c.Modo != "auto" {
		t.Fatalf("el modo del último mensaje: %+v", c)
	}
	if _, ok := ModoVigente(root, ahora.Add(VidaModo+time.Minute)); ok {
		t.Fatal("un modo de hace más de VidaModo ya no se sabe")
	}
	MarcarCiclo(root, "inventado; rm -rf", ahora)
	if _, ok := ModoVigente(root, ahora); ok {
		t.Fatal("un modo que no es de Claude Code no se anota")
	}
}
