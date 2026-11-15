package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'

// RFC 0017 §2: el aviso de commits nuevos en las rutas de la ficha de una tarea no bloquea y dice cuándo se limitó al foco.
func TestAvisarCommitsEnTareas_NombraLaTareaYSuFicha(t *testing.T) {
	var salida bytes.Buffer
	avisarCommitsEnTareas(&salida, estado.CommitsEnTareas{})
	if salida.Len() != 0 {
		t.Fatalf("sin commits nuevos, no dice nada: %q", salida.String())
	}
	avisarCommitsEnTareas(&salida, estado.CommitsEnTareas{Tareas: []estado.TareaConCommits{{Tarea: "cobro", Modulo: "pagos", Commits: 3, Desde: "2026-11-10T12:00:00Z"}}, SoloFoco: true})
	texto := salida.String()
	for _, debe := range []string{"[i]", "tarea cobro: 3 commits en las rutas de su ficha pagos", "foco"} {
		if !strings.Contains(texto, debe) {
			t.Fatalf("debe decir %q: %q", debe, texto)
		}
	}
}

func TestCambioTareaDesde_ModuloDelMCP(t *testing.T) {
	pedir := func(args map[string]any) estado.CambioTarea {
		var r mcp.CallToolRequest
		r.Params.Arguments = args
		return cambioTareaDesde(r)
	}
	if c := pedir(map[string]any{"frente": "a"}); c.CambiaModulo {
		t.Fatal("sin modulo no lo cambia")
	}
	if c := pedir(map[string]any{"frente": "a", "modulo": "pagos"}); !c.CambiaModulo || c.Modulo != "pagos" {
		t.Fatalf("modulo lo declara: %+v", c)
	}
	if c := pedir(map[string]any{"frente": "a", "modulo": "-"}); !c.CambiaModulo || c.Modulo != "" {
		t.Fatalf("«-» lo quita: %+v", c)
	}
}
