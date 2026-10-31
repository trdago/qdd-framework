// Package blueprints gestiona los blueprints de QDD: arquetipos conceptuales
// de un módulo (vocabulario, preguntas de diseño, decisiones, invariantes y
// escenarios de conformidad) que una organización mantiene en su propio
// repositorio, privado si quiere, y que cada proyecto importa para generar la
// ficha de su módulo. El core de QDD solo trae el mecanismo, nunca el
// contenido de una organización.
package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/modules"
	"gopkg.in/yaml.v3"
)

const (
	// Dir es donde el esquema anterior (v1.10–v1.11) copiaba cada blueprint
	// dentro del proyecto. Hoy solo existe hasta que Migrar lo convierte.
	Dir = ".qdd/blueprints"
	// LockFile fija origen, commit y huella de cada blueprint importado.
	LockFile = ".qdd/blueprints.lock.yaml"
	// ApplicationsDir guarda las respuestas con que se aplicó cada blueprint.
	ApplicationsDir = ".qdd/project/blueprints"
	// ManifestName es el archivo que define el blueprint dentro de su carpeta.
	ManifestName = "blueprint.yaml"
)

// Termino fija el significado de una palabra para toda la organización
// (p. ej. qué es un "perfil"), para que ningún equipo lo interprete distinto.
type Termino struct {
	Termino    string `yaml:"termino"`
	Definicion string `yaml:"definicion"`
}

// Escenario es un caso de conformidad, agnóstico del lenguaje: lo que toda
// implementación del módulo debe cumplir.
type Escenario struct {
	ID       string `yaml:"id"`
	Titulo   string `yaml:"titulo"`
	Dado     string `yaml:"dado,omitempty"`
	Cuando   string `yaml:"cuando"`
	Entonces string `yaml:"entonces"`
}

// Opcion es una respuesta posible a una pregunta, con lo que implica elegirla.
type Opcion struct {
	Valor       string      `yaml:"valor"`
	Descripcion string      `yaml:"descripcion,omitempty"`
	DependeDe   []string    `yaml:"depende_de,omitempty"`
	Decisiones  []string    `yaml:"decisiones,omitempty"`
	Invariantes []string    `yaml:"invariantes,omitempty"`
	Conformidad []Escenario `yaml:"conformidad,omitempty"`
	Alcance     Alcance     `yaml:"alcance,omitempty"` // límites que solo valen con esta opción
}

// Valores acepta en YAML un valor suelto o una lista.
type Valores []string

func (v *Valores) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		*v = Valores{node.Value}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err != nil {
		return err
	}
	*v = list
	return nil
}

// Pregunta es una decisión de diseño que el blueprint no puede tomar por el
// proyecto. Si declara 'si', solo aplica cuando las respuestas previas
// coinciden (p. ej. el proveedor de WhatsApp solo se pregunta si se eligió
// recuperar por WhatsApp).
type Pregunta struct {
	ID         string             `yaml:"id"`
	Pregunta   string             `yaml:"pregunta"`
	Ayuda      string             `yaml:"ayuda,omitempty"`
	PorDefecto string             `yaml:"por_defecto,omitempty"`
	Si         map[string]Valores `yaml:"si,omitempty"`
	Opciones   []Opcion           `yaml:"opciones"`

	abierta bool // un parámetro de negocio: se responde con un valor, no con una opción
}

// FichaBase es el punto de partida de la ficha de módulo que genera el blueprint.
type FichaBase struct {
	Nombre              string                      `yaml:"nombre"`
	Rutas               []string                    `yaml:"rutas,omitempty"`
	ObjetivoOperativo   modules.ObjetivoOperativo   `yaml:"objetivo_operativo"`
	ObjetivoTecnologico modules.ObjetivoTecnologico `yaml:"objetivo_tecnologico"`
}

