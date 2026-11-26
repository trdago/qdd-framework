package remoto

import (
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// MEJ-094 (REP-510, 2026-11-21): la IA preguntó con AskUserQuestion y la sesión quedó 3 horas esperando a la persona,
// pero el teléfono decía «trabajando». Claude Code no escribe turn_duration hasta que el turno termina, y un turno
// detenido en una pregunta no termina: el tool_use de AskUserQuestion queda en el transcript antes del diálogo.

func preguntaDeLaIA() map[string]any {
	return turno("assistant", []map[string]any{{"type": "tool_use", "name": "AskUserQuestion"}})
}

func TestTrabajando_UnaPreguntaDeLaIANoEsTrabajarYAlResponderVuelve_MEJ094(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	archivo := escribirTranscript(t, root, "s", turno("user", "hola"), finDeTurno("2026-11-21T10:00:00Z"))
	l, _ := abrirLector(archivo)
	escribirTranscript(t, root, "s", turno("user", "arregla esto"), pedidoAPI("2026-11-21T10:01:00.000Z"), preguntaDeLaIA())
	l.leer()
	if l.trabajando != "" {
		t.Fatalf("detenida en una pregunta, espera a la persona: %q", l.trabajando)
	}
	escribirTranscript(t, root, "s", turno("user", []map[string]any{{"type": "tool_result", "content": "elegiste a"}}), pedidoAPI("2026-11-21T13:01:00.000Z"))
	l.leer()
	if l.trabajando != "2026-11-21T10:01:00.000Z" {
		t.Fatalf("respondida, sigue el mismo turno: la hora de su inicio, para que el contador no se reinicie: %q", l.trabajando)
	}
	escribirTranscript(t, root, "s", finDeTurno("2026-11-21T13:05:00Z"), turno("user", "otra cosa"), pedidoAPI("2026-11-21T14:00:00.000Z"))
	l.leer()
	if l.trabajando != "2026-11-21T14:00:00.000Z" {
		t.Fatalf("un turno nuevo empieza de cero: %q", l.trabajando)
	}
}

// Al abrir el lector (qdd remoto se reinicia mientras la pregunta espera), tampoco trabaja.
func TestTrabajando_AlAbrirConUnaPreguntaAbiertaNoTrabaja_MEJ094(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	archivo := escribirTranscript(t, root, "s", turno("user", "arregla esto"), pedidoAPI("2026-11-21T10:01:00.000Z"), preguntaDeLaIA())
	l, _ := abrirLector(archivo)
	if l.trabajando != "" {
		t.Fatalf("una pregunta abierta no es trabajar: %q", l.trabajando)
	}
}

// Un subagente que pregunta no detiene la sesión principal (no llega a la persona igual).
func TestTrabajando_OtraHerramientaOUnSubagenteSiguenTrabajando_MEJ094(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	sub := preguntaDeLaIA()
	sub["isSidechain"] = true
	archivo := escribirTranscript(t, root, "s", turno("user", "arregla esto"), pedidoAPI("2026-11-21T10:01:00.000Z"), sub,
		turno("assistant", []map[string]any{{"type": "tool_use", "name": "Bash"}}))
	l, _ := abrirLector(archivo)
	if l.trabajando != "2026-11-21T10:01:00.000Z" {
		t.Fatalf("sigue trabajando: %q", l.trabajando)
	}
}

// Una pregunta que solo se responde en el PC igual espera a la persona: pide el push «pregunta», como acordó qdd-movile.
func TestAvisos_UnaPreguntaSoloEnElPCPideElPushPregunta_MEJ094(t *testing.T) {
	a := consola.Accion{ID: "pregunta:1", Tipo: consola.TipoPregunta, Estado: consola.PreguntaPendiente, Origen: consola.OrigenIA, ResponderEn: consola.ResponderEnPC}
	if got := avisoDe(a, false); got != AvisoPregunta {
		t.Fatalf("aviso: %q", got)
	}
}
