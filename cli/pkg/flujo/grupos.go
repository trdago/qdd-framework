package flujo

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Rule: Prohibido usar 'else'
// Grupos con clase (pedido de la persona, 2026-11-08: «un [] que agrupa nodos … como cola, ambiente, fase»). Los nodos
// siguen nombrando su grupo en Nodo.Grupo; Flujo.Grupos le da a ese nombre una clase (libre; la plantilla sugiere
// algunas) y una descripción. Un grupo sin definir es un grupo sin clase, como antes.

const maxClase = 30

// Grupo es la definición de un grupo de nodos.
type Grupo struct {
	Nombre      string `yaml:"nombre" json:"nombre"`
	Clase       string `yaml:"clase,omitempty" json:"clase,omitempty"`
	Descripcion string `yaml:"descripcion,omitempty" json:"descripcion,omitempty"`
}

// GrupoDe busca la definición de un grupo por su nombre.
func GrupoDe(f Flujo, nombre string) (Grupo, bool) {
	i := slices.IndexFunc(f.Grupos, func(g Grupo) bool { return g.Nombre == nombre })
	if i < 0 {
		return Grupo{Nombre: nombre}, false
	}
	return f.Grupos[i], true
}

// Etiqueta es «[fase] Ingreso», o el nombre solo si no tiene clase.
func (g Grupo) Etiqueta() string {
	if g.Clase == "" {
		return g.Nombre
	}
	return "[" + g.Clase + "] " + g.Nombre
}

// DefinirGrupo le da clase y descripción a un grupo; si ya estaba definido, lo cambia.
func DefinirGrupo(nombre, clase, descripcion string) Cambio {
	return func(f *Flujo) (hecho, error) {
		g := Grupo{Nombre: limpiar(nombre), Clase: limpiar(clase), Descripcion: limpiar(descripcion)}
		if g.Nombre == "" {
			return hecho{}, fmt.Errorf("el grupo no tiene nombre")
		}
		i := slices.IndexFunc(f.Grupos, func(o Grupo) bool { return o.Nombre == g.Nombre })
		if i >= 0 {
			f.Grupos[i] = g
			return hecho{linea: "~ grupo " + g.Etiqueta()}, nil
		}
		f.Grupos = append(f.Grupos, g)
		return hecho{linea: "+ grupo " + g.Etiqueta()}, nil
	}
}

func problemasDeGrupos(f Flujo) []string {
	var problemas []string
	vistos := map[string]bool{}
	for _, g := range f.Grupos {
		if strings.TrimSpace(g.Nombre) == "" {
			problemas = append(problemas, "hay un grupo sin nombre")
		}
		if vistos[g.Nombre] {
			problemas = append(problemas, fmt.Sprintf("grupo repetido: %q", g.Nombre))
		}
		vistos[g.Nombre] = true
		if utf8.RuneCountInString(g.Clase) > maxClase {
			problemas = append(problemas, fmt.Sprintf("el grupo %q tiene una clase de más de %d letras", g.Nombre, maxClase))
		}
	}
	return problemas
}
