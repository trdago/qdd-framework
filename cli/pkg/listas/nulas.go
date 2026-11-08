package listas

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

// Rule: Prohibido usar 'else'

// Nulas lee un JSON junto con el tipo de Go que lo produjo y dice dónde una lista o un mapa vino null («.filas[0].celdas»).
// Es el contrato que revisan las pruebas de lo que qdd le entrega al panel. Un elemento null de una lista no cuenta: dice
// «esta fila no tiene» (ver Marshal).
func Nulas(datos []byte, tipo reflect.Type) []string {
	var v any
	if json.Unmarshal(datos, &v) != nil {
		return []string{"(no es JSON)"}
	}
	var nulas []string
	revisar(v, tipo, "", false, &nulas, 0)
	slices.Sort(nulas)
	return nulas
}

func revisar(v any, t reflect.Type, camino string, elemento bool, nulas *[]string, prof int) {
	if v == nil && t.Kind() == reflect.Pointer {
		return
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if prof > maxProfundidad || t == tipoCrudo {
		return
	}
	esLista := t.Kind() == reflect.Map || (t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8)
	if v == nil && esLista && !elemento {
		*nulas = append(*nulas, camino)
		return
	}
	switch t.Kind() {
	case reflect.Struct:
		revisarStruct(v, t, camino, nulas, prof)
	case reflect.Slice, reflect.Array:
		lista, _ := v.([]any)
		for i, e := range lista {
			revisar(e, t.Elem(), camino+"["+itoa(i)+"]", true, nulas, prof+1)
		}
	case reflect.Map:
		mapa, _ := v.(map[string]any)
		claves := make([]string, 0, len(mapa))
		for k := range mapa {
			claves = append(claves, k)
		}
		slices.Sort(claves)
		for _, k := range claves {
			revisar(mapa[k], t.Elem(), camino+"."+k, false, nulas, prof+1)
		}
	}
}

func revisarStruct(v any, t reflect.Type, camino string, nulas *[]string, prof int) {
	objeto, ok := v.(map[string]any)
	if !ok {
		return
	}
	for i := range t.NumField() {
		c := t.Field(i)
		nombre, _, _ := strings.Cut(c.Tag.Get("json"), ",")
		if nombre == "-" {
			continue
		}
		if c.Anonymous && nombre == "" && c.Type.Kind() == reflect.Struct {
			revisarStruct(v, c.Type, camino, nulas, prof+1)
			continue
		}
		if !c.IsExported() {
			continue
		}
		if nombre == "" {
			nombre = c.Name
		}
		valor, esta := objeto[nombre]
		if !esta {
			continue
		}
		revisar(valor, c.Type, camino+"."+nombre, false, nulas, prof+1)
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
