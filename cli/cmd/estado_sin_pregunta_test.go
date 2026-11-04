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
