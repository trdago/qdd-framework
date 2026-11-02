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
const descripcionEstadoMCP = "Estado por tarea (estado.yaml). Retomar: ver, desde los ⏳. Avanzar: set antes de reportar (✅ evidencia · ⏳ qué y de quién · ❌ falla). objetivo: criterio, foco, ciclo («-» proyecto); se cumple con evidencia. nueva: qdd da el id. tarea: objetivo y depende_de."

func registerEstadoTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_estado",
		mcp.WithDescription(descripcionEstadoMCP),
		mcp.WithString("accion", mcp.Description("ver|nueva|set|cerrar|check|activar|pulso|terminar|objetivo|tarea|ciclo")),
		mcp.WithString("fuente", mcp.Description("agente|subagente|comando")),
		mcp.WithString("frente", mcp.Description("Id.")),
		mcp.WithString("nombre", mcp.Description("Al crearla.")),
		mcp.WithString("etapa", mcp.Description("Columna (ciclo: todas, con coma).")),
		mcp.WithString("estado", mcp.Description("ok|espera|falla|na")),
		mcp.WithString("evidencia"),
		mcp.WithString("bloqueo"),
		mcp.WithString("de_quien"),
		mcp.WithString("falla"),
		mcp.WithString("requiere_aprobacion", mcp.Description("Quién aprueba el ✅.")),
		mcp.WithBoolean("aprobado", mcp.Description("Dada por quien debía; nunca tú.")),
		mcp.WithString("motivo", mcp.Description("Del cierre.")),
		mcp.WithBoolean("descartado", mcp.Description("En vez de cerrar.")),
		mcp.WithString("respuesta"),
		mcp.WithString("objetivo", mcp.Description("Id; «-» ninguno.")),
		mcp.WithString("criterio"),
		mcp.WithBoolean("foco"),
		mcp.WithBoolean("reabrir"),
		mcp.WithString("depende_de", mcp.Description("Ids con coma; «-» ninguna.")),
		mcp.WithString("ciclo"),
		mcp.WithBoolean("aunque_exista"),
	)
	RegisterToolWithTelemetry(s, tool, handleEstadoTool)
}

func handleEstadoTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	accion := strings.TrimSpace(stringArg(request, "accion"))
	now := time.Now()
	if accion == "nueva" {
		id, err := estado.Nueva(root, estado.CambioNueva{Nombre: stringArg(request, "nombre"), Objetivo: cambioTareaDesde(request).Objetivo,
			DependeDe: cambioTareaDesde(request).DependeDe, AunqueExista: boolArg(request, "aunque_exista")}, now)
		return resultadoEscritura(err, "tarea creada: "+id)
	}
	if accion == "set" {
		return resultadoEscritura(estado.Set(root, cambioDesde(request), now), "celda actualizada")
	}
	if accion == "cerrar" {
		err := estado.Cerrar(root, stringArg(request, "frente"), stringArg(request, "motivo"), boolArg(request, "descartado"), now)
		return resultadoEscritura(err, "tarea cerrada")
	}
	if accion == "ciclo" {
		err := estado.DeclararCiclo(root, stringArg(request, "ciclo"), listaDeIDs(stringArg(request, "etapa")), now)
		return resultadoEscritura(err, "ciclo declarado")
	}
	if accion == "objetivo" {
		c := estado.CambioObjetivo{Nombre: stringArg(request, "nombre"), Criterio: stringArg(request, "criterio"), Foco: boolArg(request, "foco")}
		if ciclo := strings.TrimSpace(stringArg(request, "ciclo")); ciclo != "" {
			c.CambiaCiclo, c.Ciclo = true, strings.TrimPrefix(ciclo, "-")
		}
		err := aplicarObjetivo(root, stringArg(request, "objetivo"), c, stringArg(request, "evidencia"), stringArg(request, "motivo"), boolArg(request, "descartado"), boolArg(request, "reabrir"), now)
		return resultadoEscritura(err, "objetivo actualizado")
	}
	if accion == "tarea" {
		return resultadoEscritura(estado.GuardarTarea(root, cambioTareaDesde(request), now), "tarea actualizada")
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
		return mcp.NewToolResultError(fmt.Sprintf("accion desconocida %q: usa ver, nueva, set, cerrar, check, activar, pulso, terminar, objetivo, tarea o ciclo", accion)), nil
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
		AunqueExista:       boolArg(request, "aunque_exista"),
	}
}

// cambioTareaDesde: un argumento ausente no cambia nada; «-» lo deja vacío.
func cambioTareaDesde(request mcp.CallToolRequest) estado.CambioTarea {
	c := estado.CambioTarea{ID: stringArg(request, "frente"), Nombre: stringArg(request, "nombre"), AunqueExista: boolArg(request, "aunque_exista")}
	if o := strings.TrimSpace(stringArg(request, "objetivo")); o != "" {
		c.CambiaObjetivo = true
		c.Objetivo = strings.TrimPrefix(o, "-")
	}
	if d := strings.TrimSpace(stringArg(request, "depende_de")); d != "" {
		c.CambiaDependencias = true
		c.DependeDe = listaDeIDs(strings.TrimPrefix(d, "-"))
	}
	return c
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
