package integration

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de sfrc, decidido por la persona (2026-11-03): lo que la IA necesita de la persona no se queda en el chat.

func TestProtocolo_LasConsultasALaPersonaNoSeQuedanEnElChat(t *testing.T) {
	for _, debe := range []string{"QUESTIONS FOR THE HUMAN", "AskUserQuestion", "recommended option", "qdd_estado", "¿…?", "qdd_consola_pedir", "qdd estado check"} {
		if !strings.Contains(QDDCommandProtocol, debe) {
			t.Errorf("el bloque de AGENTS.md debe decir %q", debe)
		}
	}
}

// MEJ-021: una sesión larga nunca declaró su tarea porque ninguna regla lo pedía, y el tablero y el teléfono no la mostraban.

func TestProtocolo_LaTareaEnCursoSeDeclara_MEJ021(t *testing.T) {
	for _, debe := range []string{"CURRENT TASK", "accion=activar", "accion=terminar", "qdd estado activa"} {
		if !strings.Contains(QDDCommandProtocol, debe) {
			t.Errorf("el bloque de AGENTS.md debe decir %q", debe)
		}
	}
}
