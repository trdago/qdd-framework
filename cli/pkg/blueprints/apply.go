package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/modules"
	"gopkg.in/yaml.v3"
)

// Generado recuerda lo que el blueprint escribió en la ficha, para distinguirlo
// de lo que el proyecto agregó por su cuenta cuando se vuelve a aplicar.
type Generado struct {
	Decisiones  []string `yaml:"decisiones,omitempty"`
	Invariantes []string `yaml:"invariantes,omitempty"`
}

// Aplicacion registra con qué respuestas se aplicó un blueprint a un módulo.
// Es la decisión de diseño del proyecto: se versiona con el código.
type Aplicacion struct {
	Blueprint  string            `yaml:"blueprint"`
	Version    string            `yaml:"version"`
	Modulo     string            `yaml:"modulo"`
	Respuestas map[string]string `yaml:"respuestas"`
	Generado   Generado          `yaml:"generado"`
	AplicadoEl string            `yaml:"aplicado_el"`

	File string `yaml:"-"`
}

// Resolver cruza las respuestas con las preguntas del blueprint. Devuelve las
// respuestas que aplican, las preguntas aplicables aún sin respuesta (en
// orden; la primera siempre se puede preguntar ya) y los problemas (valores
// inválidos o preguntas inexistentes). Las respuestas a preguntas que no
// aplican se descartan.
func (b Blueprint) Resolver(respuestas map[string]string) (map[string]string, []Pregunta, []string) {
	aplicadas := map[string]string{}
	var pendientes []Pregunta
	var problems []string
	conocidas := map[string]bool{}
	for _, p := range b.Preguntas {
		conocidas[p.ID] = true
		estado := condicion(p, aplicadas, pendientes)
		if estado == condicionNoCumple {
			continue
		}
		valor, ok := respuestas[p.ID]
		if !ok || estado == condicionPendiente {
			pendientes = append(pendientes, p)
			continue
		}
		if _, valido := p.opcion(valor); !valido {
			problems = append(problems, fmt.Sprintf("'%s=%s' no es válido; opciones: %s", p.ID, valor, strings.Join(p.Valores(), ", ")))
			continue
		}
		aplicadas[p.ID] = valor
	}
	for id := range respuestas {
		if conocidas[id] {
			continue
		}
		problems = append(problems, fmt.Sprintf("el blueprint '%s' no tiene la pregunta '%s'", b.ID, id))
	}
	sort.Strings(problems)
	return aplicadas, pendientes, problems
}

const (
	condicionCumple = iota
	condicionNoCumple
	condicionPendiente
)

func condicion(p Pregunta, aplicadas map[string]string, pendientes []Pregunta) int {
	for id, valores := range p.Si {
		if contiene(pendientes, id) {
			return condicionPendiente
		}
		if !inList(valores, aplicadas[id]) {
			return condicionNoCumple
		}
	}
	return condicionCumple
}

func contiene(preguntas []Pregunta, id string) bool {
	for _, p := range preguntas {
		if p.ID == id {
			return true
		}
	}
	return false
}

func inList(list []string, v string) bool {
	for _, it := range list {
		if it == v {
			return true
		}
	}
	return false
}

// componer reúne lo que implican las respuestas: decisiones, invariantes y
// escenarios de conformidad, con los {{id}} reemplazados por la respuesta.
func (b Blueprint) componer(respuestas map[string]string) (Generado, []Escenario) {
	gen := Generado{
		Decisiones:  append([]string{}, b.Ficha.ObjetivoTecnologico.Decisiones...),
		Invariantes: append([]string{}, b.Ficha.ObjetivoTecnologico.Invariantes...),
	}
	for _, t := range b.Vocabulario {
		gen.Decisiones = append(gen.Decisiones, fmt.Sprintf("Vocabulario «%s»: %s", t.Termino, strings.TrimSpace(t.Definicion)))
	}
	escenarios := append([]Escenario{}, b.Conformidad...)
	for _, p := range b.Preguntas {
		valor, ok := respuestas[p.ID]
		if !ok {
			continue
		}
		o, _ := p.opcion(valor)
		gen.Decisiones = append(gen.Decisiones, fmt.Sprintf("%s → %s", p.ID, valor))
		gen.Decisiones = append(gen.Decisiones, o.Decisiones...)
		gen.Invariantes = append(gen.Invariantes, o.Invariantes...)
		escenarios = append(escenarios, o.Conformidad...)
	}
	gen.Decisiones = expandAll(dedupe(gen.Decisiones), respuestas)
	gen.Invariantes = expandAll(dedupe(gen.Invariantes), respuestas)
	escenarios = dedupeEscenarios(escenarios)
	for i := range escenarios {
		escenarios[i].Titulo = expand(escenarios[i].Titulo, respuestas)
		escenarios[i].Dado = expand(escenarios[i].Dado, respuestas)
		escenarios[i].Cuando = expand(escenarios[i].Cuando, respuestas)
		escenarios[i].Entonces = expand(escenarios[i].Entonces, respuestas)
	}
	return gen, escenarios
}

