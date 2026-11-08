package flujo

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Rule: Prohibido usar 'else'

// Historias de usuario del flujo (RFC 0025): «como … quiero … para …», con sus criterios y los nodos que tocan. Las
// escribe la persona, o un invitado o el asistente las proponen y la persona las aprueba; solo las aprobadas se guardan.
// Origen dice de dónde salió («persona», «cliente:Ana», «transcripcion@12:40») y Tarea, la tarea del tablero que la
// implementa. Ninguno de los dos viaja en la foto del cliente.

const (
	maxHistoria         = 500
	maxCriterios        = 10
	maxNodosDeHistoria  = 20
	maxHistorias        = 100
	maxOrigenDeHistoria = 120
	prefijoDeHistoria   = "h"
)

// Historia es una historia de usuario aprobada.
type Historia struct {
	ID        string   `yaml:"id" json:"id"`
	Como      string   `yaml:"como,omitempty" json:"como,omitempty"`
	Quiero    string   `yaml:"quiero" json:"quiero"`
	Para      string   `yaml:"para,omitempty" json:"para,omitempty"`
	Criterios []string `yaml:"criterios,omitempty" json:"criterios,omitempty"`
	Nodos     []string `yaml:"nodos,omitempty" json:"nodos,omitempty"`
	Origen    string   `yaml:"origen,omitempty" json:"origen,omitempty"`
	Tarea     string   `yaml:"tarea,omitempty" json:"tarea,omitempty"`
}

// Frase es la historia en una línea: «como vendedor, quiero …, para …».
func (h Historia) Frase() string {
	partes := []string{"quiero " + h.Quiero}
	if h.Como != "" {
		partes = slices.Insert(partes, 0, "como "+h.Como)
	}
	if h.Para != "" {
		partes = append(partes, "para "+h.Para)
	}
	return strings.Join(partes, ", ")
}

// AgregarHistoria suma la historia con el id que sigue. La IA la lee entera: son las palabras de quien la pidió.
func AgregarHistoria(h Historia, origen string) Cambio {
	return func(f *Flujo) (hecho, error) {
		if len(f.Historias) >= maxHistorias {
			return hecho{}, fmt.Errorf("el flujo ya tiene %d historias: el tope es %d", len(f.Historias), maxHistorias)
		}
		h, err := historiaLimpia(h)
		if err != nil {
			return hecho{}, err
		}
		h.ID, h.Origen, h.Tarea = idDeHistoria(f.Historias), acotar(limpiar(origen), maxOrigenDeHistoria), ""
		if problemas := problemasDeHistoria(*f, h); len(problemas) > 0 {
			return hecho{}, errors.New(strings.Join(problemas, "; "))
		}
		f.Historias = append(f.Historias, h)
		linea := fmt.Sprintf("📖 historia %s: %s", h.ID, h.Frase())
		if len(h.Criterios) > 0 {
			linea += " · criterios: " + strings.Join(h.Criterios, "; ")
		}
		if len(h.Nodos) > 0 {
			linea += " · en " + strings.Join(h.Nodos, ", ")
		}
		return hecho{linea: linea, entera: true,
			pista: fmt.Sprintf("Para implementarla, crea su tarea con: qdd flujo historia tarea %s %s", f.ID, h.ID)}, nil
	}
}

// historiaLimpia quita lo que no se ve de cada texto y descarta los nodos repetidos.
func historiaLimpia(h Historia) (Historia, error) {
	h.Como, h.Quiero, h.Para = limpiar(h.Como), limpiar(h.Quiero), limpiar(h.Para)
	criterios := make([]string, 0, len(h.Criterios))
	for _, c := range h.Criterios {
		c = limpiar(c)
		if c == "" {
			return Historia{}, errors.New("un criterio está vacío")
		}
		criterios = append(criterios, c)
	}
	h.Criterios = criterios
	nodos := []string{}
	for _, n := range h.Nodos {
		nodos = sumarSalida(nodos, n, "")
	}
	h.Nodos = nodos
	return h, nil
}

// QuitarHistoria saca la historia; su tarea, si tiene, sigue en el tablero.
func QuitarHistoria(id string) Cambio {
	return func(f *Flujo) (hecho, error) {
		i := slices.IndexFunc(f.Historias, func(h Historia) bool { return h.ID == id })
		if i < 0 {
			return hecho{}, fmt.Errorf("el flujo no tiene la historia %q", id)
		}
		h := f.Historias[i]
		f.Historias = slices.Delete(f.Historias, i, i+1)
		return hecho{linea: fmt.Sprintf("− historia %s: %s", h.ID, h.Frase())}, nil
	}
}

