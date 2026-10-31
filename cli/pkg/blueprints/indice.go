package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Índice de un blueprint (pedido del usuario, 2026-10-31): leer un blueprint no puede costar miles de tokens. El índice
// es lo esencial en un JSON corto (identidad, preguntas con sus opciones, cuántas invariantes, qué escenarios) y una
// línea por blueprint para listarlos; el detalle se pide por sección. Se calcula del contenido ya verificado por su
// sha256, así que nunca queda desactualizado ni deja archivos en el store.

const (
	// MaxLineaIndice es el tope de la línea con que se lista un blueprint.
	MaxLineaIndice = 140
	maxResumen     = 100
	maxPregunta    = 80
)

// SeccionesDeBlueprint son las secciones que se piden por separado (qdd_blueprint_list con seccion).
var SeccionesDeBlueprint = []string{"preguntas", "invariantes", "decisiones", "escenarios", "vocabulario", "todo"}

// Indice es el resumen de un blueprint que lee una IA antes de pedir detalle.
type Indice struct {
	Ref         string           `json:"ref"`
	ID          string           `json:"id"`
	Version     string           `json:"version"`
	Nombre      string           `json:"nombre"`
	Resumen     string           `json:"resumen,omitempty"`
	Preguntas   []IndicePregunta `json:"preguntas,omitempty"`
	Invariantes int              `json:"invariantes"`
	Escenarios  []string         `json:"escenarios,omitempty"`
	Modulos     []string         `json:"modulos,omitempty"`   // módulos del proyecto generados con él
	Secciones   map[string]int   `json:"secciones,omitempty"` // línea de cada sección en blueprint.yaml
}

// IndicePregunta es una pregunta de diseño sin su ayuda ni lo que implica cada opción.
type IndicePregunta struct {
	ID         string   `json:"id"`
	Pregunta   string   `json:"pregunta"`
	Opciones   []string `json:"opciones"`
	PorDefecto string   `json:"por_defecto,omitempty"`
	Si         string   `json:"si,omitempty"`
}

// IndiceDe resume el blueprint. Si se leyó de una carpeta, anota la línea de cada sección de su manifiesto.
func IndiceDe(b Blueprint) Indice {
	i := Indice{Ref: b.Ref(), ID: b.ID, Version: b.Version, Nombre: b.Nombre, Resumen: recortar(primeraFrase(primero(b.Descripcion, b.Ficha.ObjetivoOperativo.Problema)), maxResumen)}
	invariantes := dedupe(b.Ficha.ObjetivoTecnologico.Invariantes)
	i.Escenarios = idsDe(b.Conformidad, nil)
	for _, p := range b.Preguntas {
		i.Preguntas = append(i.Preguntas, IndicePregunta{ID: p.ID, Pregunta: recortar(p.Pregunta, maxPregunta), Opciones: p.Valores(), PorDefecto: p.PorDefecto, Si: describeSi(p)})
		for _, o := range p.Opciones {
			invariantes = dedupe(append(invariantes, o.Invariantes...)) // una invariante compartida (ancla YAML) cuenta una vez (FND-093)
			i.Escenarios = idsDe(o.Conformidad, i.Escenarios)
		}
	}
	i.Invariantes = len(invariantes)
	i.Secciones = lineasDeSecciones(b.Dir)
	return i
}

// LineaIndice lista un blueprint en una línea: ref · nombre · ❓preguntas 🔒invariantes ✓escenarios · dónde se aplicó.
func LineaIndice(i Indice) string {
	donde := "sin aplicar"
	if len(i.Modulos) > 0 {
		donde = "en " + strings.Join(i.Modulos, ", ")
	}
	l := fmt.Sprintf("%s · %s · ❓%d 🔒%d ✓%d · %s", i.Ref, recortar(i.Nombre, 40), len(i.Preguntas), i.Invariantes, len(i.Escenarios), donde)
	return recortar(l, MaxLineaIndice)
}

