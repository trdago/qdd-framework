package flujo

import (
	"sort"
	"strings"

	"github.com/qdd-framework/qdd/pkg/materializar"
)

// Rule: Prohibido usar 'else'
// Acciones sobre el nodo en el dibujo (pedido de la persona, 2026-11-08: «no me gusta tener que bajar hasta abajo para
// una acción al nodo … que fuera directa sobre el elemento para no perder el hilo»). Cada bloque del dibujo sabe qué
// columnas de qué línea son de qué nodo, y las mueve consigo al apilarse, ponerse lado a lado o colgar con sangría. Al
// final, cada línea se parte en segmentos: el panel dibuja pulsable el de un nodo, ahí donde se ve.

// tramo es un trozo de una línea del bloque que es de un nodo, en columnas de pantalla: [desde, hasta).
type tramo struct {
	fila, desde, hasta int
	nodo               string
}

// Segmento es un trozo de una línea del dibujo; Nodo dice de qué nodo es, o nada si es solo trazo.
type Segmento struct {
	Texto string `json:"texto"`
	Nodo  string `json:"nodo,omitempty"`
}

func moverTramos(tramos []tramo, filas, columnas int) []tramo {
	movidos := make([]tramo, len(tramos))
	for i, t := range tramos {
		movidos[i] = tramo{fila: t.fila + filas, desde: t.desde + columnas, hasta: t.hasta + columnas, nodo: t.nodo}
	}
	return movidos
}

// tramosDeLineas: cada línea entera es del nodo (su caja, su círculo o la referencia a él).
func tramosDeLineas(lineas []string, nodo string) []tramo {
	tramos := make([]tramo, len(lineas))
	for i, l := range lineas {
		tramos[i] = tramo{fila: i, desde: 0, hasta: materializar.Ancho(l), nodo: nodo}
	}
	return tramos
}

// tramosDeEtiqueta reparte una línea de la etiqueta de un desvío entre los nodos que nombra, en orden. Si un nombre
// quedó partido entre dos líneas, la línea entera es del primer nodo que falta por nombrar.
func (d *dibujante) tramosDeEtiqueta(parte string, fila, desde int, candidatos []string) []tramo {
	if len(candidatos) == 0 {
		return nil
	}
	var tramos []tramo
	cursor := 0
	for _, c := range candidatos {
		nombre := d.porID[c].Nombre
		i := strings.Index(parte[cursor:], nombre)
		if nombre == "" || i < 0 {
			continue
		}
		inicio := cursor + i
		cursor = inicio + len(nombre)
		tramos = append(tramos, tramo{fila: fila, desde: desde + materializar.Ancho(parte[:inicio]), hasta: desde + materializar.Ancho(parte[:cursor]), nodo: c})
	}
	if len(tramos) > 0 {
		return tramos
	}
	return []tramo{{fila: fila, desde: desde, hasta: desde + materializar.Ancho(parte), nodo: candidatos[0]}}
}

// segmentar parte cada línea con nodos en sus segmentos; una línea sin nodos queda en nil.
func segmentar(lineas []string, tramos []tramo) [][]Segmento {
	porFila := map[int][]tramo{}
	for _, t := range tramos {
		porFila[t.fila] = append(porFila[t.fila], t)
	}
	segmentos := make([][]Segmento, len(lineas))
	for fila, ts := range porFila {
		if fila < 0 || fila >= len(lineas) {
			continue
		}
		segmentos[fila] = partirLinea(lineas[fila], ts)
	}
	return segmentos
}

// partirLinea junta las runas seguidas que son del mismo nodo (o de ninguno). La columna de cada runa es el ancho de lo
// que tiene antes, el mismo cálculo con que se armó el dibujo.
func partirLinea(linea string, tramos []tramo) []Segmento {
	sort.Slice(tramos, func(i, j int) bool { return tramos[i].desde < tramos[j].desde })
	var segmentos []Segmento
	var actual, antes strings.Builder
	nodoActual := ""
	for _, r := range linea {
		nodo := nodoEn(tramos, materializar.Ancho(antes.String()))
		antes.WriteRune(r)
		if nodo != nodoActual && actual.Len() > 0 {
			segmentos = append(segmentos, Segmento{Texto: actual.String(), Nodo: nodoActual})
			actual.Reset()
		}
		nodoActual = nodo
		actual.WriteRune(r)
	}
	if actual.Len() > 0 {
		segmentos = append(segmentos, Segmento{Texto: actual.String(), Nodo: nodoActual})
	}
	if !conNodo(segmentos) {
		return nil
	}
	return segmentos
}

func nodoEn(tramos []tramo, columna int) string {
	for _, t := range tramos {
		if columna >= t.desde && columna < t.hasta {
			return t.nodo
		}
	}
	return ""
}

func conNodo(segmentos []Segmento) bool {
	for _, s := range segmentos {
		if s.Nodo != "" {
			return true
		}
	}
	return false
}
