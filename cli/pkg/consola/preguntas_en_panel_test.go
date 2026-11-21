package consola

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-057 (la persona eligió «Arriba con botones», 2026-11-15): una pregunta de la terminal se responde también desde
// «▶ Para ti», con un botón por opción. La respuesta queda en la cola como la del teléfono, con el lugar «el panel», y
// la sesión que la espera la aplica al diálogo de Claude Code.

func TestPreguntaEnPanel_UnaOpcionDeAskUserQuestionLlegaAClaudeCode(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Qué base usamos?")}, now)
	p, err := ResponderPreguntaEnPanel(root, id, "b", now)
	if err != nil {
		t.Fatal(err)
	}
	e := EntregaDe(p)
	if e.Estado != PreguntaRespondida || e.Lugar != LugarPanel || e.Respuestas["¿Qué base usamos?"] != "SQLite" {
		t.Fatalf("la sesión recibe la opción elegida en el panel: %+v", e)
	}
	if a, _ := accionDelPanel(root, id, now.Add(time.Second)); a.Turno != 0 {
		t.Fatalf("respondida en el panel sale de «Para ti»: %+v", a)
	}
}

func TestPreguntaEnPanel_UnPermisoSePermiteUnaVezONiega(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: "Bash", Entrada: json.RawMessage(`{"command":"git push"}`)}, now)
	p, err := ResponderPreguntaEnPanel(root, id, Negar, now)
	if err != nil {
		t.Fatal(err)
	}
	if e := EntregaDe(p); e.Decision != Negar {
		t.Fatalf("el permiso queda negado: %+v", e)
	}
}

func TestPreguntaEnPanel_UnaOpcionQueNoOfreceSeRechazaSinCambiarNada(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Qué base usamos?")}, now)
	if _, err := ResponderPreguntaEnPanel(root, id, "z", now); !errors.Is(err, ErrRespuestaInvalida) {
		t.Fatalf("una opción que no está se rechaza: %v", err)
	}
	if a, _ := accionDelPanel(root, id, now); a.Turno == 0 {
		t.Fatalf("sigue esperando a la persona: %+v", a)
	}
}

func TestPreguntaEnPanel_VariasPreguntasJuntasSeRespondenEnLaTerminal(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	dos := json.RawMessage(`{"questions":[{"question":"¿Uno?","header":"A","options":[{"label":"Sí"},{"label":"No"}]},{"question":"¿Dos?","header":"B","options":[{"label":"Sí"},{"label":"No"}]}]}`)
	id := registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: dos}, now)
	if _, err := ResponderPreguntaEnPanel(root, id, "a", now); !errors.Is(err, ErrVariasPreguntas) {
		t.Fatalf("con varias preguntas el panel no elige por la persona: %v", err)
	}
}

func TestPreguntaEnPanel_YaRespondidaEnLaTerminalDiceDonde(t *testing.T) {
	root := proyectoParaTi(t)
	now := time.Now()
	id := registrada(t, root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: askDePrueba("¿Qué base usamos?")}, now)
	if _, err := CerrarPregunta(root, id, EnPC, "", now); err != nil {
		t.Fatal(err)
	}
	var cerrada *PreguntaCerrada
	if _, err := ResponderPreguntaEnPanel(root, id, "a", now); !errors.As(err, &cerrada) {
		t.Fatalf("ya respondida: lo dice, no la pisa: %v", err)
	}
}
