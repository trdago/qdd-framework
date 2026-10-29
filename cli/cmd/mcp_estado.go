package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Tool MCP qdd_estado (RFC 0006 §3.3): la misma tabla que 'qdd estado', con las mismas validaciones, para que un
// agente la lea al retomar y la actualice al avanzar sin pasar por la terminal.

func registerEstadoTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_estado",
		mcp.WithDescription("Estado por frente del proyecto (.qdd/project/estado.yaml, blueprint estado-por-frente con destino «archivo»). Al RETOMAR una sesión o empezar como agente nuevo, llama accion=ver y sigue desde los ⏳, sin rehacer lo hecho. Al AVANZAR, actualiza la celda con accion=set ANTES de reportar: un ✅ lleva evidencia (commit, versión, n/n pruebas o id de corrida), un ⏳ dice qué espera y de quién, un ❌ nombra la falla. Un frente se cierra con accion=cerrar y su motivo; nunca se borra. accion=check aplica el lector automático (EPF-C01…C07 y, con 'respuesta', EPF-A01). Las escrituras se validan: si violan una regla, no se escribe nada."),
		mcp.WithString("accion", mcp.Description("ver (por defecto) | set | cerrar | check")),
		mcp.WithString("frente", mcp.Description("Id del frente (set, cerrar).")),
		mcp.WithString("nombre", mcp.Description("Nombre del frente al crearlo (set).")),
		mcp.WithString("etapa", mcp.Description("Etapa (columna) tal como está en el archivo (set).")),
		mcp.WithString("estado", mcp.Description("ok | espera | falla | na, o ✅ ⏳ ❌ — (set).")),
		mcp.WithString("evidencia", mcp.Description("Evidencia del ✅.")),
		mcp.WithString("bloqueo", mcp.Description("Qué espera el ⏳.")),
		mcp.WithString("de_quien", mcp.Description("De quién lo espera el ⏳.")),
		mcp.WithString("falla", mcp.Description("La falla del ❌, nombrada.")),
		mcp.WithString("requiere_aprobacion", mcp.Description("Quién debe aprobar antes de pasar a ✅.")),
		mcp.WithBoolean("aprobado", mcp.Description("La aprobación ya la dio quien debía darla (nunca la marques por tu cuenta).")),
		mcp.WithString("motivo", mcp.Description("Por qué se cierra el frente (cerrar).")),
		mcp.WithBoolean("descartado", mcp.Description("Se descarta en vez de cerrarse (cerrar).")),
		mcp.WithString("respuesta", mcp.Description("Texto de la respuesta o el reporte a contrastar con el archivo (check, EPF-A01).")),
	)
	RegisterToolWithTelemetry(s, tool, handleEstadoTool)
}

func handleEstadoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	accion := strings.TrimSpace(stringArg(request, "accion"))
	now := time.Now()
	if accion == "set" {
		return resultadoEscritura(estado.Set(root, cambioDesde(request), now), "celda actualizada")
	}
	if accion == "cerrar" {
		err := estado.Cerrar(root, stringArg(request, "frente"), stringArg(request, "motivo"), boolArg(request, "descartado"), now)
		return resultadoEscritura(err, "frente cerrado")
	}
	e, err := estado.Load(root)
	if errors.Is(err, os.ErrNotExist) {
		return mcp.NewToolResultError(estado.ErrSinArchivo.Error()), nil
	}
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if accion == "check" {
		return resultadoCheck(e, root, stringArg(request, "respuesta"), now), nil
	}
	if accion != "" && accion != "ver" {
		return mcp.NewToolResultError(fmt.Sprintf("accion desconocida %q: usa ver, set, cerrar o check", accion)), nil
	}
	return mcp.NewToolResultText(estado.Markdown(e, now, estado.AlcanceAbiertos)), nil
}

func cambioDesde(request mcp.CallToolRequest) estado.Cambio {
	return estado.Cambio{
		Frente:             stringArg(request, "frente"),
		Nombre:             stringArg(request, "nombre"),
		Etapa:              stringArg(request, "etapa"),
		Estado:             stringArg(request, "estado"),
		Evidencia:          stringArg(request, "evidencia"),
		Bloqueo:            stringArg(request, "bloqueo"),
		DeQuien:            stringArg(request, "de_quien"),
		Falla:              stringArg(request, "falla"),
		RequiereAprobacion: stringArg(request, "requiere_aprobacion"),
		Aprobado:           boolArg(request, "aprobado"),
	}
}

func resultadoEscritura(err error, ok string) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError("no se escribió: " + err.Error()), nil
	}
	return mcp.NewToolResultText(ok + " en " + estado.Ruta + "."), nil
}

func resultadoCheck(e estado.Estado, root, respuesta string, now time.Time) *mcp.CallToolResult {
	vs := estado.Check(e, estado.Anterior(root))
	if respuesta != "" {
		vs = append(vs, estado.CheckRespuesta(e, respuesta, now)...)
	}
	if len(vs) == 0 {
		return mcp.NewToolResultText(estado.Ruta + " cumple EPF-C01…C07 (y EPF-A01 si pasaste 'respuesta'). EPF-C08 (texto y tabla no se contradicen) queda para revisión humana.")
	}
	lineas := make([]string, 0, len(vs))
	for _, v := range vs {
		lineas = append(lineas, v.String())
	}
	return mcp.NewToolResultError(strings.Join(lineas, "\n"))
}