// TareaDeHistoria anota la tarea del tablero que implementa la historia.
func TareaDeHistoria(id, tarea string) Cambio {
	return func(f *Flujo) (hecho, error) {
		i := slices.IndexFunc(f.Historias, func(h Historia) bool { return h.ID == id })
		if i < 0 {
			return hecho{}, fmt.Errorf("el flujo no tiene la historia %q", id)
		}
		f.Historias[i].Tarea = tarea
		return hecho{linea: fmt.Sprintf("historia %s → tarea %s", id, tarea)}, nil
	}
}

// HistoriaDe busca la historia por su id.
func HistoriaDe(f Flujo, id string) (Historia, error) {
	i := slices.IndexFunc(f.Historias, func(h Historia) bool { return h.ID == id })
	if i < 0 {
		return Historia{}, fmt.Errorf("el flujo %q no tiene la historia %q", f.ID, id)
	}
	return f.Historias[i], nil
}

// idDeHistoria es h<n> con n uno más que el mayor del flujo.
func idDeHistoria(historias []Historia) string {
	mayor := 0
	for _, h := range historias {
		n, err := strconv.Atoi(strings.TrimPrefix(h.ID, prefijoDeHistoria))
		if err == nil && n > mayor {
			mayor = n
		}
	}
	return prefijoDeHistoria + strconv.Itoa(mayor+1)
}

// sinNodoEnHistorias saca el nodo quitado de las historias que lo nombraban: la historia sigue valiendo.
func sinNodoEnHistorias(f *Flujo, nodo string) {
	for i := range f.Historias {
		f.Historias[i].Nodos = slices.DeleteFunc(f.Historias[i].Nodos, func(n string) bool { return n == nodo })
	}
}

func problemasDeHistorias(f Flujo) []string {
	var problemas []string
	if len(f.Historias) > maxHistorias {
		problemas = append(problemas, fmt.Sprintf("%d historias: el tope es %d", len(f.Historias), maxHistorias))
	}
	vistas := map[string]bool{}
	for _, h := range f.Historias {
		if vistas[h.ID] {
			problemas = append(problemas, fmt.Sprintf("id de historia repetido: %q", h.ID))
		}
		vistas[h.ID] = true
		problemas = append(problemas, problemasDeHistoria(f, h)...)
	}
	return problemas
}

func problemasDeHistoria(f Flujo, h Historia) []string {
	var problemas []string
	if !idValido.MatchString(h.ID) {
		problemas = append(problemas, fmt.Sprintf("id de historia %q: usa minúsculas, números y guiones", h.ID))
	}
	if strings.TrimSpace(h.Quiero) == "" {
		problemas = append(problemas, fmt.Sprintf("la historia %q no dice qué se quiere", h.ID))
	}
	for _, texto := range append([]string{h.Como, h.Quiero, h.Para}, h.Criterios...) {
		if largo := len([]rune(texto)); largo > maxHistoria {
			problemas = append(problemas, fmt.Sprintf("la historia %q tiene un texto de %d caracteres: el tope es %d", h.ID, largo, maxHistoria))
		}
	}
	if len(h.Criterios) > maxCriterios || len(h.Nodos) > maxNodosDeHistoria {
		problemas = append(problemas, fmt.Sprintf("la historia %q tiene %d criterios y %d nodos: el tope es %d y %d", h.ID, len(h.Criterios), len(h.Nodos), maxCriterios, maxNodosDeHistoria))
	}
	for _, c := range h.Criterios {
		if strings.TrimSpace(c) == "" {
			problemas = append(problemas, fmt.Sprintf("la historia %q tiene un criterio vacío", h.ID))
		}
	}
	for _, n := range h.Nodos {
		if !slices.ContainsFunc(f.Nodos, func(x Nodo) bool { return x.ID == n }) {
			problemas = append(problemas, fmt.Sprintf("la historia %q nombra el nodo %q, que no existe", h.ID, n))
		}
	}
	return problemas
}

// historiasSinNodosQuitados conserva las historias de un flujo reescrito, sin los nodos que ya no están.
func historiasSinNodosQuitados(historias []Historia, nodos []Nodo) []Historia {
	f := Flujo{Nodos: nodos, Historias: slices.Clone(historias)}
	for _, h := range historias {
		for _, n := range h.Nodos {
			if !slices.ContainsFunc(nodos, func(x Nodo) bool { return x.ID == n }) {
				sinNodoEnHistorias(&f, n)
			}
		}
	}
	return f.Historias
}
