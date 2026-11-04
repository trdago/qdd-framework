package consola

import (
	"os"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-125 (MEJ-008, sfrc, 2026-11-03): un pedido armado se ejecutó dos veces: dos workflow_dispatch en el mismo segundo.
// Dos «qdd consola correr» esperaban el mismo pedido y despertaban a la misma hora (Proximo). El primero lo pasaba a
// ejecutando; el segundo, al verlo ejecutando, también lo corría. Un pedido tiene un solo dueño vivo: quien ya no lo es,
// termina sin correrlo.

func pedidoConDueno(t *testing.T, estadoPedido string, pid int) (string, string) {
	t.Helper()
	root := proyectoConCola(t)
	p, err := Pedir(root, "true", "Desplegar", root, enCola)
	if err != nil {
		t.Fatal(err)
	}
	vencido := enCola.Add(-time.Minute)
	err = conCola(root, enCola, func(cola []Pedido) ([]Pedido, error) {
		i := indiceID(cola, p.ID)
		cola[i].Estado, cola[i].PID, cola[i].Hijo, cola[i].ArmadoEn, cola[i].Proximo = estadoPedido, pid, -1, &enCola, &vencido
		return cola, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return root, p.ID
}

func TestTurno_ConElPedidoEjecutandoEnOtroProcesoNoLoCorre_FND125(t *testing.T) {
	otro := os.Getppid()
	root, id := pedidoConDueno(t, EstadoEjecutando, otro)
	turno, _, err := tomarTurno(root, id, enCola, false)
	if err != nil || turno != turnoFin {
		t.Fatalf("otro proceso ya lo corre: este termina sin correrlo: %q %v", turno, err)
	}
	if p := PedidosA(root, enCola)[0]; p.PID != otro {
		t.Fatalf("el pedido sigue a nombre de quien lo corre: %d", p.PID)
	}
}

func TestTurno_ConElPedidoArmadoPorOtroProcesoVivoNoLoCorre_FND125(t *testing.T) {
	otro := os.Getppid()
	root, id := pedidoConDueno(t, EstadoArmado, otro)
	turno, _, err := tomarTurno(root, id, enCola, false)
	if err != nil || turno != turnoFin {
		t.Fatalf("otro proceso vivo lo espera: este termina: %q %v", turno, err)
	}
	if p := PedidosA(root, enCola)[0]; p.PID != otro || p.Estado != EstadoArmado {
		t.Fatalf("no le quita el pedido al otro: %+v", p)
	}
}

func TestTurno_SuDuenoLoCorreUnaSolaVezAunqueOtroDespierteJunto_FND125(t *testing.T) {
	root, id := pedidoConDueno(t, EstadoArmado, os.Getpid())
	turno, _, err := tomarTurno(root, id, enCola, false)
	if err != nil || turno != turnoCorrer {
		t.Fatalf("a su dueño le toca correrlo: %q %v", turno, err)
	}
	err = conCola(root, enCola, func(cola []Pedido) ([]Pedido, error) {
		cola[indiceID(cola, id)].PID = os.Getppid()
		return cola, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if turno, _, err := tomarTurno(root, id, enCola, false); err != nil || turno != turnoFin {
		t.Fatalf("el que despierta en el mismo instante no lo corre de nuevo: %q %v", turno, err)
	}
}

func TestTurno_UnArmadoDeUnProcesoMuertoVuelveAPendienteSinCorrer_FND125(t *testing.T) {
	root, id := pedidoConDueno(t, EstadoArmado, 1<<30)
	if turno, _, err := tomarTurno(root, id, enCola, false); err != nil || turno != turnoFin {
		t.Fatalf("su dueño murió: nadie lo corre sin que la persona lo confirme de nuevo: %q %v", turno, err)
	}
	if p := PedidosA(root, enCola)[0]; p.Estado != EstadoPendiente {
		t.Fatalf("vuelve a pendiente: %+v", p)
	}
}
