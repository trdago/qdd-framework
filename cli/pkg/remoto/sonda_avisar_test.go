package remoto

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// La sonda prueba también avisar, la Edge Function de los push (sugerencia de qdd-movile, 2026-11-15), con un tópico y
// una firma que no son de nadie: un relay al día responde 401; 404 es que no está y 400 «tipo inválido», que está
// atrasada. Nunca lleva las credenciales de la máquina.

func TestSonda_AvisarAlDiaRespondeQueLasCredencialesNoSonDeNadie(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable", topicoPrueba)
	r := Sondear(context.Background(), relayDe(s), credPrueba(), 3*time.Second)
	if !r.OK || r.Tramo != "" {
		t.Fatalf("avisar responde como un relay al día: %+v", r)
	}
	avisados := s.Avisados()
	if len(avisados) != 1 {
		t.Fatalf("una llamada a avisar por sonda: %v", avisados)
	}
	a := avisados[0]
	if a["topico"] == credPrueba().Topico || a["firma"] == credPrueba().Firma || a["topico"] != topicoSonda || a["firma"] != firmaSonda {
		t.Fatalf("nunca las credenciales de la máquina: %v", a)
	}
	if a["tipo"] != AvisoPregunta || a["proyecto"] != "sonda" || a["clave"] != "sonda" {
		t.Fatalf("tipo pregunta y nada de un proyecto real: %v", a)
	}
}

func TestSonda_AvisarAusenteOAtrasadaTieneSuCodigo(t *testing.T) {
	casos := []struct {
		estado int
		texto  string
		codigo string
	}{
		{http.StatusNotFound, "", MotivoAvisarAusente},
		{http.StatusBadRequest, "tipo inválido: pedido, ficha, termino_bien", MotivoAvisarAtrasado},
		{http.StatusInternalServerError, "relay", MotivoAvisarInesperado},
	}
	for _, c := range casos {
		s := nuevoServidorFalso(t, "publicable", topicoPrueba)
		s.presencias["d1"] = []map[string]any{{"rol": "movil", "dispositivo": "d1", "phx_ref": "r1"}}
		s.Avisar(c.estado, c.texto)
		r := Sondear(context.Background(), relayDe(s), credPrueba(), 3*time.Second)
		if r.OK || r.Tramo != TramoAvisar || r.Codigo != c.codigo || r.Motivo == "" {
			t.Fatalf("%d: %+v", c.estado, r)
		}
		if !r.Eco || r.Mensajes != 2 {
			t.Fatalf("conserva lo que midió del canal: %+v", r)
		}
		if ReparacionDe(r) != "" {
			t.Fatalf("avisar la despliega quien lleva el relay: el PC no repara nada: %+v", r)
		}
		d := diagnosticoDe(&r, time.Now())
		if d == nil || d.Motivo != c.codigo || strings.Contains(d.Motivo, " ") {
			t.Fatalf("al teléfono va el código: %+v", d)
		}
	}
}