func dedupeEscenarios(escenarios []Escenario) []Escenario {
	seen := map[string]bool{}
	var out []Escenario
	for _, e := range escenarios {
		if seen[e.ID] {
			continue
		}
		seen[e.ID] = true
		out = append(out, e)
	}
	return out
}

func expand(s string, respuestas map[string]string) string {
	for id, v := range respuestas {
		s = strings.ReplaceAll(s, "{{"+id+"}}", v)
	}
	return s
}

func expandAll(items []string, respuestas map[string]string) []string {
	for i := range items {
		items[i] = expand(items[i], respuestas)
	}
	return items
}

func dedupe(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" || seen[it] {
			continue
		}
		seen[it] = true
		out = append(out, it)
	}
	return out
}

// ApplyInput es lo que necesita Apply.
type ApplyInput struct {
	Root       string
	Blueprint  string            // id del blueprint importado
	Modulo     string            // id de la ficha; por defecto, el id del blueprint
	Rutas      []string          // carpetas del módulo; por defecto, ficha.rutas del blueprint
	Respuestas map[string]string // se suman a las de una aplicación anterior
	Pruebas    []string          // comandos de prueba del proyecto para este módulo
}

// ApplyResult describe lo que dejó Apply. Si Pendientes no está vacío no se
// escribió nada: faltan decisiones que debe tomar un humano.
type ApplyResult struct {
	Ficha       string
	Aplicacion  string
	Conformidad string
	Esqueletos  []string
	Pendientes  []Pregunta
}

// Apply genera, a partir de un blueprint importado y de las respuestas del
// proyecto, la ficha del módulo (siempre en BORRADOR: la aprueba un humano),
// sus escenarios de conformidad, el registro de la aplicación y, para las
// rutas que aún no existen, un esqueleto con el diseño conceptual.
func Apply(in ApplyInput) (ApplyResult, []string) {
	b, err := Find(in.Root, in.Blueprint)
	if err != nil {
		return ApplyResult{}, []string{err.Error()}
	}
	if problems := b.Validate(); len(problems) > 0 {
		return ApplyResult{}, prefix("blueprint inválido: ", problems)
	}
	modulo := firstNonEmpty(in.Modulo, b.ID)
	if !idPattern.MatchString(modulo) {
		return ApplyResult{}, []string{"modulo: en minúsculas, solo letras, números, '-' y '_'"}
	}
	previa, _ := LoadApplication(in.Root, modulo)
	if previa.Blueprint != "" && previa.Blueprint != b.ID {
		return ApplyResult{}, []string{fmt.Sprintf("el módulo '%s' ya se generó con el blueprint '%s'; usa otro --modulo", modulo, previa.Blueprint)}
	}
	respuestas, pendientes, problems := b.Resolver(merge(previa.Respuestas, in.Respuestas))
	if len(problems) > 0 {
		return ApplyResult{}, problems
	}
	if len(pendientes) > 0 {
		return ApplyResult{Pendientes: pendientes}, nil
	}
	rutas := in.Rutas
	if len(rutas) == 0 {
		rutas = b.Ficha.Rutas
	}
	if problems := validateRutas(rutas); len(problems) > 0 {
		return ApplyResult{}, problems
	}

	gen, escenarios := b.componer(respuestas)
	res := ApplyResult{}
	res.Conformidad = filepath.ToSlash(filepath.Join(".qdd", "project", "goldensets", modulo, "conformidad.yaml"))
	ficha := b.ficha(in.Root, modulo, rutas, gen, previa.Generado, in.Pruebas, res.Conformidad)

	esqueletos, err := writeSkeletons(in.Root, b, ficha, respuestas)
	if err != nil {
		return ApplyResult{}, []string{err.Error()}
	}
	res.Esqueletos = esqueletos
	if err := writeConformidad(in.Root, res.Conformidad, b, escenarios); err != nil {
		return ApplyResult{}, []string{err.Error()}
	}
	path, problems := modules.Save(in.Root, ficha)
	if len(problems) > 0 {
		return ApplyResult{}, problems
	}
	res.Ficha = path
	app := Aplicacion{Blueprint: b.ID, Version: b.Version, Modulo: modulo, Respuestas: respuestas, Generado: gen, AplicadoEl: time.Now().Format("2006-01-02")}
	res.Aplicacion, err = saveApplication(in.Root, app)
	if err != nil {
		return ApplyResult{}, []string{err.Error()}
	}
	return res, nil
}

