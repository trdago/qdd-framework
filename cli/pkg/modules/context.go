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
}

// ContextInput reúne lo que el contexto necesita y que se calcula fuera del
// paquete (p. ej. los comandos de prueba, que resuelve el motor de auditoría).
type ContextInput struct {
	Root         string
	Paths        []string // rutas que la IA va a tocar (opcional)
	TestCommands []string
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
	b.WriteString("\n## Cómo se prueba el proyecto\n")
	writeList(&b, in.TestCommands, "No hay comandos de prueba detectables: decláralos en .qdd/config.yaml (test_commands) antes de cambiar código.")

	for _, err := range loadErrs {
		fmt.Fprintf(&b, "\n⚠️ Ficha ilegible: %v\n", err)
	}

	if len(in.Paths) > 0 {
		writeRelevant(&b, in.Root, mods, in.Paths)
	}
	writeIndex(&b, in.Root, mods)
	writeUncovered(&b, in.Root, mods)
	return b.String()
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

func writeRelevant(b *strings.Builder, root string, mods []Module, paths []string) {
	b.WriteString("\n## Módulos que vas a tocar\n")
	seen := map[string]bool{}
	for _, p := range paths {
		rel := relativeTo(root, p)
		covering := Covering(mods, rel)
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
