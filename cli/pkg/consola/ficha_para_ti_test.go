package consola

import (
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-059 (#243): una ficha en borrador tiene que llegar a «▶ Para ti» y al teléfono: el panel la pone ahí solo si
// trae su turno (RFC 0019), y el teléfono la avisa como ficha.

func TestMEJ059_UnaFichaEnBorradorLlevaTurnoEnElPanel(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoConFichas(t, nil, []string{"descubrimiento-compra-agil"})
	a, err := BuscarAccion(Panel(root, time.Now()).Acciones, "aprobar:descubrimiento-compra-agil")
	if err != nil {
		t.Fatal(err)
	}
	if a.Turno == 0 || a.TurnoDesde == nil {
		t.Fatalf("la ficha en borrador espera a la persona: lleva turno y entra a «Para ti»: %+v", a)
	}
}
