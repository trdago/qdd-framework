package modules

import (
	"strings"
	"testing"
)

// La IA que trabaja con QDD tiene que encontrar la razón de fondo del cero 'else' donde lee la regla, no solo en la
// documentación (pedido del usuario, 2026-10-31).
func TestMetodoDeTrabajo_CeroElseDiceElPorQue(t *testing.T) {
	m := MetodoDeTrabajo()
	for _, debe := range []string{"Cero 'else'", "No es una moda", PorQueCeroElse, "cualquier lenguaje"} {
		if !strings.Contains(m, debe) {
			t.Errorf("la forma de trabajo debe decir %q", debe)
		}
	}
}
