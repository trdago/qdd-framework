package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// Tool MCP qdd_consola_pedir (RFC 0006 §3.9): la IA deja en el panel un comando que debe ejecutar la persona. QDD no lo
// ejecuta: corre solo cuando la persona lo confirma en el panel.

func registerConsolaTools(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_consola_pedir",
		mcp.WithDescription("Deja un comando en el panel de estado de la persona para que lo ejecute con un clic. Úsalo cuando un comando lo tiene que ejecutar la persona: no tienes permiso para correrlo, te lo bloquearon, o publica algo (git push de un tag, npm publish, gh pr merge…). En vez de pedirle que lo escriba con '!', déjalo aquí con su motivo. QDD NO lo ejecuta: corre solo cuando la persona lo confirma en el panel, que muestra el motivo, el comando exacto y la carpeta. Nunca lo ejecutes tú con 'qdd consola accion'."),
		mcp.WithString("comando", mcp.Required(), mcp.Description("El comando exacto, como lo escribiría la persona en su shell (puede usar && o |).")),
		mcp.WithString("motivo", mcp.Required(), mcp.Description("Por qué lo tiene que ejecutar la persona y qué hace; se lee en el panel antes de confirmar.")),
		mcp.WithString("carpeta", mcp.Description("Dónde corre; por defecto, la raíz del proyecto.")),
	)
	RegisterToolWithTelemetry(s, tool, handleConsolaPedirTool)
}

func handleConsolaPedirTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	p, err := consola.Pedir(root, stringArg(request, "comando"), stringArg(request, "motivo"), strings.TrimSpace(stringArg(request, "carpeta")), time.Now())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Quedó en el panel de estado con la tecla %s: «%s» → %s (en %s).\nNo lo ejecutes tú: dile a la persona que lo ejecute desde el panel (clic en el panel, tecla %s y «s  Sí»). Si termina bien sale de la cola; puedes revisar con 'qdd consola pedidos'.", consola.TeclaDe(root, p.ID), p.Motivo, p.Comando, p.Carpeta, consola.TeclaDe(root, p.ID))), nil
}