// Blueprint es el arquetipo conceptual de un módulo.
type Blueprint struct {
	ID           string      `yaml:"id"`
	Nombre       string      `yaml:"nombre"`
	Version      string      `yaml:"version"`
	Organizacion string      `yaml:"organizacion,omitempty"`
	RequiereQDD  string      `yaml:"requiere_qdd,omitempty"` // versión mínima de qdd que entiende la receta
	Licencia     string      `yaml:"licencia,omitempty"`
	Descripcion  string      `yaml:"descripcion,omitempty"`
	Alcance      Alcance     `yaml:"alcance,omitempty"`
	Vocabulario  []Termino   `yaml:"vocabulario,omitempty"`
	Parametros   []Parametro `yaml:"parametros,omitempty"`
	Constantes   []Constante `yaml:"constantes,omitempty"`
	Preguntas    []Pregunta  `yaml:"preguntas,omitempty"`
	Pasos        []Paso      `yaml:"pasos,omitempty"`
	Ficha        FichaBase   `yaml:"ficha"`
	Conformidad  []Escenario `yaml:"conformidad,omitempty"`
	Referencias  []string    `yaml:"referencias,omitempty"`
	Reemplaza    []string    `yaml:"reemplaza,omitempty"` // RFC 0009: ids de la misma fuente que este blueprint reemplaza

	Dir string `yaml:"-"`
}

var (
	idPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
)

// Ref identifica la versión exacta del blueprint, p. ej. "auth@1.2.0".
func (b Blueprint) Ref() string {
	return b.ID + "@" + b.Version
}

// Parse lee un blueprint desde YAML.
func Parse(data []byte) (Blueprint, error) {
	var b Blueprint
	if err := yaml.Unmarshal(data, &b); err != nil {
		return Blueprint{}, fmt.Errorf("YAML inválido: %w", err)
	}
	return b, nil
}

// ParseDir lee el blueprint de una carpeta (su blueprint.yaml).
func ParseDir(dir string) (Blueprint, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return Blueprint{}, fmt.Errorf("no se encontró %s en %s", ManifestName, dir)
	}
	b, err := Parse(data)
	if err != nil {
		return Blueprint{}, err
	}
	b.Dir = dir
	return b, nil
}

// Validate devuelve los problemas del blueprint; vacío significa válido.
func (b Blueprint) Validate() []string {
	var problems []string
	if !idPattern.MatchString(b.ID) {
		problems = append(problems, "id: obligatorio, en minúsculas, solo letras, números, '-' y '_' (p. ej. 'auth')")
	}
	if strings.TrimSpace(b.Nombre) == "" {
		problems = append(problems, "nombre: obligatorio")
	}
	if !semverPattern.MatchString(b.Version) {
		problems = append(problems, "version: obligatoria y en semver (p. ej. 1.0.0)")
	}
	problems = append(problems, b.validateFicha()...)
	problems = append(problems, b.validateVocabulario()...)
	problems = append(problems, b.validatePreguntas()...)
	problems = append(problems, b.validateEscenarios()...)
	problems = append(problems, b.validateReemplaza()...)
	problems = append(problems, b.validateReceta()...)
	return append(problems, b.validateReferencias()...)
}

func (b Blueprint) validateFicha() []string {
	var problems []string
	if strings.TrimSpace(b.Ficha.Nombre) == "" {
		problems = append(problems, "ficha.nombre: obligatorio")
	}
	if strings.TrimSpace(b.Ficha.ObjetivoOperativo.Problema) == "" {
		problems = append(problems, "ficha.objetivo_operativo.problema: obligatorio")
	}
	if strings.TrimSpace(b.Ficha.ObjetivoOperativo.CriterioDeExito) == "" {
		problems = append(problems, "ficha.objetivo_operativo.criterio_de_exito: obligatorio")
	}
	if strings.TrimSpace(b.Ficha.ObjetivoTecnologico.Arquitectura) == "" {
		problems = append(problems, "ficha.objetivo_tecnologico.arquitectura: obligatorio")
	}
	return problems
}

func (b Blueprint) validateVocabulario() []string {
	var problems []string
	seen := map[string]bool{}
	for i, t := range b.Vocabulario {
		key := strings.ToLower(strings.TrimSpace(t.Termino))
		if key == "" || strings.TrimSpace(t.Definicion) == "" {
			problems = append(problems, fmt.Sprintf("vocabulario[%d]: 'termino' y 'definicion' son obligatorios", i))
			continue
		}
		if seen[key] {
			problems = append(problems, fmt.Sprintf("vocabulario: el término '%s' está repetido", t.Termino))
		}
		seen[key] = true
	}
	return problems
}