// Seccion devuelve el detalle de una sección del blueprint; "todo" es la descripción completa.
func Seccion(b Blueprint, nombre string) (string, error) {
	var s strings.Builder
	switch strings.TrimSpace(nombre) {
	case "todo":
		return Describe(b), nil
	case "preguntas":
		for _, p := range b.Preguntas {
			s.WriteString(DescribePregunta(p))
		}
	case "invariantes":
		escribirPorOpcion(&s, b.Ficha.ObjetivoTecnologico.Invariantes, b.Preguntas, func(o Opcion) []string { return o.Invariantes })
	case "decisiones":
		escribirPorOpcion(&s, b.Ficha.ObjetivoTecnologico.Decisiones, b.Preguntas, func(o Opcion) []string { return o.Decisiones })
	case "escenarios":
		escribirEscenarios(&s, "", b.Conformidad)
		for _, p := range b.Preguntas {
			for _, o := range p.Opciones {
				escribirEscenarios(&s, p.ID+"="+o.Valor+" → ", o.Conformidad)
			}
		}
	case "vocabulario":
		for _, t := range b.Vocabulario {
			fmt.Fprintf(&s, "- %s: %s\n", t.Termino, strings.TrimSpace(t.Definicion))
		}
	default:
		return "", fmt.Errorf("sección '%s' desconocida: usa %s", nombre, strings.Join(SeccionesDeBlueprint, ", "))
	}
	if s.Len() == 0 {
		return "(" + nombre + ": vacío)\n", nil
	}
	return s.String(), nil
}

func escribirPorOpcion(s *strings.Builder, base []string, preguntas []Pregunta, de func(Opcion) []string) {
	for _, x := range base {
		fmt.Fprintf(s, "- %s\n", x)
	}
	for _, p := range preguntas {
		for _, o := range p.Opciones {
			for _, x := range de(o) {
				fmt.Fprintf(s, "- %s=%s → %s\n", p.ID, o.Valor, x)
			}
		}
	}
}

func escribirEscenarios(s *strings.Builder, prefijo string, escenarios []Escenario) {
	for _, e := range escenarios {
		dado := ""
		if e.Dado != "" {
			dado = "dado " + strings.TrimSpace(e.Dado) + "; "
		}
		fmt.Fprintf(s, "- %s%s %s: %scuando %s → %s\n", prefijo, e.ID, e.Titulo, dado, strings.TrimSpace(e.Cuando), strings.TrimSpace(e.Entonces))
	}
}

func idsDe(escenarios []Escenario, ya []string) []string {
	for _, e := range escenarios {
		if !inList(ya, e.ID) {
			ya = append(ya, e.ID)
		}
	}
	return ya
}

// lineasDeSecciones ubica cada sección en el blueprint.yaml de dir (sin carpeta, no hay líneas).
func lineasDeSecciones(dir string) map[string]int {
	if dir == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return nil
	}
	var doc yaml.Node
	if yaml.Unmarshal(data, &doc) != nil || len(doc.Content) == 0 {
		return nil
	}
	raiz := doc.Content[0]
	out := map[string]int{}
	anotar(out, raiz, "vocabulario", "vocabulario")
	anotar(out, raiz, "preguntas", "preguntas")
	anotar(out, raiz, "conformidad", "escenarios")
	tec := valorDe(valorDe(raiz, "ficha"), "objetivo_tecnologico")
	anotar(out, tec, "invariantes", "invariantes")
	anotar(out, tec, "decisiones", "decisiones")
	return out
}

func anotar(out map[string]int, m *yaml.Node, clave, seccion string) {
	if k := claveDe(m, clave); k != nil {
		out[seccion] = k.Line
	}
}

func claveDe(m *yaml.Node, clave string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == clave {
			return m.Content[i]
		}
	}
	return nil
}

func valorDe(m *yaml.Node, clave string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == clave {
			return m.Content[i+1]
		}
	}
	return nil
}

func primero(valores ...string) string {
	for _, v := range valores {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// primeraFrase corta en el primer punto seguido de espacio o salto: el resumen es una frase.
func primeraFrase(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i+1]
	}
	return s
}

func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// MapaDeSecciones dice en una línea dónde está cada sección del blueprint.yaml de dir, en orden: así un aporte edita
// la sección justa sin leer el archivo entero. Sin manifiesto, "".
func MapaDeSecciones(dir string) string {
	lineas := lineasDeSecciones(dir)
	if len(lineas) == 0 {
		return ""
	}
	nombres := make([]string, 0, len(lineas))
	for n := range lineas {
		nombres = append(nombres, n)
	}
	sort.Slice(nombres, func(a, b int) bool { return lineas[nombres[a]] < lineas[nombres[b]] })
	partes := make([]string, 0, len(nombres))
	for _, n := range nombres {
		partes = append(partes, fmt.Sprintf("%s L%d", n, lineas[n]))
	}
	return ManifestName + ": " + strings.Join(partes, " · ")
}
