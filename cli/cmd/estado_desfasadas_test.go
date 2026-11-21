package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

func TestAvisarDesfasadas_NombraLasTareasSinBloquear_MEJ010(t *testing.T) {
	var salida bytes.Buffer
	avisarDesfasadas(&salida, estado.TareasDesfasadas{})
	if salida.Len() != 0 {
		t.Fatalf("sin desfases, no dice nada: %q", salida.String())
	}
	avisarDesfasadas(&salida, estado.TareasDesfasadas{Listas: []string{"lista"}, CerradasConEspera: []string{"cerrada"}})
	texto := salida.String()
	for _, debe := range []string{"[i]", "lista", "qdd estado cerrar", "cerrada", "qdd estado nueva"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q: %q", debe, texto)
		}
	}
}
