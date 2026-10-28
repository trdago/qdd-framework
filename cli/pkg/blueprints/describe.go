package blueprints

import (
	"fmt"
	"sort"
	"strings"
)

// Describe presenta el blueprint para que un humano (o una IA) lo entienda y
// pueda responder sus preguntas.
func Describe(b Blueprint) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s (`%s`)\n", b.Nombre, b.Ref())
	if b.Organizacion != "" {
		fmt.Fprintf(&s, "Organización: %s", b.Organizacion)
		if b.Licencia != "" {
			fmt.Fprintf(&s, " · Licencia: %s", b.Licencia)
		}
		s.WriteString("\n")
	}
	if b.Descripcion != "" {
		fmt.Fprintf(&s, "\n%s\n", strings.TrimSpace(b.Descripcion))
	}
	fmt.Fprintf(&s, "\n**Problema:** %s\n**Criterio de éxito:** %s\n", strings.TrimSpace(b.Ficha.ObjetivoOperativo.Problema), strings.TrimSpace(b.Ficha.ObjetivoOperativo.CriterioDeExito))
	if len(b.Vocabulario) > 0 {
		s.WriteString("\n## Vocabulario\n")
		for _, t := range b.Vocabulario {
			fmt.Fprintf(&s, "- **%s**: %s\n", t.Termino, strings.TrimSpace(t.Definicion))
		}
	}
	if len(b.Preguntas) > 0 {
		s.WriteString("\n## Preguntas de diseño\n")
		for _, p := range b.Preguntas {
			s.WriteString(DescribePregunta(p))
		}
	}
	if len(b.Ficha.ObjetivoTecnologico.Invariantes) > 0 {
		s.WriteString("\n## Invariantes de base\n")
		for _, inv := range b.Ficha.ObjetivoTecnologico.Invariantes {
			fmt.Fprintf(&s, "- %s\n", inv)
		}
	}
	return s.String()
}

// DescribePregunta presenta una pregunta con sus opciones y de qué depende cada una.
func DescribePregunta(p Pregunta) string {
	var s strings.Builder
	fmt.Fprintf(&s, "\n### `%s`: %s\n", p.ID, p.Pregunta)
	if cond := describeSi(p); cond != "" {
		fmt.Fprintf(&s, "_Solo si %s._\n", cond)
	}
	if p.Ayuda != "" {
		fmt.Fprintf(&s, "%s\n", strings.TrimSpace(p.Ayuda))
	}
	for _, o := range p.Opciones {
		marca := ""
		if o.Valor == p.PorDefecto {
			marca = " (sugerida)"
		}
		fmt.Fprintf(&s, "- `%s`%s: %s\n", o.Valor, marca, strings.TrimSpace(o.Descripcion))
		for _, d := range o.DependeDe {
			fmt.Fprintf(&s, "  - depende de: %s\n", d)
		}
	}
	return s.String()
}

// Condicion describe de qué respuestas depende la pregunta ("" si siempre aplica).
func (p Pregunta) Condicion() string {
	return describeSi(p)
}

func describeSi(p Pregunta) string {
	ids := make([]string, 0, len(p.Si))
	for id := range p.Si {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%s = %s", id, strings.Join(p.Si[id], " o ")))
	}
	return strings.Join(parts, " y ")
}
