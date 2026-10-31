package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/blueprints"
)

func registerBlueprintTools(s *server.MCPServer) {
	add := mcp.NewTool("qdd_blueprint_add",
		mcp.WithDescription("Importa blueprints (estándares de módulo) de la organización: solo los que pidió el humano. Sin repositorio registrado, pregúntale la URL (nunca la inventes) y pásala en 'origen'; QDD la recuerda. Luego → qdd_blueprint_apply."),
		mcp.WithString("blueprints", mcp.Description("Ids, por coma (p. ej. 'auth, auditoria').")),
		mcp.WithString("paquete", mcp.Description("Paquete del catalogo.yaml (en vez de 'blueprints').")),
		mcp.WithBoolean("todos", mcp.Description("Todos los del repositorio.")),
		mcp.WithString("origen", mcp.Description("SOLO si el humano lo indicó (sin uno registrado, u otra fuente suya); al actualizar no hace falta.")),
		mcp.WithBoolean("reemplazar", mcp.Description("Reemplaza uno importado de otra fuente por el de 'origen'. SOLO si el humano lo pidió.")),
		mcp.WithString("ref", mcp.Description("Tag vX.Y.Z (por defecto, la última). Solo si el humano pidió una versión.")),
	)
	RegisterToolWithTelemetry(s, add, handleBlueprintAddTool)

	list := mcp.NewTool("qdd_blueprint_list",
		mcp.WithDescription("Blueprints del proyecto, una línea c/u (❓preguntas 🔒invariantes ✓escenarios); detalle por sección a pedido. Antes de construir un módulo que uno define: manda sobre tus preferencias."),
		mcp.WithString("blueprint", mcp.Description("Id: su índice en JSON.")),
		mcp.WithString("seccion", mcp.Description("Con blueprint: preguntas|invariantes|decisiones|escenarios|vocabulario|todo. calidad (con o sin blueprint): mide la receta, ⚠ defectos y ✂ lo que se acorta sin perder nada.")),
	)
	RegisterToolWithTelemetry(s, list, handleBlueprintListTool)

	apply := mcp.NewTool("qdd_blueprint_apply",
		mcp.WithDescription("Genera desde un blueprint la ficha de un módulo (BORRADOR: la aprueba un humano). Sus preguntas son del HUMANO: házselas tal cual, nunca las respondas tú; si faltan, las devuelve sin escribir nada."),
		mcp.WithString("blueprint", mcp.Required(), mcp.Description("Id del blueprint.")),
		mcp.WithString("respuestas", mcp.Description("Del humano: pregunta=valor, por línea o coma.")),
		mcp.WithString("modulo", mcp.Description("Id de la ficha (por defecto, el del blueprint).")),
		mcp.WithString("rutas", mcp.Description("Carpetas, por coma (por defecto, las del blueprint).")),
		mcp.WithString("pruebas", mcp.Description("Comandos que lo prueban, uno por línea.")),
	)
	RegisterToolWithTelemetry(s, apply, handleBlueprintApplyTool)

	sync := mcp.NewTool("qdd_blueprint_sync",
		mcp.WithDescription("Trae lo publicado en cada fuente de blueprints y dice cuáles tienen versión nueva. Úsala ante BP-06-BEHIND o si el humano pregunta; luego → qdd_blueprint_add y qdd_blueprint_apply."),
	)
	RegisterToolWithTelemetry(s, sync, handleBlueprintSyncTool)

	contribute := mcp.NewTool("qdd_blueprint_contribute",
		mcp.WithDescription("Prepara un cambio a un blueprint (o uno nuevo) en la carpeta central de QDD, nunca en el proyecto: da la carpeta, la rama aporte/<id> y la línea de cada sección. Úsala si el humano quiere mejorar un estándar o BP-05-LEGACY lo pide."),
		mcp.WithString("blueprint", mcp.Required(), mcp.Description("Id a cambiar o crear.")),
		mcp.WithString("origen", mcp.Description("Fuente de un blueprint NUEVO, SOLO si el humano la indicó. Uno importado va a su propia fuente.")),
	)
	RegisterToolWithTelemetry(s, contribute, handleBlueprintContributeTool)

	propose := mcp.NewTool("qdd_blueprint_propose",
		mcp.WithDescription("Publica el aporte como PR a la fuente del blueprint: QDD valida, calcula el semver y fija la versión (no la elijas tú). Publica fuera del proyecto: SOLO con confirmación explícita del humano."),
		mcp.WithString("blueprint", mcp.Required(), mcp.Description("Id del blueprint.")),
		mcp.WithString("mensaje", mcp.Required(), mcp.Description("Qué cambia y por qué (commit y PR).")),
		mcp.WithString("origen", mcp.Description("La misma de qdd_blueprint_contribute (blueprint nuevo).")),
	)
	RegisterToolWithTelemetry(s, propose, handleBlueprintProposeTool)
}

func handleBlueprintSyncTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, err := blueprints.Sincronizar(projectRootForMCP())
	if err != nil && len(res) == 0 {
		return mcp.NewToolResultError(err.Error()), nil
	}
	var s strings.Builder
	var ids []string
	for _, f := range res {
		if f.Problema != "" {
			fmt.Fprintf(&s, "⚠️ %s no se pudo sincronizar: %s\n", f.Origen, f.Problema)
			continue
		}
		fmt.Fprintf(&s, "%s sincronizado (última versión %s).\n", f.Origen, f.Ref)
		for _, n := range f.Novedades {
			fmt.Fprintf(&s, "- %s: %s → %s\n", n.ID, n.Actual, n.Disponible)
			ids = append(ids, n.ID)
		}
	}
	if err != nil {
		fmt.Fprintf(&s, "⚠️ %v\n", err)
	}
	if len(ids) == 0 {
		s.WriteString("No hay versiones nuevas de los blueprints del proyecto.")
		return mcp.NewToolResultText(s.String()), nil
	}
	fmt.Fprintf(&s, "Actualiza con qdd_blueprint_add (blueprints=%s, sin origen: QDD usa la fuente de cada uno) y vuelve a aplicar cada uno con qdd_blueprint_apply; si aparecen preguntas nuevas, házselas al humano.", strings.Join(ids, ", "))
	return mcp.NewToolResultText(s.String()), nil
}

func handleBlueprintContributeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := strings.TrimSpace(stringArg(request, "blueprint"))
	ap, err := blueprints.Aportar(projectRootForMCP(), id, strings.TrimSpace(stringArg(request, "origen")))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	mapa := ""
	if m := blueprints.MapaDeSecciones(ap.Carpeta); m != "" {
		mapa = m + " (edita solo la sección que cambias; detalle: qdd_blueprint_list {blueprint, seccion}).\n"
	}
	return mcp.NewToolResultText(fmt.Sprintf("Aporte listo: rama %s de %s.\nCarpeta: %s\n%sNo cambies 'version:' (la fija QDD). Al terminar, muéstrale el cambio al humano y, con su OK, qdd_blueprint_propose (blueprint=%s).", ap.Rama, ap.Repo, ap.Carpeta, mapa, id)), nil
}

func handleBlueprintProposeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := strings.TrimSpace(stringArg(request, "blueprint"))
	p, err := blueprints.Proponer(projectRootForMCP(), id, stringArg(request, "mensaje"), strings.TrimSpace(stringArg(request, "origen")))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	msg := fmt.Sprintf("%s v%s (cambio %s) publicado en la rama %s.\n- %s\n", id, p.Version, p.Cambio, p.Rama, strings.Join(p.Razones, "\n- "))
	if p.Via != "" {
		msg += p.Via + "\n"
	}
	if p.Aviso != "" {
		return mcp.NewToolResultText(msg + "⚠️ " + p.Aviso), nil
	}
	return mcp.NewToolResultText(msg + "Pull request: " + p.PR + "\nTras el merge, un mantenedor publica la versión con 'qdd blueprint release'."), nil
}

func handleBlueprintAddTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, problems := blueprints.AddManaged(blueprints.AddInput{
		Root:       projectRootForMCP(),
		Origen:     strings.TrimSpace(stringArg(request, "origen")),
		Ref:        strings.TrimSpace(stringArg(request, "ref")),
		Blueprints: splitPaths(stringArg(request, "blueprints")),
		Paquete:    strings.TrimSpace(stringArg(request, "paquete")),
		Todos:      boolArg(request, "todos"),
		Reemplazar: boolArg(request, "reemplazar"),
	})
	if len(problems) > 0 && len(res.Entries) == 0 {
		return mcp.NewToolResultError("No se importó nada:\n- " + strings.Join(problems, "\n- ")), nil
	}
	var s strings.Builder
	var ids []string
	for _, e := range res.Entries {
		fmt.Fprintf(&s, "Importado %s@%s.\n", e.ID, e.Version)
		if aviso := avisoDeReemplazo(e); aviso != "" {
			fmt.Fprintf(&s, "%s Las preguntas nuevas son del humano: llama a qdd_blueprint_apply con ese módulo y házselas tal cual.\n", strings.TrimPrefix(aviso, "[+] "))
		}
		ids = append(ids, e.ID)
	}
	if res.Guardado {
		fmt.Fprintf(&s, "Repositorio de la organización registrado en .qdd/config.yaml y en el registro central de QDD: %s\n", res.Registro.Origen)
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

// handleBlueprintListTool lista una línea por blueprint (su índice); con blueprint da su índice en JSON y con seccion
// solo ese detalle: leer un blueprint no cuesta miles de tokens (pedido del usuario, 2026-10-31).
func handleBlueprintListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root := projectRootForMCP()
	id, seccion := strings.TrimSpace(stringArg(request, "blueprint")), strings.TrimSpace(stringArg(request, "seccion"))
	if seccion == seccionCalidad {
		return calidadDeBlueprints(root, id)
	}
	if id != "" {
		return detalleDeBlueprint(root, id, seccion)
	}
	ix, errs := indicesDelProyecto(root)
	var s strings.Builder
	for _, err := range errs {
		fmt.Fprintf(&s, "⚠️ Blueprint ilegible: %v\n", err)
	}
	if reg, _ := blueprints.LoadRegistro(root); reg.Origen != "" {
		fmt.Fprintf(&s, "Fuente: %s (más con qdd_blueprint_add).\n", reg.Origen)
	}
	if len(ix) == 0 {
		s.WriteString("No hay blueprints importados. Se importan con qdd_blueprint_add (o 'qdd blueprint add' en la terminal).\n")
	}
	for _, i := range ix {
		s.WriteString("- " + blueprints.LineaIndice(i) + "\n")
	}
	if len(ix) > 0 {
		fmt.Fprintf(&s, "Detalle: qdd_blueprint_list {blueprint, seccion: %s}. Para construir, la receta completa: qdd_blueprint_apply.\n", strings.Join(blueprints.SeccionesDeBlueprint, "|"))
	}
	for _, p := range blueprints.Check(root) {
		fmt.Fprintf(&s, "⚠️ %s: %s\n", p.Regla, p.Descripcion)
	}
	return mcp.NewToolResultText(s.String()), nil
}

// seccionCalidad mide la receta en vez de mostrar una sección: de un blueprint, o de todos sin blueprint.
const seccionCalidad = "calidad"

func calidadDeBlueprints(root, id string) (*mcp.CallToolResult, error) {
	var pedidos []string
	if id != "" {
		pedidos = []string{id}
	}
	medidas, fuente, err := blueprints.Medir(root, pedidos)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(informeDeCalidad(medidas, fuente, "qdd_blueprint_contribute", "qdd_blueprint_propose")), nil
}

func detalleDeBlueprint(root, id, seccion string) (*mcp.CallToolResult, error) {
	b, err := blueprints.Find(root, id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if seccion != "" {
		texto, err := blueprints.Seccion(b, seccion)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(texto), nil
	}
	i := blueprints.IndiceDe(b)
	i.Modulos = modulosDe(root)[b.ID]
	out, err := json.Marshal(i)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(out) + "\nEsto es el índice, para elegir. Detalle: seccion = " + strings.Join(blueprints.SeccionesDeBlueprint, "|") + " ('secciones': su línea en blueprint.yaml). Para construir, la receta completa (decisiones, invariantes y escenarios con su detalle): qdd_blueprint_apply."), nil
}

// indicesDelProyecto es el índice de cada blueprint importado, con los módulos generados con él.
func indicesDelProyecto(root string) ([]blueprints.Indice, []error) {
	bps, errs := blueprints.List(root)
	mods := modulosDe(root)
	out := make([]blueprints.Indice, 0, len(bps))
	for _, b := range bps {
		i := blueprints.IndiceDe(b)
		i.Modulos = mods[b.ID]
		out = append(out, i)
	}
	return out, errs
}

func modulosDe(root string) map[string][]string {
	apps, _ := blueprints.LoadApplications(root)
	out := map[string][]string{}
	for _, a := range apps {
		out[a.Blueprint] = append(out[a.Blueprint], a.Modulo)
	}
	return out
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
