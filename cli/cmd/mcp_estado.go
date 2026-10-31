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

// descripcionEstadoMCP dice «tarea» (blueprint estado-por-tareas); el parámetro sigue llamándose «frente» por compatibilidad.
const descripcionEstadoMCP = "Estado por tarea (.qdd/project/estado.yaml). RETOMAR → accion=ver, sigue desde los ⏳. AVANZAR → accion=set ANTES de reportar (✅ evidencia · ⏳ qué espera y de quién · ❌ la falla). cerrar con motivo, nunca borrar. check: EPF-C01…C07 (+EPF-A01 con 'respuesta'). Escritura inválida → no escribe."

func registerEstadoTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_estado",
		mcp.WithDescription(descripcionEstadoMCP),
		mcp.WithString("accion", mcp.Description("ver (por defecto) | set | cerrar | check | activar | pulso | terminar")),
		mcp.WithString("fuente", mcp.Description("agente (def.) | subagente | comando (pulso).")),
		mcp.WithString("frente", mcp.Description("Id de la tarea.")),
		mcp.WithString("nombre", mcp.Description("Al crearla (set).")),
		mcp.WithString("etapa", mcp.Description("Columna, como en el archivo (set).")),
		mcp.WithString("estado", mcp.Description("ok | espera | falla | na, o ✅ ⏳ ❌ — (set).")),
		mcp.WithString("evidencia", mcp.Description("Del ✅.")),
		mcp.WithString("bloqueo", mcp.Description("Qué espera el ⏳.")),
		mcp.WithString("de_quien", mcp.Description("De quién espera el ⏳.")),
		mcp.WithString("falla", mcp.Description("Del ❌, nombrada.")),
		mcp.WithString("requiere_aprobacion", mcp.Description("Quién aprueba antes del ✅.")),
		mcp.WithBoolean("aprobado", mcp.Description("Ya la dio quien debía (nunca la marques tú).")),
		mcp.WithString("motivo", mcp.Description("Del cierre (cerrar).")),
		mcp.WithBoolean("descartado", mcp.Description("Descartar en vez de cerrar.")),
		mcp.WithString("respuesta", mcp.Description("Texto a contrastar con el archivo (check, EPF-A01).")),
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
		return resultadoEscritura(err, "tarea cerrada")
	}
	if accion == "activar" {
		return resultadoEscritura(estado.Activar(root, stringArg(request, "frente"), "mcp", now), "tarea en curso: el panel la muestra «● running» y se apaga sola a los 120 s sin pulsos")
	}
	if accion == "terminar" {
		return resultadoEscritura(estado.Terminar(root, stringArg(request, "frente")), "tarea fuera de curso")
	}
	if accion == "pulso" {
		fuente := strings.TrimSpace(stringArg(request, "fuente"))
		if fuente == "" {
			fuente = estado.FuenteAgente
		}
		id, err := estado.Pulsar(root, stringArg(request, "frente"), fuente, now)
		if err == nil && id == "" {
			return mcp.NewToolResultText("sin tarea en curso: el pulso no hizo nada (declárala con accion=activar)"), nil
		}
		return resultadoEscritura(err, "pulso de "+id)
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
		return mcp.NewToolResultError(fmt.Sprintf("accion desconocida %q: usa ver, set, cerrar, check, activar, pulso o terminar", accion)), nil
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