// ficha arma la ficha del módulo. Si ya existía, conserva lo que el proyecto
// le agregó (decisiones e invariantes propias, ADRs, findings, pruebas, caso
// real) y reemplaza solo lo que venía del blueprint.
func (b Blueprint) ficha(root, modulo string, rutas []string, gen, genPrevio Generado, pruebas []string, conformidad string) modules.Module {
	actual, _ := modules.Find(root, modulo)
	base := b.Ficha
	m := modules.Module{
		ID:                  modulo,
		Nombre:              base.Nombre,
		Rutas:               rutas,
		Blueprint:           b.Ref(),
		ObjetivoOperativo:   base.ObjetivoOperativo,
		ObjetivoTecnologico: base.ObjetivoTecnologico,
	}
	m.ObjetivoOperativo.Usuarios = firstNonEmpty(actual.ObjetivoOperativo.Usuarios, base.ObjetivoOperativo.Usuarios)
	m.ObjetivoOperativo.CasoReal = firstNonEmpty(actual.ObjetivoOperativo.CasoReal, base.ObjetivoOperativo.CasoReal)
	m.ObjetivoTecnologico.Decisiones = dedupe(append(gen.Decisiones, propias(actual.ObjetivoTecnologico.Decisiones, genPrevio.Decisiones)...))
	m.ObjetivoTecnologico.Invariantes = dedupe(append(gen.Invariantes, propias(actual.ObjetivoTecnologico.Invariantes, genPrevio.Invariantes)...))
	m.Pruebas.Comandos = dedupe(append(actual.Pruebas.Comandos, pruebas...))
	m.Pruebas.Goldensets = dedupe(append(actual.Pruebas.Goldensets, conformidad))
	m.ADRs = actual.ADRs
	m.Findings = actual.Findings
	m.Fuentes = dedupe(append(actual.Fuentes,
		filepath.ToSlash(filepath.Join(Dir, b.ID, ManifestName)),
		filepath.ToSlash(filepath.Join(ApplicationsDir, modulo+".yaml"))))
	return m
}

// propias devuelve lo que está en la ficha pero no lo generó el blueprint.
func propias(actuales, generados []string) []string {
	gen := map[string]bool{}
	for _, g := range generados {
		gen[g] = true
	}
	var out []string
	for _, a := range actuales {
		if gen[a] || strings.HasPrefix(a, "Vocabulario «") {
			continue
		}
		out = append(out, a)
	}
	return out
}

func validateRutas(rutas []string) []string {
	if len(rutas) == 0 {
		return []string{"rutas: indica dónde vivirá el código del módulo (p. ej. --ruta internal/auth); el blueprint no sugiere ninguna"}
	}
	var problems []string
	for _, r := range rutas {
		clean := filepath.ToSlash(filepath.Clean(r))
		if !isInside(r) {
			problems = append(problems, fmt.Sprintf("rutas: '%s' debe ser relativa a la raíz del proyecto", r))
			continue
		}
		if clean == ".qdd" || strings.HasPrefix(clean, ".qdd/") {
			problems = append(problems, fmt.Sprintf("rutas: '%s' está dentro de .qdd; el código de la aplicación nunca vive ahí", r))
		}
	}
	return problems
}

