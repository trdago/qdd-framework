package blueprints

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Rule: Prohibido usar 'else'
// Receta reproducible (pedido del usuario, 2026-10-31): QDD es quien genera los blueprints, así que asegura que cada
// uno se pueda construir paso a paso en cualquier proyecto y llegue a la misma funcionalidad. Para eso separa:
//   - lo técnico, que se replica igual: pasos en orden (acción → resultado, con el escenario que lo verifica),
//     decisiones, invariantes, escenarios y constantes (un valor técnico fijo, con su porqué);
//   - lo de negocio, que cada proyecto adapta: parámetros (nombres, textos, idioma, umbrales, cantidades) que la
//     persona decide al aplicar, como las preguntas de diseño;
//   - y los límites: qué incluye la receta y qué queda fuera.

// RequiereQDDReceta es la versión de qdd que entiende parámetros, constantes, pasos y alcance.
const RequiereQDDReceta = "2.8.0"

// VersionQDD es la versión de esta CLI (la fija cmd); un blueprint que pide una mayor no se aplica a medias.
var VersionQDD = ""

// ReglasDeReceta es lo que QDD le dice a quien destila o edita un blueprint.
const ReglasDeReceta = "Receta: técnico fijo (pasos en orden acción → resultado con el escenario que lo verifica; decisiones, invariantes, escenarios dado/cuando/entonces; constantes {{id}} con su porqué), sin código ni rutas del proyecto. Negocio adaptable: nombres, textos, idioma, umbrales y cantidades como parámetros {{id}} con por_defecto, que decide la persona. Alternativas de diseño: preguntas. Límites: alcance.incluye / alcance.excluye, preguntados a la persona hasta estar seguro."

// Alcance son los límites de la receta: lo que construye y lo que deja fuera a propósito.
type Alcance struct {
	Incluye []string `yaml:"incluye,omitempty"`
	Excluye []string `yaml:"excluye,omitempty"`
}

// Parametro es un valor de negocio que cada proyecto adapta (p. ej. cuántas tareas se ven): lo decide la persona al
// aplicar, con un valor sugerido; las reglas lo citan como {{id}}.
type Parametro struct {
	ID         string             `yaml:"id"`
	Si         map[string]Valores `yaml:"si,omitempty"` // solo se pregunta con esas respuestas
	Pregunta   string             `yaml:"pregunta"`
	Ayuda      string             `yaml:"ayuda,omitempty"`
	Tipo       string             `yaml:"tipo,omitempty"` // numero | texto (por defecto)
	PorDefecto string             `yaml:"por_defecto"`
	Ejemplos   []string           `yaml:"ejemplos,omitempty"`
}

// Constante es un valor técnico fijo de la receta, igual en todo proyecto, con la razón de ese valor.
type Constante struct {
	ID     string `yaml:"id"`
	Valor  string `yaml:"valor"`
	PorQue string `yaml:"por_que"`
}

// Paso es un paso de construcción: qué se hace, qué resultado observable deja y qué escenarios lo verifican.
type Paso struct {
	ID       string             `yaml:"id"`
	Si       map[string]Valores `yaml:"si,omitempty"`
	Accion   string             `yaml:"accion"`
	Produce  string             `yaml:"produce"`
	Verifica []string           `yaml:"verifica,omitempty"`
}

const tipoNumero = "numero"

// comoPregunta presenta el parámetro como una pregunta abierta, para pedírsela a la persona igual que las de diseño.
func (p Parametro) comoPregunta() Pregunta {
	ayuda := strings.TrimSpace(p.Ayuda)
	if len(p.Ejemplos) > 0 {
		ayuda = strings.TrimSpace(ayuda + " Ejemplos: " + strings.Join(p.Ejemplos, ", ") + ".")
	}
	return Pregunta{ID: p.ID, Pregunta: p.Pregunta, Ayuda: ayuda, PorDefecto: p.PorDefecto, abierta: true}
}

// valorValido dice si v sirve como valor del parámetro.
func (p Parametro) valorValido(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	if p.Tipo != tipoNumero {
		return true
	}
	_, err := strconv.ParseFloat(strings.Replace(strings.TrimSpace(v), ",", ".", 1), 64)
	return err == nil
}

