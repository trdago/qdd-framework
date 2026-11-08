package flujo

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// Ítems de un nodo (pedido de la persona, 2026-11-07). Al probar con qdd-human son las pruebas de ese paso, cada una
// pendiente, pasada o fallada con su evidencia; en cualquier flujo sirven para notas, características y funciones. Se
// agregan desde el panel o el teléfono, como cualquier otra edición.

const (
	ItemPrueba         = "prueba"
	ItemNota           = "nota"
	ItemCaracteristica = "caracteristica"
	ItemFuncion        = "funcion"

	PruebaPendiente = "pendiente"
	PruebaPasa      = "pasa"
	PruebaFalla     = "falla"

	maxItem          = 500
	maxItemsPorNodo  = 50
	maxEvidenciaItem = 300
)

var tiposDeItem = []string{ItemPrueba, ItemNota, ItemCaracteristica, ItemFuncion}

var estadosDePrueba = []string{PruebaPendiente, PruebaPasa, PruebaFalla}

// Item es una prueba, una nota, una característica o una función de un nodo. Estado, Evidencia y Cuando son de una prueba.
type Item struct {
	ID        string `yaml:"id" json:"id"`
	Tipo      string `yaml:"tipo" json:"tipo"`
	Texto     string `yaml:"texto" json:"texto"`
	Estado    string `yaml:"estado,omitempty" json:"estado,omitempty"`
	Evidencia string `yaml:"evidencia,omitempty" json:"evidencia,omitempty"`
	Cuando    string `yaml:"cuando,omitempty" json:"cuando,omitempty"`
}

// AgregarItem suma un ítem al nodo; una prueba nace pendiente. La IA ve qué hay que probar y cómo marcarlo.
func AgregarItem(nodo, tipo, texto string, ahora time.Time) Cambio {
	return func(f *Flujo) (hecho, error) {
		if !slices.Contains(tiposDeItem, tipo) {
			return hecho{}, fmt.Errorf("tipo de ítem %q: usa %s", tipo, strings.Join(tiposDeItem, ", "))
		}
		texto = limpiar(texto)
		if texto == "" {
			return hecho{}, fmt.Errorf("el ítem está vacío")
		}
		if largo := len([]rune(texto)); largo > maxItem {
			return hecho{}, fmt.Errorf("el ítem tiene %d caracteres: el tope es %d", largo, maxItem)
		}
		i, err := indiceDe(f, nodo)
		if err != nil {
			return hecho{}, err
		}
		n := &f.Nodos[i]
		if len(n.Items) >= maxItemsPorNodo {
			return hecho{}, fmt.Errorf("el nodo %q ya tiene %d ítems: el tope es %d", nodo, len(n.Items), maxItemsPorNodo)
		}
		it := Item{ID: idDeItem(n.Items), Tipo: tipo, Texto: texto}
		if tipo == ItemPrueba {
			it.Estado = PruebaPendiente
		}
		n.Items = append(n.Items, it)
		h := hecho{linea: fmt.Sprintf("☐ %s «%s»: %s nueva %s: %s", n.ID, n.Nombre, tipo, it.ID, texto), entera: true}
		if tipo == ItemPrueba {
			h.pista = fmt.Sprintf("Al probarla, márcala con: qdd flujo item marcar %s %s %s pasa|falla --evidencia <run o commit>", f.ID, n.ID, it.ID)
		}
		return h, nil
	}
}

// idDeItem es i<n> con n uno más que el mayor del nodo.
func idDeItem(items []Item) string {
	mayor := 0
	for _, it := range items {
		n, err := strconv.Atoi(strings.TrimPrefix(it.ID, "i"))
		if err == nil && n > mayor {
			mayor = n
		}
	}
	return "i" + strconv.Itoa(mayor+1)
}

