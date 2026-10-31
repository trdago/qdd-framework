package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/modules"
)

func registerProjectContextTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_project_context",
		mcp.WithDescription("PRIMERA llamada de toda tarea. Devuelve la abstracción del proyecto según QDD: propósito, forma de trabajo obligatoria, cómo se prueba, la ficha completa (objetivo operativo, objetivo tecnológico, invariantes, pruebas, ADRs) de los módulos que vas a tocar, el mapa de módulos y las zonas de código sin ficha."),
		mcp.WithString("paths", mcp.Description("Archivos o carpetas que vas a modificar, separados por comas o saltos de línea (relativos a la raíz del proyecto). Opcional: sin rutas devuelve el mapa general.")),
	)
	RegisterToolWithTelemetry(s, tool, handleProjectContextTool)
}

func handleProjectContextTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	notice := migrationNotice(root)
	modo, visto := modoDeLaIA(root)
	text := modules.BuildContext(modules.ContextInput{
		Root:         root,
		Paths:        splitPaths(stringArg(request, "paths")),
		TestCommands: audit.TestCommandLabels(root),
		Estandares:   blueprints.Estandares(root),
		Modo:         modo,
		ModoVisto:    visto,
	})
	if notice != "" {
		text = notice + "\n" + text
	}
	return mcp.NewToolResultText(text), nil
}

func registerModuleSaveTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_module_save",
		mcp.WithDescription("Crea o actualiza la ficha de un módulo en .qdd/project/modules/<id>.yaml. Se valida contra el esquema QDD y SIEMPRE queda en 'borrador': solo un humano la aprueba (qdd modules approve <id>). Basa el contenido en el código y la documentación existente (docs/, ADRs, README) y cita esas fuentes en 'fuentes'; no inventes objetivos."),
		mcp.WithString("ficha_yaml", mcp.Required(), mcp.Description("Ficha en YAML con: id, nombre, rutas[], objetivo_operativo{problema, usuarios, criterio_de_exito, caso_real}, objetivo_tecnologico{arquitectura, decisiones[], invariantes[]}, pruebas{comandos[], goldensets[]}, adrs[], findings[], fuentes[]. Plantilla: .qdd/core/templates/module.yaml")),
	)
	RegisterToolWithTelemetry(s, tool, handleModuleSaveTool)
}

func handleModuleSaveTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	raw := stringArg(request, "ficha_yaml")
	if strings.TrimSpace(raw) == "" {
		return mcp.NewToolResultError("Falta 'ficha_yaml'. Usa la plantilla .qdd/core/templates/module.yaml."), nil
	}
	m, err := modules.Parse([]byte(raw))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, problems := modules.Save(projectRootForMCP(), m)
	if len(problems) > 0 {
		return mcp.NewToolResultError("Ficha rechazada. Corrige y vuelve a enviarla:\n- " + strings.Join(problems, "\n- ")), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Ficha guardada en %s como BORRADOR. Pide al humano que la revise y la apruebe con: qdd modules approve %s", path, m.ID)), nil
}

// projectRootForMCP resuelve la raíz real del proyecto aunque el servidor MCP
// se haya lanzado desde una subcarpeta.
func projectRootForMCP() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return integration.FindProjectRoot(cwd)
}

func stringArg(request mcp.CallToolRequest, name string) string {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := args[name].(string)
	return value
}

func splitPaths(raw string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
