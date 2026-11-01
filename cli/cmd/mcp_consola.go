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
		mcp.WithDescription("Deja en el panel un comando que la persona ejecuta con un clic (sin permiso, bloqueado o que publica: push de tag, npm publish, gh pr merge) en vez de pedirle '!'. Corre solo si confirma; nunca lo ejecutes tú. Falta algo (CI, tag) → exit 75. Si la tarea sigue cambiando sale del panel: pídelo de nuevo."),
		mcp.WithString("comando", mcp.Description("Exacto, como en su shell (&& y | valen).")),
		mcp.WithString("motivo", mcp.Description("Resumen estilo chat ≤ 60 car., con símbolos (p. ej. «Publicar v2.8.0-rc.6 → npm preview»); el comando ya se ve aparte.")),
		mcp.WithString("carpeta", mcp.Description("Dónde corre (raíz por defecto).")),
		mcp.WithString("tarea", mcp.Description("Tarea que extiende (por defecto, la en curso).")),
		mcp.WithString("despues", mcp.Description("Ids o teclas que deben quedar hechos antes (coma), p. ej. merge → tag.")),
		mcp.WithString("antes", mcp.Description("Ids o teclas que esperan a este (coma); repetirlo lo reordena sin duplicar.")),
		mcp.WithString("resuelto", mcp.Description("Tecla o id resuelto por la persona en el chat: se quita; hecho=true si lo hizo ella.")),
		mcp.WithBoolean("hecho"),
	)
	RegisterToolWithTelemetry(s, tool, handleConsolaPedirTool)
}

// handleConsolaResueltoTool refleja en el panel lo que la persona resolvió en el chat (sincronía panel–chat).
func handleConsolaResueltoTool(root, pedido string, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if request.GetBool("hecho", false) {
		if err := consola.HechoPorLaPersona(root, pedido, "", time.Now()); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText("Pedido " + pedido + " hecho por la persona: el panel lo muestra hecho."), nil
	}
	if err := consola.Descartar(root, pedido); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Pedido " + pedido + " quitado del panel."), nil
}

func handleConsolaPedirTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	if resuelto := strings.TrimSpace(stringArg(request, "resuelto")); resuelto != "" {
		return handleConsolaResueltoTool(root, resuelto, request)
	}
	if strings.TrimSpace(stringArg(request, "comando")) == "" || strings.TrimSpace(stringArg(request, "motivo")) == "" {
		return mcp.NewToolResultError("faltan comando y motivo (o resuelto, para quitar uno que la persona resolvió en el chat)"), nil
	}
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
