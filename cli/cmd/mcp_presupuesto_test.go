package cmd

import (
	"encoding/json"
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
