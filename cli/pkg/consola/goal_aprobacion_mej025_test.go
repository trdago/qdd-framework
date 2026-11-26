package consola

import (
	"encoding/json"
	"testing"
)

// preguntaConEncabezado registra un AskUserQuestion con ese encabezado y texto, y una opción recomendada.
func preguntaConEncabezado(t *testing.T, root, encabezado, texto string) string {
	t.Helper()
	entrada, _ := json.Marshal(map[string]any{"questions": []map[string]any{{
		"question": texto, "header": encabezado, "multiSelect": false,
		"options": []map[string]string{{"label": "Sí (Recommended)", "description": "d"}, {"label": "No", "description": "d"}},
	}}})
	p, err := RegistrarPregunta(root, NuevaPregunta{Sesion: "s1", Herramienta: herramientaPreguntar, Entrada: entrada}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

// MEJ-025 (issue #147, la persona eligió «Sí, siempre mías», 2026-11-26): el modo goal respondió «¿Apruebas lo que
// propuse en la guía de voz?» y la IA lo dejó escrito como aprobación de la persona. Una pregunta de aprobación
// espera a la persona aunque traiga opción recomendada, igual que una de diseño de un blueprint.
// contrato: goal#16ed2b52
func TestGoal_UnaPreguntaDeAprobacionEsperaALaPersona_MEJ025(t *testing.T) {
	casos := []struct{ encabezado, texto string }{
		{"Aprobación", "¿Publico la guía de voz como quedó?"},
		{"aprobar", "¿Sigo con la guía de voz?"},
		{"Approval", "Ship the voice guide?"},
		{"Guía", "¿Apruebas lo que propuse en la guía de voz?"},
		{"Guía", "  ¿ apruebo   la ficha voz?"},
		{"Guía", "¿Aprobamos el cambio?"},
		{"Guide", "Do you approve the voice guide?"},
	}
	for _, c := range casos {
		root := conGoal(t)
		id := preguntaConEncabezado(t, root, c.encabezado, c.texto)
		r := cicloGoal(t, root, silencio(0), enCola)
		if p, _ := PreguntaDe(root, id, enCola); p.Estado != PreguntaPendiente || len(r.Elegidas) != 0 {
			t.Fatalf("%q / %q es de aprobación: la responde la persona aunque traiga recomendada: %+v %+v", c.encabezado, c.texto, p, r.Elegidas)
		}
		if g := VistaGoal(root, enCola); g.Estado != GoalPausado || g.Motivo != MotivoEsperaPersona {
			t.Fatalf("el modo queda en pausa esperando a la persona: %+v", g)
		}
	}
}

// Lo demás con opción recomendada lo sigue eligiendo el modo: «aprobado» en medio de una frase no la vuelve de aprobación.
func TestGoal_UnaPreguntaComunLaSigueEligiendo_MEJ025(t *testing.T) {
	root := conGoal(t)
	id := preguntaConEncabezado(t, root, "Canal", "¿Mando el aviso cuando el PR quede aprobado?")
	r := cicloGoal(t, root, silencio(0), enCola)
	if p, _ := PreguntaDe(root, id, enCola); p.Estado == PreguntaPendiente || len(r.Elegidas) != 1 {
		t.Fatalf("una pregunta común con recomendada la elige el modo: %+v %+v", p, r.Elegidas)
	}
}
