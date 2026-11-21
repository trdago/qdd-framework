package cmd

import (
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// FLU-02 (RFC 0025, FLC-04): un flujo editado a mano queda con una revisión que su historial no tiene; doctor lo dice y
// --fix la anota como «edición sin registro».

func TestDoctorFlujos_HistorialAlDiaYSuReparacion(t *testing.T) {
	root := proyectoConFlujo(t)
	if item := itemDeGrupo(buildFlujosChecks(root), itemFlujosHistorial); item == nil || !item.Success {
		t.Fatalf("recién creado, el historial cuadra: %+v", item)
	}
	f, _ := flujo.Cargar(root, "pago")
	f.Revision = 9
	flujo.Guardar(root, f)
	g := buildFlujosChecks(root)
	item := itemDeGrupo(g, itemFlujosHistorial)
	if item == nil || item.Success || !strings.Contains(item.Error, "pago") {
		t.Fatalf("una revisión sin historial se dice: %+v", item)
	}
	repairFlujos(root, g)
	if item := itemDeGrupo(buildFlujosChecks(root), itemFlujosHistorial); !item.Success {
		t.Fatalf("--fix la anota: %+v", item)
	}
}

func itemDeGrupo(g *CheckGroup, nombre string) *CheckItem {
	for _, it := range g.Items {
		if it.Name == nombre {
			return it
		}
	}
	return nil
}
