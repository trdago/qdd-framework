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
	add := mcp.NewTool("qdd_blueprint_add",
		mcp.WithDescription("Importa al proyecto estándares de módulo (blueprints) desde el repositorio de la organización registrado en .qdd/config.yaml, p. ej. cuando el humano pide 'agrega el blueprint de auth y auditoría'. Úsala solo con lo que el humano pidió. Si no hay repositorio registrado, pregúntale al humano la URL del repositorio de blueprints de su organización (nunca la inventes) y pásala en 'origen': QDD la recordará. Después aplica cada uno con qdd_blueprint_apply."),
		mcp.WithString("blueprints", mcp.Description("Ids a importar, separados por comas (p. ej. 'auth, auditoria').")),
		mcp.WithString("paquete", mcp.Description("Paquete del catalogo.yaml del repositorio (alternativa a 'blueprints').")),
		mcp.WithBoolean("todos", mcp.Description("Importa todos los blueprints del repositorio.")),
		mcp.WithString("origen", mcp.Description("Repositorio de la organización, SOLO si el humano lo indicó y no hay uno registrado.")),
		mcp.WithString("ref", mcp.Description("Tag o versión (p. ej. v1.2.0); por defecto, la registrada.")),
	)
	RegisterToolWithTelemetry(s, add, handleBlueprintAddTool)

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

func handleBlueprintAddTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, problems := blueprints.AddManaged(blueprints.AddInput{
		Root:       projectRootForMCP(),
		Origen:     strings.TrimSpace(stringArg(request, "origen")),
		Ref:        strings.TrimSpace(stringArg(request, "ref")),
		Blueprints: splitPaths(stringArg(request, "blueprints")),
		Paquete:    strings.TrimSpace(stringArg(request, "paquete")),
		Todos:      boolArg(request, "todos"),
	})
	if len(problems) > 0 && len(res.Entries) == 0 {
		return mcp.NewToolResultError("No se importó nada:\n- " + strings.Join(problems, "\n- ")), nil
	}
	var s strings.Builder
	var ids []string
	for _, e := range res.Entries {
		fmt.Fprintf(&s, "Importado %s@%s.\n", e.ID, e.Version)
		ids = append(ids, e.ID)
	}
	if res.Guardado {
		fmt.Fprintf(&s, "Repositorio de la organización registrado en .qdd/config.yaml: %s %s\n", res.Registro.Origen, res.Registro.Ref)
	}
	for _, p := range problems {
		fmt.Fprintf(&s, "⚠️ %s\n", p)
	}
	fmt.Fprintf(&s, "Siguiente paso: aplica cada uno con qdd_blueprint_apply (%s) y hazle al humano las preguntas que devuelva; no las respondas tú.", strings.Join(ids, ", "))
	return mcp.NewToolResultText(s.String()), nil
}

func boolArg(request mcp.CallToolRequest, name string) bool {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return false
	}
	value, _ := args[name].(bool)
	return value
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
	if reg, _ := blueprints.LoadRegistro(root); reg.Origen != "" {
		fmt.Fprintf(&s, "Repositorio de la organización: %s %s (importa más con qdd_blueprint_add).\n\n", reg.Origen, reg.Ref)
	}
	if len(bps) == 0 {
		s.WriteString("No hay blueprints importados. Se importan con qdd_blueprint_add (o 'qdd blueprint add' en la terminal).\n")
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
