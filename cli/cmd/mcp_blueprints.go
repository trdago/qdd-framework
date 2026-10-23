package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/blueprints"
)

func registerBlueprintTools(s *server.MCPServer) {
	list := mcp.NewTool("qdd_blueprint_list",
		mcp.WithDescription("Lista los blueprints importados al proyecto (arquetipos de módulo de la organización: usuarios y roles, etc.) con su vocabulario, sus preguntas de diseño y sus invariantes. Úsala antes de construir un módulo que un blueprint ya define: el blueprint manda sobre tus preferencias."),
		mcp.WithString("blueprint", mcp.Description("Id de un blueprint para ver solo ese (opcional).")),
	)
	RegisterToolWithTelemetry(s, list, handleBlueprintListTool)

	apply := mcp.NewTool("qdd_blueprint_apply",
		mcp.WithDescription("Genera la ficha de un módulo desde un blueprint importado: decisiones, invariantes, escenarios de conformidad y esqueleto. La ficha queda en BORRADOR (la aprueba un humano). Las preguntas del blueprint son decisiones del HUMANO: pregúntaselas tal cual y no las respondas tú. Si faltan respuestas, la tool las devuelve sin escribir nada."),
		mcp.WithString("blueprint", mcp.Required(), mcp.Description("Id del blueprint importado.")),
		mcp.WithString("respuestas", mcp.Description("Respuestas del humano, una por línea o separadas por comas, como pregunta=valor.")),
		mcp.WithString("modulo", mcp.Description("Id de la ficha a generar (por defecto, el id del blueprint).")),
		mcp.WithString("rutas", mcp.Description("Carpetas del módulo, separadas por comas (por defecto, las que sugiere el blueprint).")),
		mcp.WithString("pruebas", mcp.Description("Comandos que prueban el módulo, uno por línea.")),
	)
	RegisterToolWithTelemetry(s, apply, handleBlueprintApplyTool)
}

func handleBlueprintListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	if id := strings.TrimSpace(stringArg(request, "blueprint")); id != "" {
		b, err := blueprints.Find(root, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(blueprints.Describe(b)), nil
	}
	bps, errs := blueprints.List(root)
	var s strings.Builder
	for _, err := range errs {
		fmt.Fprintf(&s, "⚠️ Blueprint ilegible: %v\n", err)
	}
	if len(bps) == 0 {
		s.WriteString("No hay blueprints importados. El humano los importa con: qdd blueprint add <carpeta | repo//subcarpeta> --ref vX.Y.Z\n")
	}
	for _, b := range bps {
		s.WriteString(blueprints.Describe(b) + "\n")
	}
	for _, p := range blueprints.Check(root) {
		fmt.Fprintf(&s, "⚠️ %s: %s\n", p.Regla, p.Descripcion)
	}
	return mcp.NewToolResultText(s.String()), nil
}

func handleBlueprintApplyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	respuestas, err := parseRespuestas(splitPaths(stringArg(request, "respuestas")))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	res, problems := blueprints.Apply(blueprints.ApplyInput{
		Root:       projectRootForMCP(),
		Blueprint:  strings.TrimSpace(stringArg(request, "blueprint")),
		Modulo:     strings.TrimSpace(stringArg(request, "modulo")),
		Rutas:      splitPaths(stringArg(request, "rutas")),
		Respuestas: respuestas,
		Pruebas:    splitLines(stringArg(request, "pruebas")),
	})
	if len(problems) > 0 {
		return mcp.NewToolResultError("No se aplicó el blueprint:\n- " + strings.Join(problems, "\n- ")), nil
	}
	if len(res.Pendientes) > 0 {
		var s strings.Builder
		s.WriteString("Faltan decisiones del humano; no se escribió nada. Hazle estas preguntas (no las respondas tú) y vuelve a llamar con 'respuestas':\n")
		for _, p := range res.Pendientes {
			s.WriteString(blueprints.DescribePregunta(p))
		}
		return mcp.NewToolResultText(s.String()), nil
	}
	msg := fmt.Sprintf("Ficha generada en %s como BORRADOR.\nEscenarios de conformidad: %s (impleméntalos como pruebas del proyecto).\nDecisiones registradas en %s.\n", res.Ficha, res.Conformidad, res.Aplicacion)
	for _, e := range res.Esqueletos {
		msg += fmt.Sprintf("Esqueleto creado en %s/README.md.\n", e)
	}
	return mcp.NewToolResultText(msg + "Pide al humano que revise la ficha y la apruebe con: qdd modules approve <id>"), nil
}

func splitLines(raw string) []string {
	var out []string
	for _, l := range strings.Split(raw, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}
