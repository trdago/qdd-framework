package estado

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-199 (MEJ-065, relay:REP-484): el panel mostraba solo las MaxTareas pendientes más nuevas y escondía tareas del
// objetivo en foco, aunque la persona lo marcó como foco justamente para tenerlas a la vista.

func TestFND199_LasTareasDelFocoSiempreSeVen(t *testing.T) {
	d := DatosPendientes(conObjetivos(), ahora, 1)
	var ids []string
	for _, f := range d.Frentes {
		ids = append(ids, f.ID)
	}
	got := strings.Join(ids, ",")
	if got != "modelo,dibujo,limpiar" {
		t.Fatalf("las tareas del foco se ven siempre y el tope cuenta el resto: got %s, want modelo,dibujo,limpiar", got)
	}
	if d.Ocultos != 2 {
		t.Fatalf("ocultos cuenta solo las que no se ven (panel y herramientas): got %d", d.Ocultos)
	}
}

func TestFND199_SinFocoSigueElTope(t *testing.T) {
	e := conObjetivos()
	e.Objetivos[1].Foco = false
	d := DatosPendientes(e, ahora, 1)
	if len(d.Frentes) != 1 || d.Frentes[0].ID != "limpiar" || d.Ocultos != 4 {
		t.Fatalf("sin foco, el tope deja las más nuevas: %+v ocultos=%d", d.Frentes, d.Ocultos)
	}
}
