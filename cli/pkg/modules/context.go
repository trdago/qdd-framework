package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PorQueCeroElse es la razón de fondo del cero 'else', la misma en cada lugar donde una IA lee la regla (la forma de
// trabajo, el gate y las instrucciones del proyecto): no es estilo, es para quién se escribe el código.
const PorQueCeroElse = "cada vez menos personas leen el código línea a línea (lo escriben y revisan IAs), así que el que sí se lee tiene que entenderse de un vistazo: con guardas, cada caso especial se resuelve arriba y sale, y el camino principal queda recto, sin ramas que seguir ni estado que recordar"

// Principios de trabajo QDD que toda IA debe seguir. Se entregan junto con el
// contexto para que la forma de trabajo no dependa de qué asistente se use.
var workingMethod = []string{
	"Antes de tocar un módulo, lee su ficha y respeta su objetivo operativo, su objetivo tecnológico y sus invariantes. Si tu cambio altera alguno, actualiza la ficha con qdd_module_save (queda en borrador) y avísale al humano para que la apruebe.",
	"Pruebas primero: escribe o ajusta la prueba que expresa el cambio, comprueba que falla y recién entonces implementa.",
	"Cero 'else' y salida temprana (cláusulas de guarda) en todas las funciones, en cualquier lenguaje. No es una moda: " + PorQueCeroElse + ".",
	"Todo bug se documenta como finding en .qdd/project/findings/ con su prueba de regresión.",
	"Quien reporta valida: un reporte a otro proyecto o a QDD dice desde el inicio cómo lo vas a verificar (qdd reportar --verifico), y queda contigo un ⏳ para hacerlo. Cuando sale el arreglo lo repites y lo cierras con evidencia (qdd reportar --verifique REP-N --evidencia …, o --sigue-fallando para reabrir el mismo). Quien arregla lo deja arreglada, no hecho: se cierra solo con esa verificación.",
	"No se commitea ni se despliega sin pasar el gate (qdd_audit): pruebas del proyecto + reglas QDD.",
	"Si un módulo tiene un estándar de la organización (blueprint), se construye aplicándolo con qdd_blueprint_apply: sus preguntas las responde el humano (nunca tú) y sus invariantes no se discuten.",
	"Lo temporal va en ./tmp (git lo ignora): scripts de un solo uso, pruebas manuales, descargas, logs y salidas. Nunca lo dejes en la raíz ni en las carpetas de código; si algo vale la pena conservar, muévelo a su lugar con su prueba.",
	"Declara la tarea en la que trabajas con qdd_estado accion=activar: el tablero y el teléfono solo muestran como en curso la que se declara. Al terminarla o al quedar esperando a alguien, accion=terminar.",
	"En desarrollo la app corre directo en el PC; Docker es solo para la base de datos. Si alguien igual lo quiere para la app, avísale antes que será más lento y por qué (docs/docker-en-desarrollo.md del repositorio de QDD), y hazlo solo si lo confirma.",
	"Si lo que se explica tiene forma (un proceso, un pipeline, un camino de decisiones), dibújalo con qdd flujo crear (qdd flujo --help): el panel lo muestra en la pestaña Flujos y la persona ve lo mismo que tú.",
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
	// TareasPorRevisar: tareas del estado cuya ficha cambió después de su última celda, una línea cada una (RFC 0017 §3;
	// las calcula el paquete estado, que depende de este).
	TareasPorRevisar []string
	// Completo trae las fichas enteras y el mapa aunque haya rutas. Por defecto las decisiones salen en una línea y el
	// mapa solo sin rutas: con fichas grandes el contexto pasaba el tope de una herramienta (MEJ-073).
	Completo bool
}

// largoDecision es lo que se muestra de cada decisión en el contexto liviano: su primera frase, hasta este largo.
const largoDecision = 160

// maxDecisiones es desde cuántas decisiones una ficha pide dividirse.
const maxDecisiones = 25

// lineaDelProyecto es lo que se hace sin pedir autorización cuando la IA puede actuar sola.
const lineaDelProyecto = "Sigue sin pedir autorización todo lo que continúa la línea del proyecto: las tareas abiertas del estado, " +
	"las pruebas, los findings y lo que se implementa dentro de lo que ya dicen las fichas y los blueprints."

// loDefinido es lo que el proyecto ya decidió: cambiarlo siempre lo autoriza la persona.
const loDefinido = "Pide autorización (una pregunta o un pedido en el panel) solo antes de cambiar algo ya definido: el objetivo, " +
	"las decisiones o las invariantes de una ficha aprobada; una decisión o una invariante de un blueprint (eso va al " +
	"repositorio de la organización); la versión, un tag, una publicación o un despliegue; o algo que borra datos o no se " +
	"puede deshacer. Las preguntas de diseño de un blueprint siguen siendo de la persona."

// loDelChat (sincronía panel–chat, 2026-11-01): lo que el panel espera de la persona puede resolverse en el chat.
const loDelChat = "Si la persona resuelve en el chat algo que el panel espera (un pedido, una ficha, un ⏳ suyo), refléjalo en el acto: " +
	"qdd_consola_pedir con resuelto (hecho si lo hizo ella), qdd_estado set; una ficha, solo si ella la aprueba."

// ReglaDeAutorizacion dice, según el modo de permisos de la IA principal (permission_mode de Claude Code), cuándo
// pedir autorización. Es la misma regla para todo proyecto: lo que cambia es quién ya está revisando cada acción.
func ReglaDeAutorizacion(modo string) []string {
	switch modo {
	case "auto", "acceptEdits", "bypassPermissions", "dontAsk":
		return []string{"La IA actúa sola en este modo. " + lineaDelProyecto, loDefinido, loDelChat}
	case "default":
		return []string{"La consola ya le pide permiso a la persona por cada acción: no sumes preguntas para lo que continúa la línea del proyecto.", loDefinido, loDelChat}
	case "plan":
		return []string{"Modo plan: investiga y propone; no ejecutes cambios hasta que la persona apruebe el plan."}
	}
	return []string{"Modo sin medir (no hay un mensaje reciente con el panel de qdd): pide autorización como lo pida la consola.", loDefinido, loDelChat}
}

// BuildContext arma la abstracción del proyecto para una IA: propósito, forma de trabajo, la ficha de lo que va a tocar
// (decisiones en una línea salvo Completo), el mapa de módulos (sin rutas o Completo) y las zonas sin ficha.
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
	writeTareasPorRevisar(&b, in.TareasPorRevisar)

	for _, err := range loadErrs {
		fmt.Fprintf(&b, "\n⚠️ Ficha ilegible: %v\n", err)
	}

	if len(in.Paths) == 0 || in.Completo {
		writeRelevant(&b, in, mods)
		writeIndex(&b, in.Root, mods)
		writeUncovered(&b, in.Root, mods)
		return b.String()
	}
	writeRelevant(&b, in, mods)
	fmt.Fprintf(&b, "\n## Mapa de módulos\n%d módulos con ficha: el mapa sale con qdd_project_context sin rutas.\n", len(mods))
	writeUncovered(&b, in.Root, mods)
	return b.String()
}

