package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'

func TestAvisarEsperasSinPregunta_NombraCadaUnaYDiceComoArreglarla(t *testing.T) {
	var salida bytes.Buffer
	avisarEsperasSinPregunta(&salida, nil)
	if salida.Len() != 0 {
		t.Fatalf("sin esperas, no dice nada: %q", salida.String())
	}
	avisarEsperasSinPregunta(&salida, []consola.EsperaSinPregunta{{Tarea: "Barra", Etapa: "PR", Bloqueo: "falta tu visto bueno"}})
	texto := salida.String()
	for _, debe := range []string{"[i]", "Barra · PR", "falta tu visto bueno", "¿…?", "[tecla]"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q: %q", debe, texto)
		}
	}
}

// FND-176: un ⏳ que espera un pedido ya hecho no llega al teléfono; el check le pide a la IA reflejarlo.
func TestAvisarEsperasTrasPedido_PideActualizarElEstado_FND176(t *testing.T) {
	var salida bytes.Buffer
	avisarEsperasTrasPedido(&salida, nil)
	if salida.Len() != 0 {
		t.Fatalf("sin esperas, no dice nada: %q", salida.String())
	}
	avisarEsperasTrasPedido(&salida, []consola.EsperaSinPregunta{{Tarea: "Barra", Etapa: "PR", Bloqueo: "¿Confirmas [a]?"}})
	texto := salida.String()
	for _, debe := range []string{"[i]", "Barra · PR", "¿Confirmas [a]?", "ya terminó", "qdd_estado set"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q: %q", debe, texto)
		}
	}
}
