package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/modules"
	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Destilar (pedido del usuario, 2026-10-31): QDD genera el blueprint desde lo que ya funciona, la ficha aprobada de
// un módulo, y entrevista a la persona hasta delimitar exactamente qué es la receta: qué incluye y qué no, qué valor
// es de negocio (lo adapta cada proyecto) y cuál es técnico (fijo, con su porqué), y qué hace cada detalle de la
// implementación. La entrevista termina cuando no queda nada por delimitar; entonces la receta se certifica.

// Destilado es lo que dejó Destilar: el borrador en su aporte, su calidad y lo que hay que preguntarle a la persona.
type Destilado struct {
	Aporte     Aporte
	Calidad    Calidad
	Entrevista []string
	Siguiente  string
}

// Destilar arma el borrador del blueprint id desde la ficha aprobada del módulo, en un aporte nuevo.
func Destilar(root, modulo, id, origen string) (Destilado, error) {
	m, ok := modules.Find(root, modulo)
	if !ok {
		return Destilado{}, fmt.Errorf("el módulo '%s' no tiene ficha: escríbela y apruébala antes de destilarla", modulo)
	}
	if m.Estado != modules.EstadoAprobado {
		return Destilado{}, fmt.Errorf("la ficha '%s' está en %s: se destila solo lo aprobado (qdd modules approve %s)", modulo, m.Estado, modulo)
	}
	if m.Blueprint != "" {
		return Destilado{}, fmt.Errorf("'%s' ya se generó con %s: para mejorarlo, qdd blueprint contribute %s", modulo, m.Blueprint, strings.Split(m.Blueprint, "@")[0])
	}
	ap, err := Aportar(root, id, origen)
	if err != nil {
		return Destilado{}, err
	}
	manifest := filepath.Join(ap.Carpeta, ManifestName)
	if exists(manifest) {
		return Destilado{}, fmt.Errorf("'%s' ya existe: edítalo con qdd blueprint contribute %s", id, id)
	}
	b := borradorDe(m, id)
	if err := escribirBorrador(manifest, b); err != nil {
		return Destilado{}, err
	}
	b.Dir = ap.Carpeta
	c := CalidadDe(b)
	return Destilado{Aporte: ap, Calidad: c, Entrevista: Entrevista(c), Siguiente: SiguientePaso(c, id)}, nil
}

// borradorDe copia de la ficha lo que es receta. Las rutas, el caso real y los usuarios son del proyecto.
func borradorDe(m modules.Module, id string) Blueprint {
	return Blueprint{
		ID: id, Nombre: m.Nombre, Version: "1.0.0", RequiereQDD: RequiereQDDReceta,
		Descripcion: primeraFrase(m.ObjetivoOperativo.Problema),
		Ficha: FichaBase{Nombre: m.Nombre,
			ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: m.ObjetivoOperativo.Problema, CriterioDeExito: m.ObjetivoOperativo.CriterioDeExito},
			ObjetivoTecnologico: m.ObjetivoTecnologico},
	}
}

func escribirBorrador(manifest string, b Blueprint) error {
	data, err := yaml.Marshal(b)
	if err != nil {
		return err
	}
	cabecera := "# Borrador destilado por 'qdd blueprint distill' desde una ficha aprobada.\n# " + ReglasDeReceta + "\n"
	if err := os.MkdirAll(filepath.Dir(manifest), 0755); err != nil {
		return err
	}
	return os.WriteFile(manifest, append([]byte(cabecera), data...), 0644)
}

// preguntaDe dice qué preguntarle a la persona por cada hallazgo que solo ella puede resolver; los demás los arregla
// quien escribe la receta.
var preguntaDe = map[string]func(HallazgoCalidad) string{
	CalSinAlcance: func(HallazgoCalidad) string {
		return "¿Qué incluye exactamente este estándar y qué queda fuera a propósito? (alcance.incluye / alcance.excluye)"
	},
	CalValorFijo: func(h HallazgoCalidad) string {
		return h.Detalle + " → ¿lo decide cada proyecto (negocio) o es fijo por una razón técnica? ¿cuál?"
	},
	CalImplementacion: func(h HallazgoCalidad) string {
		return h.Detalle + " → ¿qué hace, en palabras de la receta (acción y resultado)?"
	},
	CalOpcionVacia: func(h HallazgoCalidad) string {
		return "¿Qué implica elegir " + h.Donde + "?"
	},
}

// Entrevista son las preguntas que quedan para la persona; vacía, la receta está delimitada.
// Un mismo valor escrito en varias reglas se pregunta una vez.
func Entrevista(c Calidad) []string {
	var out []string
	vistas := map[string]bool{}
	for _, h := range c.Hallazgos {
		pregunta, ok := preguntaDe[h.Codigo]
		clave := h.Codigo + " " + strings.SplitN(h.Detalle, "»", 2)[0]
		if !ok || vistas[clave] {
			continue
		}
		vistas[clave] = true
		out = append(out, pregunta(h))
	}
	return out
}

// SiguientePaso dice qué hacer con la receta según cómo está.
func SiguientePaso(c Calidad, id string) string {
	if len(Entrevista(c)) > 0 {
		return "Pregúntale a la persona lo de la entrevista (no lo respondas tú), escribe sus respuestas en la receta y vuelve a medir; repite hasta que no quede nada que delimitar."
	}
	if c.Defectos > 0 {
		return "Arregla los ⚠ (pasos, escenarios) y vuelve a medir."
	}
	return "Delimitada: certifícala a ciegas (qdd blueprint certify " + id + ") y, con el OK de la persona, propose."
}
