package remoto

import (
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// MEJ-083 (FND-232, propuesta de qdd-movile): el teléfono dibuja el estado de la tarea de un nodo buscándola en los
// frentes del panel, que solo traen las visibles; una tarea cerrada decía «sin estado en el tablero». El nodo trae
// tarea_estado cuando su tarea existe y no está en el tablero visible; con la tarea a la vista, el panel ya lo dice.

func tareaX(t *testing.T, root string, estadoCelda string) {
	t.Helper()
	if err := estado.Set(root, estado.Cambio{Frente: "x", Nombre: "X", Etapa: "Diseño", Estado: estadoCelda, Evidencia: "e", Bloqueo: "¿va?", DeQuien: "persona"}, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestFlujos_UnNodoConSuTareaCerradaTraeSuEstado_MEJ083(t *testing.T) {
	root := proyectoConFlujo(t)
	tareaX(t, root, "ok")
	if err := estado.Cerrar(root, "x", "terminada", false, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	m := esperarMensajesDeFlujo(t, tr, 1)[0]
	te := m.Flujo.Nodos[0].TareaEstado
	if te == nil || te.Estado != estado.EstadoCerrado || len(te.Simbolos) != 2 || te.Simbolos[0] != estado.Hecho {
		t.Fatalf("la tarea cerrada llega con su estado y sus celdas: %+v", te)
	}
	if m.Flujo.Nodos[1].TareaEstado != nil {
		t.Fatalf("un nodo sin tarea no trae tarea_estado: %+v", m.Flujo.Nodos[1].TareaEstado)
	}
}

func TestFlujos_UnNodoConSuTareaEnElTableroNoLaRepite_MEJ083(t *testing.T) {
	root := proyectoConFlujo(t)
	tareaX(t, root, "espera")
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	if te := esperarMensajesDeFlujo(t, tr, 1)[0].Flujo.Nodos[0].TareaEstado; te != nil {
		t.Fatalf("una tarea a la vista la dicen los frentes del panel, al día: %+v", te)
	}
}

func TestFlujos_UnNodoConUnaTareaQueNoExisteNoTraeEstado_MEJ083(t *testing.T) {
	proyectoConFlujo(t)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	if te := esperarMensajesDeFlujo(t, tr, 1)[0].Flujo.Nodos[0].TareaEstado; te != nil {
		t.Fatalf("sin esa tarea no viene el campo y la app sigue diciendo que no está: %+v", te)
	}
}

func TestFlujos_CerrarLaTareaDeUnNodoReenviaElFlujoConSuEstado_MEJ083(t *testing.T) {
	root := proyectoConFlujo(t)
	tareaX(t, root, "espera")
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	primero := esperarMensajesDeFlujo(t, tr, 1)[0]
	if err := estado.Cerrar(root, "x", "terminada", true, time.Now()); err != nil {
		t.Fatal(err)
	}
	segundo := esperarMensajesDeFlujo(t, tr, 2)[1]
	if segundo.Flujo.Huella == primero.Flujo.Huella {
		t.Fatal("al salir del tablero cambia la huella y el flujo se reenvía")
	}
	if te := segundo.Flujo.Nodos[0].TareaEstado; te == nil || te.Estado != estado.EstadoDescartado {
		t.Fatalf("llega descartada: %+v", te)
	}
}