// MarcarItem pone el estado de una prueba. La evidencia es de esta marca: una marca nueva la reemplaza.
func MarcarItem(nodo, item, estado, evidencia string, ahora time.Time) Cambio {
	return func(f *Flujo) (hecho, error) {
		if !slices.Contains(estadosDePrueba, estado) {
			return hecho{}, fmt.Errorf("estado %q: usa %s", estado, strings.Join(estadosDePrueba, ", "))
		}
		evidencia = limpiar(evidencia)
		if largo := len([]rune(evidencia)); largo > maxEvidenciaItem {
			return hecho{}, fmt.Errorf("la evidencia tiene %d caracteres: el tope es %d", largo, maxEvidenciaItem)
		}
		n, it, err := itemDe(f, nodo, item)
		if err != nil {
			return hecho{}, err
		}
		if it.Tipo != ItemPrueba {
			return hecho{}, fmt.Errorf("el ítem %s de %q es %s: solo una prueba pasa o falla", item, nodo, it.Tipo)
		}
		it.Estado, it.Evidencia, it.Cuando = estado, evidencia, ahora.UTC().Format(time.RFC3339)
		return hecho{linea: fmt.Sprintf("%s %s «%s» %s %s: %s", simboloDePrueba(estado), n.ID, n.Nombre, it.ID, estado, it.Texto), entera: true}, nil
	}
}

func simboloDePrueba(estado string) string {
	if estado == PruebaPasa {
		return "✓"
	}
	if estado == PruebaFalla {
		return "✗"
	}
	return "☐"
}

// QuitarItem saca un ítem del nodo.
func QuitarItem(nodo, item string) Cambio {
	return func(f *Flujo) (hecho, error) {
		n, it, err := itemDe(f, nodo, item)
		if err != nil {
			return hecho{}, err
		}
		texto := it.Texto
		n.Items = slices.DeleteFunc(n.Items, func(x Item) bool { return x.ID == item })
		return hecho{linea: fmt.Sprintf("− %s «%s» quitó %s: %s", n.ID, n.Nombre, item, texto)}, nil
	}
}

func itemDe(f *Flujo, nodo, item string) (*Nodo, *Item, error) {
	i, err := indiceDe(f, nodo)
	if err != nil {
		return nil, nil, err
	}
	n := &f.Nodos[i]
	j := slices.IndexFunc(n.Items, func(x Item) bool { return x.ID == item })
	if j < 0 {
		return nil, nil, fmt.Errorf("el nodo %q no tiene un ítem %q", nodo, item)
	}
	return n, &n.Items[j], nil
}

func problemasDeItems(nodo string, items []Item) []string {
	var problemas []string
	vistos := map[string]bool{}
	for _, it := range items {
		if vistos[it.ID] || !idValido.MatchString(it.ID) {
			problemas = append(problemas, fmt.Sprintf("el nodo %q tiene un ítem con id %q repetido o inválido", nodo, it.ID))
		}
		vistos[it.ID] = true
		problemas = append(problemas, problemasDeItem(nodo, it)...)
	}
	return problemas
}

func problemasDeItem(nodo string, it Item) []string {
	if !slices.Contains(tiposDeItem, it.Tipo) {
		return []string{fmt.Sprintf("el ítem %s de %q es de tipo %q: usa %s", it.ID, nodo, it.Tipo, strings.Join(tiposDeItem, ", "))}
	}
	if strings.TrimSpace(it.Texto) == "" {
		return []string{fmt.Sprintf("el ítem %s de %q está vacío", it.ID, nodo)}
	}
	if it.Tipo != ItemPrueba && it.Estado != "" {
		return []string{fmt.Sprintf("el ítem %s de %q es %s: solo una prueba tiene estado", it.ID, nodo, it.Tipo)}
	}
	if it.Tipo == ItemPrueba && !slices.Contains(estadosDePrueba, it.Estado) {
		return []string{fmt.Sprintf("la prueba %s de %q tiene estado %q: usa %s", it.ID, nodo, it.Estado, strings.Join(estadosDePrueba, ", "))}
	}
	return nil
}

// contarPruebas suma las pruebas de todos los nodos por estado; nil si el flujo no tiene ninguna.
func contarPruebas(f Flujo) *ConteoPruebas {
	var c ConteoPruebas
	for _, n := range f.Nodos {
		for _, it := range n.Items {
			if it.Tipo != ItemPrueba {
				continue
			}
			c.Pasa += btoi(it.Estado == PruebaPasa)
			c.Falla += btoi(it.Estado == PruebaFalla)
			c.Pendiente += btoi(it.Estado == PruebaPendiente)
		}
	}
	if c == (ConteoPruebas{}) {
		return nil
	}
	return &c
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
