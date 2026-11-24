package consola

import (
	"bytes"
	"testing"
)

// Rule: Prohibido usar 'else'

// FND-317: el aviso de un pedido que falla dice «déjalo de nuevo en el panel», y dejarlo de nuevo no lo sacaba del fallo.
func TestPedir_DeNuevoUnoQueFalloVuelveAPendiente_FND317(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedirEnOrden(t, root, "exit 3", Orden{DePersona: true})
	sigue := mustPedirEnOrden(t, root, "echo sigue", Orden{Despues: []string{p.ID}})
	fallar(t, root, p.ID)

	otra := mustPedirEnOrden(t, root, "exit 3", Orden{Reactiva: true})
	if otra.ID != p.ID || otra.Tecla != p.Tecla || otra.Estado != EstadoPendiente {
		t.Fatalf("vuelve a pendiente con su id y su tecla: %+v", otra)
	}
	if otra.Codigo != 0 || otra.Salida != "" || otra.Fin != nil || otra.Inicio != nil || otra.Confirmado != nil || otra.Sesion != "" || otra.Avisado {
		t.Errorf("no arrastra el resultado ni la confirmación anterior: %+v", otra)
	}
	if !otra.DePersona {
		t.Error("perdió la marca de que espera a la persona: el modo goal lo correría")
	}
	for _, q := range Pedidos(root) {
		if q.ID == sigue.ID && (len(q.Despues) != 1 || q.Despues[0] != p.ID) {
			t.Errorf("lo que esperaba por él quedó suelto: %+v", q)
		}
	}
}

func TestPedir_DeNuevoUnoPendienteNoCambia_FND317(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedirEnOrden(t, root, "echo a", Orden{})
	if otra := mustPedirEnOrden(t, root, "echo a", Orden{Reactiva: true}); otra.ID != p.ID || otra.Estado != EstadoPendiente {
		t.Errorf("el que sigue pendiente se devuelve igual: %+v", otra)
	}
	if n := len(Pedidos(root)); n != 1 {
		t.Errorf("hay %d pedidos, se esperaba 1", n)
	}
}

// Un ciclo automático vuelve a pedir lo suyo en cada pasada: con el modo goal repetiría el fallo sin fin.
func TestPedir_UnCicloAutomaticoNoReactivaElQueFallo_FND317(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedirEnOrden(t, root, "exit 3", Orden{})
	fallar(t, root, p.ID)
	if otra := mustPedirEnOrden(t, root, "exit 3", Orden{}); otra.Estado != EstadoFallo || otra.Codigo != 3 {
		t.Errorf("lo reactivó un ciclo automático: %+v", otra)
	}
}

func fallar(t *testing.T, root, id string) {
	t.Helper()
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+id)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo == 0 {
		t.Fatalf("el pedido tenía que fallar: %s", salida.String())
	}
}
