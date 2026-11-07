package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Botones de los ⏳ (2026-11-07): el panel del PC responde un ⏳ de «Para ti» con 'qdd consola pregunta responder <id>
// <opción>' e imprime qué quedó y el mensaje que le envía a la IA.

func TestConsolaPregunta_ResponderUnEsperaDesdeElPanel(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	now := time.Now()
	if err := estado.Init(root, []string{"Código"}, now); err != nil {
		t.Fatal(err)
	}
	if err := estado.Set(root, estado.Cambio{Frente: "t1", Nombre: "Barra", Etapa: "Código", Estado: "⏳", Bloqueo: "¿mergeamos el #8?", DeQuien: "la persona"}, now); err != nil {
		t.Fatal(err)
	}
	esperas := consola.EsperasDelPanel(root, now)
	if len(esperas) != 1 {
		t.Fatalf("el ⏳ está en «Para ti»: %+v", esperas)
	}
	var salida, errores bytes.Buffer
	if codigo := preguntaResponder(root, esperas[0].Pregunta, "quizas", &salida, &errores); codigo != 1 || !strings.Contains(errores.String(), "quizas") {
		t.Fatalf("una opción que no existe se rechaza: %d %s", codigo, errores.String())
	}
	errores.Reset()
	if codigo := preguntaResponder(root, esperas[0].Pregunta, consola.EsperaHecho, &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var r struct{ Aplicado, Mensaje string }
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		t.Fatalf("%v: %s", err, salida.String())
	}
	if r.Aplicado == "" || !strings.Contains(r.Mensaje, "Ya lo hice") {
		t.Fatalf("dice qué quedó y lo que se le envía a la IA: %+v", r)
	}
	if got := consola.EsperasDelPanel(root, now); len(got) != 0 {
		t.Fatalf("sale de «Para ti»: %+v", got)
	}
}