func (b Blueprint) validateReceta() []string {
	ids := map[string]string{}
	for _, p := range b.Preguntas {
		ids[p.ID] = "pregunta"
	}
	var problems []string
	for i, p := range b.Parametros {
		label := fmt.Sprintf("parametros[%d] (%s)", i, p.ID)
		problems = append(problems, validarID(label, p.ID, "parámetro", ids)...)
		if strings.TrimSpace(p.Pregunta) == "" || strings.TrimSpace(p.PorDefecto) == "" {
			problems = append(problems, label+": 'pregunta' y 'por_defecto' son obligatorios (la persona decide, con un valor sugerido)")
		}
		if p.Tipo != "" && p.Tipo != tipoNumero && p.Tipo != "texto" {
			problems = append(problems, label+": 'tipo' es numero o texto")
		}
		if p.PorDefecto != "" && !p.valorValido(p.PorDefecto) {
			problems = append(problems, fmt.Sprintf("%s: por_defecto '%s' no es un número", label, p.PorDefecto))
		}
		problems = append(problems, validateCondicion(label, Pregunta{Si: p.Si}, preguntasPorID(b))...)
	}
	for i, c := range b.Constantes {
		label := fmt.Sprintf("constantes[%d] (%s)", i, c.ID)
		problems = append(problems, validarID(label, c.ID, "constante", ids)...)
		if strings.TrimSpace(c.Valor) == "" || strings.TrimSpace(c.PorQue) == "" {
			problems = append(problems, label+": 'valor' y 'por_que' son obligatorios (un valor técnico fijo dice por qué es ese)")
		}
	}
	problems = append(problems, b.validatePasos()...)
	if b.RequiereQDD != "" && !semverPattern.MatchString(strings.TrimPrefix(b.RequiereQDD, "v")) {
		problems = append(problems, "requiere_qdd: en semver (p. ej. 2.8.0)")
	}
	return append(problems, b.validateCitas(ids)...)
}

func preguntasPorID(b Blueprint) map[string]Pregunta {
	out := map[string]Pregunta{}
	for _, p := range b.Preguntas {
		out[p.ID] = p
	}
	return out
}

func validarID(label, id, que string, ids map[string]string) []string {
	if !idPattern.MatchString(id) {
		return []string{label + ": id obligatorio, en minúsculas, solo letras, números, '-' y '_'"}
	}
	if previo, dup := ids[id]; dup {
		return []string{fmt.Sprintf("%s: el id '%s' ya es de una %s", label, id, previo)}
	}
	ids[id] = que
	return nil
}

func (b Blueprint) validatePasos() []string {
	escenarios := map[string]bool{}
	for _, e := range todosLosEscenarios(b) {
		escenarios[e.ID] = true
	}
	preguntas := map[string]Pregunta{}
	for _, p := range b.Preguntas {
		preguntas[p.ID] = p
	}
	var problems []string
	vistos := map[string]bool{}
	for i, p := range b.Pasos {
		label := fmt.Sprintf("pasos[%d] (%s)", i, p.ID)
		if strings.TrimSpace(p.ID) == "" || vistos[p.ID] {
			problems = append(problems, label+": id obligatorio y único")
		}
		vistos[p.ID] = true
		if strings.TrimSpace(p.Accion) == "" || strings.TrimSpace(p.Produce) == "" {
			problems = append(problems, label+": 'accion' y 'produce' son obligatorios")
		}
		for _, id := range p.Verifica {
			if !escenarios[id] {
				problems = append(problems, fmt.Sprintf("%s: verifica '%s', que no es un escenario del blueprint", label, id))
			}
		}
		problems = append(problems, validateCondicion(label, Pregunta{Si: p.Si}, preguntas)...)
	}
	return problems
}

func todosLosEscenarios(b Blueprint) []Escenario {
	out := append([]Escenario{}, b.Conformidad...)
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			out = append(out, o.Conformidad...)
		}
	}
	return out
}

var cita = regexp.MustCompile(`\{\{\s*([^}\s]+)\s*\}\}`)

// validateCitas: cada {{x}} es una pregunta, un parámetro o una constante; si no, la ficha quedaría con {{x}} sin
// reemplazar.
func (b Blueprint) validateCitas(ids map[string]string) []string {
	var problems []string
	vistas := map[string]bool{}
	for _, t := range textosDe(b) {
		for _, m := range cita.FindAllStringSubmatch(t.texto, -1) {
			if _, ok := ids[m[1]]; ok || vistas[m[1]] {
				continue
			}
			vistas[m[1]] = true
			problems = append(problems, fmt.Sprintf("%s: cita {{%s}}, que no es una pregunta, un parámetro ni una constante", t.donde, m[1]))
		}
	}
	return problems
}

// valores reúne lo que reemplaza cada {{id}}: las respuestas (preguntas y parámetros) y las constantes.
func (b Blueprint) valores(respuestas map[string]string) map[string]string {
	out := map[string]string{}
	for _, c := range b.Constantes {
		out[c.ID] = strings.TrimSpace(c.Valor)
	}
	for k, v := range respuestas {
		out[k] = v
	}
	return out
}

// pasosAplicables son los pasos que corresponden a las respuestas, en orden.
func (b Blueprint) pasosAplicables(respuestas map[string]string) []Paso {
	var out []Paso
	for _, p := range b.Pasos {
		if condicion(Pregunta{Si: p.Si}, respuestas, nil) != condicionCumple {
			continue
		}
		out = append(out, p)
	}
	return out
}

// lineaDePaso es el paso tal como queda en la ficha: «Paso N · acción → resultado (verifica …)».
func lineaDePaso(n int, p Paso) string {
	verifica := ""
	if len(p.Verifica) > 0 {
		verifica = " (verifica " + strings.Join(p.Verifica, ", ") + ")"
	}
	return fmt.Sprintf("Paso %d · %s → %s%s", n, strings.TrimSpace(p.Accion), strings.TrimSpace(p.Produce), verifica)
}
