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
		mcp.WithDescription("Deja un comando en el panel de estado de la persona para que lo ejecute con un clic. Úsalo cuando un comando lo tiene que ejecutar la persona: no tienes permiso para correrlo, te lo bloquearon, o publica algo (git push de un tag, npm publish, gh pr merge…). En vez de pedirle que lo escriba con '!', déjalo aquí con su motivo. QDD NO lo ejecuta: corre solo cuando la persona lo confirma en el panel, que muestra el motivo, el comando exacto y la carpeta. Nunca lo ejecutes tú con 'qdd consola accion'. El comando tiene que poder repetirse (revisa lo ya hecho: un tag que ya existe se reusa, un PR mergeado no se vuelve a mergear) y no esperar adentro: si falta algo (un CI, un tag), sale con 'exit 75' y la persona lo vuelve a apretar. Si la persona prefiere seguir trabajando en la tarea en vez de ejecutarlo, el pedido sale solo del panel cuando la tarea cambia; cuando vuelva a estar lista, pídelo de nuevo con la misma tarea."),
		mcp.WithString("comando", mcp.Required(), mcp.Description("El comando exacto, como lo escribiría la persona en su shell (puede usar && o |).")),
		mcp.WithString("motivo", mcp.Required(), mcp.Description("Por qué lo tiene que ejecutar la persona y qué hace; se lee en el panel antes de confirmar.")),
		mcp.WithString("carpeta", mcp.Description("Dónde corre; por defecto, la raíz del proyecto.")),
		mcp.WithString("tarea", mcp.Description("Id de la tarea del estado que el pedido extiende (por defecto, la tarea en curso). Si después de un mensaje de la persona la tarea sigue cambiando, el pedido sale del panel hasta que lo pidas de nuevo con la misma tarea.")),
		mcp.WithString("despues", mcp.Description("Si depende de otros pedidos: sus ids o teclas, separados por coma. El panel lo muestra después de ellos y no lo deja ejecutar hasta que queden hechos. Úsalo siempre que el orden importe (un merge después de un tag, un release después de un merge).")),
		mcp.WithString("antes", mcp.Description("Pedidos que ya están en la cola y tienen que esperar a este: sus ids o teclas, separados por coma. Pedir de nuevo un comando que ya está, con despues o antes, lo reordena sin duplicarlo.")),
	)
	RegisterToolWithTelemetry(s, tool, handleConsolaPedirTool)
}

func handleConsolaPedirTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	orden := consola.Orden{Despues: listaDeComas(stringArg(request, "despues")), Antes: listaDeComas(stringArg(request, "antes")), Tarea: strings.TrimSpace(stringArg(request, "tarea"))}
	p, err := consola.PedirEnOrden(root, stringArg(request, "comando"), stringArg(request, "motivo"), strings.TrimSpace(stringArg(request, "carpeta")), orden, time.Now())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	cola := consola.ColaDe(root, p.Carpeta)
	return mcp.NewToolResultText(fmt.Sprintf("Quedó en %s con la tecla %s: «%s»%s → %s (en %s).\nNo lo ejecutes tú: dile a la persona que lo ejecute desde ese panel (tecla %s y «s  Sí»). Puedes ver si quedó hecho o falló con 'qdd consola pedidos'.", panelDe(root, cola), p.Tecla, p.Motivo, conEspera(consola.EsperaDe(cola, p.ID)), p.Comando, p.Carpeta, p.Tecla)), nil
}

// listaDeComas separa «a, b» en [a b], sin vacíos.
func listaDeComas(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}
