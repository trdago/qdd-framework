package modules

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-087 (REP-503, propuesta de la persona): «quien reporta es responsable de validar que se solucionó por completo»
// es parte de la forma de trabajo, junto a «todo bug es un finding con su prueba».

func TestMetodo_QuienReportaValida_MEJ087(t *testing.T) {
	m := MetodoDeTrabajo()
	for _, debe := range []string{"Quien reporta valida", "--verifico", "--verifique", "arreglada"} {
		if !strings.Contains(m, debe) {
			t.Fatalf("la forma de trabajo dice %q:\n%s", debe, m)
		}
	}
}
