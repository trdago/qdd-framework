package integration

import (
	"strings"
	"testing"
)

// RFC 0006 §3.9: la IA deja en el panel lo que debe ejecutar la persona, en vez de pedirle que lo escriba.
func TestProtocol_TellsTheAIToLeaveHumanCommandsInThePanel(t *testing.T) {
	for _, debe := range []string{"qdd_consola_pedir", "qdd consola pedir", "never run it yourself"} {
		if !strings.Contains(QDDCommandProtocol, debe) {
			t.Errorf("el bloque de instrucciones para la IA debe mencionar %q", debe)
		}
	}
}