// writeSkeletons crea las rutas del módulo que aún no existen, con un
// README.md que deja el esqueleto conceptual. Nunca toca carpetas existentes.
func writeSkeletons(root string, b Blueprint, m modules.Module, respuestas map[string]string) ([]string, error) {
	var created []string
	for _, r := range m.Rutas {
		dir := filepath.Join(root, filepath.Clean(r))
		if _, err := os.Stat(dir); err == nil {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return created, err
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(skeletonReadme(b, m, respuestas)), 0644); err != nil {
			return created, err
		}
		created = append(created, filepath.ToSlash(filepath.Clean(r)))
	}
	return created, nil
}

func skeletonReadme(b Blueprint, m modules.Module, respuestas map[string]string) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n", m.Nombre)
	fmt.Fprintf(&s, "Esqueleto generado por QDD desde el blueprint `%s`", b.Ref())
	if b.Organizacion != "" {
		fmt.Fprintf(&s, " de %s", b.Organizacion)
	}
	fmt.Fprintf(&s, ". La fuente de verdad es la ficha `%s/%s.yaml`.\n\n", modules.Dir, m.ID)
	fmt.Fprintf(&s, "**Problema:** %s\n\n**Criterio de éxito:** %s\n\n", strings.TrimSpace(m.ObjetivoOperativo.Problema), strings.TrimSpace(m.ObjetivoOperativo.CriterioDeExito))
	s.WriteString("## Decisiones del proyecto\n")
	for _, p := range b.Preguntas {
		if v, ok := respuestas[p.ID]; ok {
			fmt.Fprintf(&s, "- %s **%s**\n", p.Pregunta, v)
		}
	}
	s.WriteString("\n## Invariantes (no romper)\n")
	for _, inv := range m.ObjetivoTecnologico.Invariantes {
		fmt.Fprintf(&s, "- %s\n", inv)
	}
	fmt.Fprintf(&s, "\n## Conformidad\nLos escenarios que toda implementación debe pasar están en `%s`.\n", m.Pruebas.Goldensets[len(m.Pruebas.Goldensets)-1])
	if len(b.Referencias) > 0 {
		s.WriteString("\n## Implementaciones de referencia\n")
		for _, r := range b.Referencias {
			fmt.Fprintf(&s, "- `%s`\n", filepath.ToSlash(filepath.Join(Dir, b.ID, r)))
		}
	}
	return s.String()
}

func writeConformidad(root, rel string, b Blueprint, escenarios []Escenario) error {
	doc := struct {
		Blueprint  string      `yaml:"blueprint"`
		Escenarios []Escenario `yaml:"escenarios"`
	}{b.Ref(), escenarios}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("# Escenarios de conformidad generados por 'qdd blueprint apply' desde %s.\n# No se editan a mano: cambian al volver a aplicar el blueprint.\n", b.Ref())
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(header), data...), 0644)
}

// LoadApplication lee el registro de aplicación de un módulo.
func LoadApplication(root, modulo string) (Aplicacion, error) {
	path := filepath.Join(root, ApplicationsDir, modulo+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Aplicacion{}, err
	}
	var a Aplicacion
	if err := yaml.Unmarshal(data, &a); err != nil {
		return Aplicacion{}, fmt.Errorf("%s: YAML inválido: %w", path, err)
	}
	a.File = path
	return a, nil
}

// LoadApplications lee todos los registros de aplicación del proyecto.
func LoadApplications(root string) ([]Aplicacion, []error) {
	files, _ := filepath.Glob(filepath.Join(root, ApplicationsDir, "*.yaml"))
	sort.Strings(files)
	var apps []Aplicacion
	var errs []error
	for _, f := range files {
		a, err := LoadApplication(root, strings.TrimSuffix(filepath.Base(f), ".yaml"))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		apps = append(apps, a)
	}
	return apps, errs
}

func saveApplication(root string, a Aplicacion) (string, error) {
	dir := filepath.Join(root, ApplicationsDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	data, err := yaml.Marshal(a)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, a.Modulo+".yaml")
	return path, os.WriteFile(path, data, 0644)
}

func merge(base, over map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func prefix(p string, items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, p+it)
	}
	return out
}