// writeTareasPorRevisar: QDD no reescribe nombres ni criterios; le dice a la IA cuáles revisar (RFC 0017 §3).
func writeTareasPorRevisar(b *strings.Builder, lineas []string) {
	if len(lineas) == 0 {
		return
	}
	b.WriteString("\n## Tareas por revisar\n")
	writeList(b, lineas, "")
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

func writeRelevant(b *strings.Builder, in ContextInput, mods []Module) {
	if len(in.Paths) == 0 {
		return
	}
	root, estandares, paths := in.Root, in.Estandares, in.Paths
	b.WriteString("\n## Módulos que vas a tocar\n")
	if !in.Completo {
		b.WriteString("Decisiones en una línea: la ficha entera con qdd_project_context detalle=completo ('qdd modules context --completo <rutas>').\n")
	}
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
			writeModule(b, root, m, in.Completo)
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

func writeModule(b *strings.Builder, root string, m Module, completo bool) {
	fmt.Fprintf(b, "\n### %s (`%s`), %s\n", m.Nombre, m.ID, m.Estado)
	if warning := statusWarning(root, m); warning != "" {
		b.WriteString("⚠️ " + warning + "\n")
	}
	fmt.Fprintf(b, "- Rutas: %s\n", strings.Join(m.Rutas, ", "))
	writeField(b, "Blueprint", m.Blueprint)
	b.WriteString("- **Objetivo operativo**\n")
	fmt.Fprintf(b, "  - Problema: %s\n", m.ObjetivoOperativo.Problema)
	if completo {
		writeField(b, "Usuarios", m.ObjetivoOperativo.Usuarios)
	}
	fmt.Fprintf(b, "  - Criterio de éxito: %s\n", m.ObjetivoOperativo.CriterioDeExito)
	if completo {
		writeField(b, "Caso real", m.ObjetivoOperativo.CasoReal)
	}
	b.WriteString("- **Objetivo tecnológico**\n")
	fmt.Fprintf(b, "  - Arquitectura: %s\n", m.ObjetivoTecnologico.Arquitectura)
	writeItems(b, "Decisiones", decisionesDe(m, completo))
	if len(m.ObjetivoTecnologico.Decisiones) > maxDecisiones {
		fmt.Fprintf(b, "  - ⚠️ %d decisiones: la ficha creció más de lo que una IA lee con cuidado; propón dividirla en módulos (MEJ-073).\n", len(m.ObjetivoTecnologico.Decisiones))
	}
	writeItems(b, "Invariantes (no romper; cada uno con su id y las pruebas que lo citan)", invariantesDe(root, m))
	writeItems(b, "Cómo se prueba", append(append([]string{}, m.Pruebas.Comandos...), m.Pruebas.Goldensets...))
	writeItems(b, "ADRs", m.ADRs)
	writeItems(b, "Findings", m.Findings)
}

// decisionesDe: enteras con completo; si no, la primera frase de cada una, hasta largoDecision.
func decisionesDe(m Module, completo bool) []string {
	if completo {
		return m.ObjetivoTecnologico.Decisiones
	}
	out := make([]string, 0, len(m.ObjetivoTecnologico.Decisiones))
	for _, d := range m.ObjetivoTecnologico.Decisiones {
		out = append(out, primeraFrase(d, largoDecision))
	}
	return out
}

// primeraFrase corta en el primer punto seguido de espacio, o en max runas con «…».
func primeraFrase(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.Index(s, ". "); i >= 0 && len([]rune(s[:i+1])) <= max {
		return s[:i+1] + " …"
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
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

// invariantesDe son los invariantes con su id de contrato y sus pruebas (contrato.go).
func invariantesDe(root string, m Module) []string {
	citas := CitasDePruebas(root)
	var out []string
	for _, cl := range Contrato(m, citas) {
		out = append(out, lineaDeClausula(cl))
	}
	return out
}
