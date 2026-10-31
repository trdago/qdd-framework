package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Principios de trabajo QDD que toda IA debe seguir. Se entregan junto con el
// contexto para que la forma de trabajo no dependa de qué asistente se use.
var workingMethod = []string{
	"Antes de tocar un módulo, lee su ficha y respeta su objetivo operativo, su objetivo tecnológico y sus invariantes. Si tu cambio altera alguno, actualiza la ficha con qdd_module_save (queda en borrador) y avísale al humano para que la apruebe.",
	"Pruebas primero: escribe o ajusta la prueba que expresa el cambio, comprueba que falla y recién entonces implementa.",
	"Cero 'else' y salida temprana (cláusulas de guarda) en todas las funciones.",
	"Todo bug se documenta como finding en .qdd/project/findings/ con su prueba de regresión.",
	"No se commitea ni se despliega sin pasar el gate (qdd_audit): pruebas del proyecto + reglas QDD.",
	"Si un módulo tiene un estándar de la organización (blueprint), se construye aplicándolo con qdd_blueprint_apply: sus preguntas las responde el humano (nunca tú) y sus invariantes no se discuten.",
	"Lo temporal va en ./tmp (git lo ignora): scripts de un solo uso, pruebas manuales, descargas, logs y salidas. Nunca lo dejes en la raíz ni en las carpetas de código; si algo vale la pena conservar, muévelo a su lugar con su prueba.",
}

// MetodoDeTrabajo devuelve la forma de trabajo obligatoria, una regla por línea.
func MetodoDeTrabajo() string {
	return strings.Join(workingMethod, "\n")
}

// Estandar resume un blueprint importado al proyecto (lo calcula el paquete
// blueprints, que depende de este).
type Estandar struct {
	ID      string
	Nombre  string
	Version string
	Rutas   []string // rutas sugeridas por el blueprint
	Modulos []string // fichas generadas con él
}

// ContextInput reúne lo que el contexto necesita y que se calcula fuera del
// paquete (p. ej. los comandos de prueba, que resuelve el motor de auditoría).
type ContextInput struct {
	Root         string
	Paths        []string // rutas que la IA va a tocar (opcional)
	TestCommands []string
	Estandares   []Estandar
	Modo         string // modo de permisos de la IA principal en el último mensaje; vacío = sin medir
	ModoVisto    string // a qué hora se vio (HH:MM)
}

// lineaDelProyecto es lo que se hace sin pedir autorización cuando la IA puede actuar sola.
const lineaDelProyecto = "Sigue sin pedir autorización todo lo que continúa la línea del proyecto: las tareas abiertas del estado, " +
	"las pruebas, los findings y lo que se implementa dentro de lo que ya dicen las fichas y los blueprints."

// loDefinido es lo que el proyecto ya decidió: cambiarlo siempre lo autoriza la persona.
const loDefinido = "Pide autorización (una pregunta o un pedido en el panel) solo antes de cambiar algo ya definido: el objetivo, " +
	"las decisiones o las invariantes de una ficha aprobada; una decisión o una invariante de un blueprint (eso va al " +
	"repositorio de la organización); la versión, un tag, una publicación o un despliegue; o algo que borra datos o no se " +
	"puede deshacer. Las preguntas de diseño de un blueprint siguen siendo de la persona."

// ReglaDeAutorizacion dice, según el modo de permisos de la IA principal (permission_mode de Claude Code), cuándo
// pedir autorización. Es la misma regla para todo proyecto: lo que cambia es quién ya está revisando cada acción.
func ReglaDeAutorizacion(modo string) []string {
	switch modo {
	case "auto", "acceptEdits", "bypassPermissions", "dontAsk":
		return []string{"La IA actúa sola en este modo. " + lineaDelProyecto, loDefinido}
	case "default":
		return []string{"La consola ya le pide permiso a la persona por cada acción: no sumes preguntas para lo que continúa la línea del proyecto.", loDefinido}
	case "plan":
		return []string{"Modo plan: investiga y propone; no ejecutes cambios hasta que la persona apruebe el plan."}
	}
	return []string{"Modo sin medir (no hay un mensaje reciente con el panel de qdd): pide autorización como lo pida la consola.", loDefinido}
}

