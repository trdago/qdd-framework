package integration

import (
	"strings"
	"testing"
)

// El bloque que QDD deja en AGENTS.md / CLAUDE.md dice por qué el cero 'else', no solo que existe.
// contrato: metodo#1995b1da
func TestProtocol_CeroElseConSuRazon(t *testing.T) {
	for _, debe := range []string{"zero else", "fewer people read code line by line", "at a glance"} {
		if !strings.Contains(QDDCommandProtocol, debe) {
			t.Errorf("el bloque de instrucciones para la IA debe decir %q", debe)
		}
	}
}
