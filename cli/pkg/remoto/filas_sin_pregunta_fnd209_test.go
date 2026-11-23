package remoto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// FND-209 (acordado con qdd-movile, 2026-11-22): el estado completo para el teléfono pesaba 184 KB de 256 KB. La
// pregunta de cada fila (lo que el panel del PC envía a la IA al pinchar una tarea) no la muestra ninguna pantalla del
// teléfono: no viaja, ni en frentes ni en ocultas_tuyas.

func TestTablero_LasFilasVanAlTelefonoSinPregunta_FND209(t *testing.T) {
	fila := estado.FrentePanel{ID: "a", Nombre: "Publicar", Pregunta: "¿Cómo vamos con «Publicar»? Avance, bloqueos y siguiente paso."}
	d := estado.DatosPanel{Frentes: []estado.FrentePanel{fila}, OcultasTuyas: []estado.FrentePanel{fila}}
	ocultarTablero(&Ocultador{}, &d)
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "pregunta") || !strings.Contains(string(b), "Publicar") {
		t.Fatalf("las filas van sin su pregunta y con su nombre: %s", b)
	}
}
