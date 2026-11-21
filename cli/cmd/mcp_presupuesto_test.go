package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/server"
)

// Las tools de qdd se cargan en el contexto de cada sesión: sus descripciones son un costo fijo en tokens y se
// escriben como un chat, cortas y con lo que la IA no puede adivinar (pedido del usuario, 2026-10-31).
const (
	maxDescripcionTool  = 400
	maxDescripcionParam = 260
	maxToolsMCP         = 16000 // caracteres de tools/list; eran 23.815 y ~4.500 son la estructura JSON de las tools
)

func TestMCP_DescripcionesDentroDelPresupuesto(t *testing.T) {
	s := server.NewMCPServer("presupuesto", "1")
	registerMCPTools(s)
	total := 0
	for nombre, st := range s.ListTools() {
		b, err := json.Marshal(st.Tool)
		if err != nil {
			t.Fatal(err)
		}
		total += utf8.RuneCount(b)
		if n := utf8.RuneCountInString(st.Tool.Description); n > maxDescripcionTool {
			t.Errorf("%s: descripción de %d caracteres (tope %d)", nombre, n, maxDescripcionTool)
		}
		for p, v := range st.Tool.InputSchema.Properties {
			m, _ := v.(map[string]interface{})
			d, _ := m["description"].(string)
			if n := utf8.RuneCountInString(d); n > maxDescripcionParam {
				t.Errorf("%s.%s: descripción de %d caracteres (tope %d)", nombre, p, n, maxDescripcionParam)
			}
		}
	}
	if total > maxToolsMCP {
		t.Errorf("tools/list mide %d caracteres (tope %d)", total, maxToolsMCP)
	}
}

func TestMCP_AnotacionesVerdaderasYSinRelleno(t *testing.T) {
	s := server.NewMCPServer("anotaciones", "1")
	registerMCPTools(s)
	tools := s.ListTools()
	for _, nombre := range []string{"qdd_project_context", "qdd_blueprint_list", "qdd_findings", "qdd_score"} {
		a := tools[nombre].Tool.Annotations
		if a.ReadOnlyHint == nil || !*a.ReadOnlyHint {
			t.Errorf("%s solo lee: readOnlyHint", nombre)
		}
	}
	for nombre, st := range tools {
		if st.Tool.Annotations.DestructiveHint != nil {
			t.Errorf("%s repite el valor por defecto del protocolo (destructiveHint): es relleno en cada sesión", nombre)
		}
	}
}

// Las reglas de seguridad que cada descripción tiene que conservar aunque se acorte: quién responde, quién confirma y
// qué no ejecuta la IA. Una frase por regla, tal como la lee la IA en tools/list.
var reglasDeSeguridadMCP = map[string][]string{
	"qdd_blueprint_apply":   {"preguntas son del HUMANO", "nunca las respondas tú", "la aprueba un humano"},
	"qdd_blueprint_propose": {"SOLO con confirmación explícita del humano", "no lo elijas tú"},
	"qdd_consola_pedir":     {"Corre solo si confirma", "nunca lo ejecutes tú"},
	"qdd_module_save":       {"SIEMPRE queda en borrador", "la aprueba un humano"},
	"qdd_blueprint_add":     {"solo los que pidió el humano", "nunca la inventes"},
}

// reglasDeSeguridadParam son las de un parámetro: lo que la IA solo pasa si el humano lo pidió.
var reglasDeSeguridadParam = map[string]map[string]string{
	"qdd_blueprint_add":        {"origen": "SOLO si el humano lo indicó", "reemplazar": "SOLO si el humano lo pidió"},
	"qdd_blueprint_contribute": {"origen": "SOLO si el humano la indicó"},
	"qdd_blueprint_apply":      {"respuestas": "Del humano"},
}

// contrato: mcp#904a6912
func TestMCP_DescripcionesConservanLasReglasDeSeguridad(t *testing.T) {
	s := server.NewMCPServer("reglas", "1")
	registerMCPTools(s)
	tools := s.ListTools()
	for nombre, frases := range reglasDeSeguridadMCP {
		st, ok := tools[nombre]
		if !ok {
			t.Errorf("%s ya no está registrada: revisa dónde quedó su regla de seguridad", nombre)
			continue
		}
		for _, f := range frases {
			if !strings.Contains(st.Tool.Description, f) {
				t.Errorf("%s perdió la regla «%s»: %q", nombre, f, st.Tool.Description)
			}
		}
	}
	for nombre, params := range reglasDeSeguridadParam {
		for p, f := range params {
			m, _ := tools[nombre].Tool.InputSchema.Properties[p].(map[string]interface{})
			if d, _ := m["description"].(string); !strings.Contains(d, f) {
				t.Errorf("%s.%s perdió la regla «%s»: %q", nombre, p, f, d)
			}
		}
	}
}