func (b Blueprint) validatePreguntas() []string {
	var problems []string
	previas := map[string]Pregunta{}
	for i, p := range b.Preguntas {
		label := fmt.Sprintf("preguntas[%d] (%s)", i, p.ID)
		if !idPattern.MatchString(p.ID) {
			problems = append(problems, label+": id obligatorio, en minúsculas, solo letras, números, '-' y '_'")
		}
		if _, dup := previas[p.ID]; dup {
			problems = append(problems, label+": id repetido")
		}
		if strings.TrimSpace(p.Pregunta) == "" {
			problems = append(problems, label+": 'pregunta' es obligatoria")
		}
		problems = append(problems, validateOpciones(label, p)...)
		problems = append(problems, validateCondicion(label, p, previas)...)
		previas[p.ID] = p
	}
	return problems
}

func validateOpciones(label string, p Pregunta) []string {
	if len(p.Opciones) < 2 {
		return []string{label + ": declara al menos dos opciones (si no hay alternativa, es una invariante, no una pregunta)"}
	}
	var problems []string
	seen := map[string]bool{}
	for _, o := range p.Opciones {
		if !idPattern.MatchString(o.Valor) {
			problems = append(problems, fmt.Sprintf("%s: el valor '%s' debe ir en minúsculas, solo letras, números, '-' y '_'", label, o.Valor))
		}
		if seen[o.Valor] {
			problems = append(problems, fmt.Sprintf("%s: la opción '%s' está repetida", label, o.Valor))
		}
		seen[o.Valor] = true
	}
	if p.PorDefecto != "" && !seen[p.PorDefecto] {
		problems = append(problems, fmt.Sprintf("%s: por_defecto '%s' no es una de sus opciones", label, p.PorDefecto))
	}
	return problems
}

// validateCondicion exige que 'si' apunte a preguntas ANTERIORES y a valores
// que existan: así el orden de las preguntas siempre se puede resolver.
func validateCondicion(label string, p Pregunta, previas map[string]Pregunta) []string {
	var problems []string
	for id, valores := range p.Si {
		previa, ok := previas[id]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: 'si' depende de '%s', que no es una pregunta anterior", label, id))
			continue
		}
		for _, v := range valores {
			if _, ok := previa.opcion(v); ok {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s: 'si' espera '%s=%s', pero '%s' no tiene esa opción", label, id, v, id))
		}
	}
	return problems
}

// validateEscenarios exige ids únicos dentro de cada grupo. Un mismo escenario
// puede repetirse en opciones alternativas (p. ej. 'rut' y 'correo_o_rut'),
// pero no puede pisar uno de la base, que aplica siempre.
func (b Blueprint) validateEscenarios() []string {
	base := map[string]bool{}
	problems := checkEscenarios("conformidad", b.Conformidad, base, nil)
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			label := fmt.Sprintf("preguntas (%s=%s).conformidad", p.ID, o.Valor)
			problems = append(problems, checkEscenarios(label, o.Conformidad, map[string]bool{}, base)...)
		}
	}
	return problems
}

func checkEscenarios(label string, escenarios []Escenario, seen, reservados map[string]bool) []string {
	var problems []string
	for _, e := range escenarios {
		if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Cuando) == "" || strings.TrimSpace(e.Entonces) == "" {
			problems = append(problems, fmt.Sprintf("%s (%s): 'id', 'cuando' y 'entonces' son obligatorios", label, e.ID))
			continue
		}
		if seen[e.ID] || reservados[e.ID] {
			problems = append(problems, fmt.Sprintf("%s: el escenario '%s' está repetido", label, e.ID))
		}
		seen[e.ID] = true
	}
	return problems
}

func (b Blueprint) validateReferencias() []string {
	var problems []string
	for _, r := range b.Referencias {
		if !isInside(r) {
			problems = append(problems, fmt.Sprintf("referencias: '%s' debe ser una ruta relativa dentro del blueprint", r))
			continue
		}
		if b.Dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(b.Dir, r)); err != nil {
			problems = append(problems, fmt.Sprintf("referencias: '%s' no existe en el blueprint", r))
		}
	}
	return problems
}

func (p Pregunta) opcion(valor string) (Opcion, bool) {
	for _, o := range p.Opciones {
		if o.Valor == valor {
			return o, true
		}
	}
	return Opcion{}, false
}

// Valores devuelve los valores posibles de la pregunta.
func (p Pregunta) Valores() []string {
	out := make([]string, 0, len(p.Opciones))
	for _, o := range p.Opciones {
		out = append(out, o.Valor)
	}
	return out
}

// isInside indica si la ruta es relativa y no escapa de su carpeta base.
func isInside(p string) bool {
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) || clean == ".." || clean == "." {
		return false
	}
	return !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}