// BuildContext arma la abstracción del proyecto para una IA: propósito, forma
// de trabajo, fichas relevantes (completas), mapa de módulos y zonas sin ficha.
func BuildContext(in ContextInput) string {
	mods, loadErrs := Load(in.Root)
	var b strings.Builder

	fmt.Fprintf(&b, "# Contexto QDD: %s\n\n", filepath.Base(in.Root))
	writePurpose(&b, in.Root)

	b.WriteString("## Forma de trabajo (obligatoria)\n")
	for i, rule := range workingMethod {
		fmt.Fprintf(&b, "%d. %s\n", i+1, rule)
	}
	writeAutorizacion(&b, in)
	b.WriteString("\n## Cómo se prueba el proyecto\n")
	writeList(&b, in.TestCommands, "No hay comandos de prueba detectables: decláralos en .qdd/config.yaml (test_commands) antes de cambiar código.")
	writeEstandares(&b, in.Estandares)

	for _, err := range loadErrs {
		fmt.Fprintf(&b, "\n⚠️ Ficha ilegible: %v\n", err)
	}

	if len(in.Paths) > 0 {
		writeRelevant(&b, in.Root, mods, in.Estandares, in.Paths)
	}
	writeIndex(&b, in.Root, mods)
	writeUncovered(&b, in.Root, mods)
	return b.String()
}

func writeAutorizacion(b *strings.Builder, in ContextInput) {
	b.WriteString("\n## Autorización según el modo de la IA\n")
	if in.Modo != "" {
		fmt.Fprintf(b, "Modo «%s» (visto a las %s).\n", in.Modo, in.ModoVisto)
	}
	writeList(b, ReglaDeAutorizacion(in.Modo), "")
}

func writePurpose(b *strings.Builder, root string) {
	data, err := os.ReadFile(filepath.Join(root, ".qdd", "understanding.json"))
	if err != nil {
		b.WriteString("## Propósito del proyecto\nSin .qdd/understanding.json: ejecuta qdd_learn para registrarlo.\n\n")
		return
	}
	var u struct {
		Summary    string   `json:"summary"`
		Objectives []string `json:"objectives"`
	}
	if json.Unmarshal(data, &u) != nil {
		b.WriteString("## Propósito del proyecto\n.qdd/understanding.json no es JSON válido.\n\n")
		return
	}
	b.WriteString("## Propósito del proyecto\n" + strings.TrimSpace(u.Summary) + "\n")
	writeList(b, u.Objectives, "")
	b.WriteString("\n")
}

func writeEstandares(b *strings.Builder, estandares []Estandar) {
	if len(estandares) == 0 {
		return
	}
	b.WriteString("\n## Estándares de la organización (blueprints)\nEstos módulos tienen un estándar: no los diseñes desde cero.\n")
	for _, e := range estandares {
		fmt.Fprintf(b, "- **%s** (`%s@%s`): ", e.Nombre, e.ID, e.Version)
		if len(e.Modulos) > 0 {
			fmt.Fprintf(b, "aplicado en %s.\n", strings.Join(quoted(e.Modulos), ", "))
			continue
		}
		fmt.Fprintf(b, "sin aplicar. Si vas a construir este módulo, llama primero a qdd_blueprint_apply (blueprint=%s) y hazle al humano las preguntas que devuelva.\n", e.ID)
	}
}

func quoted(items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, "`"+it+"`")
	}
	return out
}

// estandarPara devuelve el blueprint sin aplicar cuyas rutas sugeridas
// contienen la ruta: ahí la ficha se genera aplicándolo, no a mano.
func estandarPara(estandares []Estandar, rel string) (Estandar, bool) {
	for _, e := range estandares {
		if len(e.Modulos) > 0 {
			continue
		}
		if (Module{Rutas: e.Rutas}).covers(rel) {
			return e, true
		}
	}
	return Estandar{}, false
}

func writeRelevant(b *strings.Builder, root string, mods []Module, estandares []Estandar, paths []string) {
	b.WriteString("\n## Módulos que vas a tocar\n")
	seen := map[string]bool{}
	for _, p := range paths {
		rel := relativeTo(root, p)
		covering := Covering(mods, rel)
		if e, ok := estandarPara(estandares, rel); ok && len(covering) == 0 {
			fmt.Fprintf(b, "\n⚠️ `%s` corresponde al estándar **%s** (`%s`), que aún no se aplica. No propongas una ficha a mano: llama a qdd_blueprint_apply (blueprint=%s) y hazle al humano sus preguntas.\n", rel, e.Nombre, e.ID, e.ID)
			continue
		}
		if len(covering) == 0 {
			fmt.Fprintf(b, "\n⚠️ `%s` no pertenece a ningún módulo con ficha. Antes de cambiarlo, propón su ficha con qdd_module_save y pide al humano que la apruebe.\n", rel)
			continue
		}
		for _, m := range covering {
			if seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			writeModule(b, root, m)
		}
	}
}

