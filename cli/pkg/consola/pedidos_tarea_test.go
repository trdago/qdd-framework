package consola

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-10-31): un pedido espera en el panel (p. ej. publicar) y la persona decide seguir
// trabajando en esa tarea en vez de ejecutarlo. Como no se sabe si en el próximo mensaje se publicará, el pedido sale
// de «▶ Para ti» en cuanto la tarea sigue cambiando, y vuelve cuando la IA lo pide de nuevo: es la misma tarea que se
// extiende.

const estadoDosTareas = `formato: 1
etapas: [Código, PR]
frentes:
  - {id: barra, nombre: Barra, estado: abierto, celdas: {Código: {estado: "✅", evidencia: commit a1}, PR: {estado: "⏳", bloqueo: merge, de_quien: la persona}}}
  - {id: otra, nombre: Otra, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: b, de_quien: c}, PR: {estado: "—"}}}
`

func proyectoConTareas(t *testing.T) string {
	t.Helper()
	root := proyectoConCola(t)
	escribirArchivo(t, filepath.Join(root, estado.Ruta), estadoDosTareas)
	return root
}

func cambiarTarea(t *testing.T, root, id, evidencia string) {
	t.Helper()
	if err := estado.Set(root, estado.Cambio{Frente: id, Etapa: "Código", Estado: "✅", Evidencia: evidencia}, enCola); err != nil {
		t.Fatal(err)
	}
}

func pedidoDe(t *testing.T, root, id string, now time.Time) Pedido {
	t.Helper()
	for _, p := range PedidosA(root, now) {
		if p.ID == id {
			return p
		}
	}
	return Pedido{}
}

func TestPedido_SaleDePara_TiSiSuTareaSigueDespuesDeUnMensaje(t *testing.T) {
	root := proyectoConTareas(t)
	p := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "barra"})
	if p.Tarea != "barra" {
		t.Fatalf("el pedido recuerda la tarea que extiende: %+v", p)
	}
	if err := MarcarCiclo(root, "auto", enCola.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	cambiarTarea(t, root, "barra", "commit b2 (la persona pidió seguir)")
	got := pedidoDe(t, root, p.ID, enCola.Add(2*time.Minute))
	if got.Estado != EstadoRetirado {
		t.Fatalf("la tarea siguió después del mensaje: el pedido sale de «Para ti»: %+v", got)
	}
	if TeclaDe(root, p.ID) != "" {
		t.Fatal("y el panel no lo ofrece")
	}
	for _, a := range accionesDePedidos(root) {
		if a.pedido == p.ID {
			t.Fatalf("un retirado no se dibuja: %+v", a)
		}
	}
}

func TestPedido_LoQueCambiaLaIAEnElMismoTurnoNoLoSaca(t *testing.T) {
	root := proyectoConTareas(t)
	p := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "barra"})
	cambiarTarea(t, root, "barra", "commit b2 (anotado junto con el pedido)")
	if got := pedidoDe(t, root, p.ID, enCola.Add(time.Minute)); got.Estado != EstadoPendiente {
		t.Fatalf("sin un mensaje nuevo de la persona, el pedido sigue: %+v", got)
	}
}

func TestPedido_OtraTareaNoLoSaca(t *testing.T) {
	root := proyectoConTareas(t)
	p := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "barra"})
	MarcarCiclo(root, "auto", enCola.Add(time.Minute))
	cambiarTarea(t, root, "otra", "commit c3")
	if got := pedidoDe(t, root, p.ID, enCola.Add(2*time.Minute)); got.Estado != EstadoPendiente {
		t.Fatalf("trabajar en otra tarea no lo saca: %+v", got)
	}
}

func TestPedido_VuelveConSuTeclaCuandoLaIALoPideDeNuevo(t *testing.T) {
	root := proyectoConTareas(t)
	mustPedirEnOrden(t, root, "echo otro", Orden{})
	p := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "barra"})
	MarcarCiclo(root, "auto", enCola.Add(time.Minute))
	cambiarTarea(t, root, "barra", "commit b2")
	if pedidoDe(t, root, p.ID, enCola.Add(2*time.Minute)).Estado != EstadoRetirado {
		t.Fatal("primero sale")
	}
	otra, err := PedirEnOrden(root, "gh pr merge 42 && git push origin v2", "Publicar (con lo nuevo)", "", Orden{Tarea: "barra"}, enCola.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if otra.Tecla != p.Tecla {
		t.Fatalf("vuelve con la misma tecla (%s): %+v", p.Tecla, otra)
	}
	cola := PedidosA(root, enCola.Add(4*time.Minute))
	for _, x := range cola {
		if x.Estado == EstadoRetirado {
			t.Fatalf("el pedido nuevo de la tarea reemplaza al retirado: %+v", cola)
		}
	}
	if len(cola) != 2 {
		t.Fatalf("quedan el otro y el nuevo: %+v", cola)
	}
}

func TestPedido_SinTareaUsaLaTareaEnCurso(t *testing.T) {
	root := proyectoConTareas(t)
	if err := estado.Activar(root, "barra", "agente", time.Now()); err != nil {
		t.Fatal(err)
	}
	p, err := Pedir(root, "gh pr merge 42", "Publicar", "", time.Now())
	if err != nil || p.Tarea != "barra" {
		t.Fatalf("sin --tarea, el pedido extiende la tarea en curso: %+v %v", p, err)
	}
}

func TestPedido_SinEstadoLegibleNoSeRetira(t *testing.T) {
	root := proyectoConTareas(t)
	p := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "barra"})
	MarcarCiclo(root, "auto", enCola.Add(time.Minute))
	escribirArchivo(t, filepath.Join(root, estado.Ruta), "no: [es, yaml")
	if got := pedidoDe(t, root, p.ID, enCola.Add(2*time.Minute)); got.Estado != EstadoPendiente {
		t.Fatalf("si no se puede leer la tarea, no se sabe si cambió: el pedido sigue: %+v", got)
	}
}