func relativeTo(root, p string) string {
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(filepath.Clean(p))
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

func writeModule(b *strings.Builder, root string, m Module) {
	fmt.Fprintf(b, "\n### %s (`%s`), %s\n", m.Nombre, m.ID, m.Estado)
	if warning := statusWarning(root, m); warning != "" {
		b.WriteString("⚠️ " + warning + "\n")
	}
	fmt.Fprintf(b, "- Rutas: %s\n", strings.Join(m.Rutas, ", "))
	writeField(b, "Blueprint", m.Blueprint)
	b.WriteString("- **Objetivo operativo**\n")
	fmt.Fprintf(b, "  - Problema: %s\n", m.ObjetivoOperativo.Problema)
	writeField(b, "Usuarios", m.ObjetivoOperativo.Usuarios)
	fmt.Fprintf(b, "  - Criterio de éxito: %s\n", m.ObjetivoOperativo.CriterioDeExito)
	writeField(b, "Caso real", m.ObjetivoOperativo.CasoReal)
	b.WriteString("- **Objetivo tecnológico**\n")
	fmt.Fprintf(b, "  - Arquitectura: %s\n", m.ObjetivoTecnologico.Arquitectura)
	writeItems(b, "Decisiones", m.ObjetivoTecnologico.Decisiones)
	writeItems(b, "Invariantes (no romper)", m.ObjetivoTecnologico.Invariantes)
	writeItems(b, "Cómo se prueba", append(append([]string{}, m.Pruebas.Comandos...), m.Pruebas.Goldensets...))
	writeItems(b, "ADRs", m.ADRs)
	writeItems(b, "Findings", m.Findings)
}

func statusWarning(root string, m Module) string {
	if m.Estado != EstadoAprobado {
		return "Ficha en borrador: aún no la aprobó un humano; confírmala antes de apoyarte en ella."
	}
	drift := Drift(root, m)
	if drift > 0 {
		return fmt.Sprintf("El código del módulo cambió en %d commit(s) desde la última revisión (%s). Verifica que la ficha siga vigente.", drift, m.UltimaRevision)
	}
	return ""
}

func writeField(b *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(b, "  - %s: %s\n", label, value)
}

func writeItems(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "- %s:\n", label)
	for _, it := range items {
		fmt.Fprintf(b, "  - %s\n", it)
	}
}

func writeList(b *strings.Builder, items []string, empty string) {
	if len(items) == 0 && empty != "" {
		b.WriteString(empty + "\n")
		return
	}
	for _, it := range items {
		fmt.Fprintf(b, "- %s\n", it)
	}
}

func writeIndex(b *strings.Builder, root string, mods []Module) {
	b.WriteString("\n## Mapa de módulos\n")
	if len(mods) == 0 {
		fmt.Fprintf(b, "Todavía no hay fichas en %s. Empieza por el módulo que vas a tocar y proponla con qdd_module_save.\n", Dir)
		return
	}
	b.WriteString("| Módulo | Estado | Problema que resuelve | Alerta |\n|---|---|---|---|\n")
	for _, m := range mods {
		fmt.Fprintf(b, "| %s (`%s`) | %s | %s | %s |\n", m.Nombre, m.ID, m.Estado,
			oneLine(m.ObjetivoOperativo.Problema), oneLine(statusWarning(root, m)))
	}
}

func writeUncovered(b *strings.Builder, root string, mods []Module) {
	zones := UncoveredZones(root, mods)
	if len(zones) == 0 {
		return
	}
	const max = 25
	fmt.Fprintf(b, "\n## Zonas de código sin ficha (%d)\n", len(zones))
	for i, z := range zones {
		if i == max {
			fmt.Fprintf(b, "- … y %d más\n", len(zones)-max)
			break
		}
		fmt.Fprintf(b, "- %s\n", z)
	}
}

func oneLine(s string) string {
	s = strings.ReplaceAll(strings.Join(strings.Fields(s), " "), "|", "/")
	runes := []rune(s)
	if len(runes) > 120 {
		return string(runes[:117]) + "..."
	}
	return s
}
