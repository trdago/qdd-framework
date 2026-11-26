// Package listas entrega JSON sin listas null (FND-189). Un slice o un mapa sin valor de Go sale como null, y el panel
// que lo recorre se cae. Marshal los entrega como [] y {} a cualquier profundidad; un puntero, un json.RawMessage o
// unos []byte sin valor siguen saliendo null, porque ahí null significa algo. Tampoco cambia un elemento de una lista:
// una fila null dentro de una lista dice «esta no tiene» (los segmentos de una línea del dibujo), y [] diría «vacía».
package listas

import (
	"encoding/json"
	"reflect"
)

// Rule: Prohibido usar 'else'

// maxProfundidad corta un ciclo de punteros: json.Marshal lo rechaza igual, pero no hay que colgarse antes.
const maxProfundidad = 64

var tipoCrudo = reflect.TypeFor[json.RawMessage]()

// Marshal es json.Marshal con los slices y mapas sin valor como [] y {}. No cambia el valor de quien llama.
func Marshal(v any) ([]byte, error) {
	if v == nil {
		return json.Marshal(v)
	}
	copia := copiar(reflect.ValueOf(v), 0)
	return json.Marshal(copia.Interface())
}

// copiar devuelve una copia con las listas llenas; comparte lo que no cambia.
func copiar(v reflect.Value, prof int) reflect.Value {
	if prof > maxProfundidad {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		return copiarPuntero(v, prof)
	case reflect.Interface:
		return copiarInterfaz(v, prof)
	case reflect.Struct:
		return copiarStruct(v, prof)
	case reflect.Slice:
		return copiarSlice(v, prof)
	case reflect.Map:
		return copiarMapa(v, prof)
	case reflect.Array:
		return copiarArreglo(v, prof)
	}
	return v
}

func copiarPuntero(v reflect.Value, prof int) reflect.Value {
	if v.IsNil() {
		return v
	}
	nuevo := reflect.New(v.Type().Elem())
	nuevo.Elem().Set(copiar(v.Elem(), prof+1))
	return nuevo
}

func copiarInterfaz(v reflect.Value, prof int) reflect.Value {
	if v.IsNil() {
		return v
	}
	nuevo := reflect.New(v.Type()).Elem()
	nuevo.Set(copiar(v.Elem(), prof+1))
	return nuevo
}

// copiarStruct copia los campos exportados; los demás no salen en el JSON. Un struct incrustado sin exportar no se puede
// copiar sin unsafe: sus listas quedan como vienen, y la prueba de contrato (Nulas) lo dice. Un tipo con su propio MarshalJSON también
// pasa por aquí: sus campos siguen siendo los que él serializa.
func copiarStruct(v reflect.Value, prof int) reflect.Value {
	nuevo := reflect.New(v.Type()).Elem()
	nuevo.Set(v)
	for i := range v.NumField() {
		if !v.Type().Field(i).IsExported() {
			continue
		}
		nuevo.Field(i).Set(copiar(v.Field(i), prof+1))
	}
	return nuevo
}

func copiarSlice(v reflect.Value, prof int) reflect.Value {
	if v.Type() == tipoCrudo || v.Type().Elem().Kind() == reflect.Uint8 {
		return v
	}
	if v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0)
	}
	nuevo := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	for i := range v.Len() {
		nuevo.Index(i).Set(copiarElemento(v.Index(i), prof+1))
	}
	return nuevo
}

// copiarElemento: un elemento de una lista que es una lista o un mapa sin valor sigue null; lo demás se copia igual.
func copiarElemento(v reflect.Value, prof int) reflect.Value {
	if (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.IsNil() {
		return v
	}
	return copiar(v, prof)
}

func copiarMapa(v reflect.Value, prof int) reflect.Value {
	nuevo := reflect.MakeMapWithSize(v.Type(), v.Len())
	for it := v.MapRange(); it.Next(); {
		nuevo.SetMapIndex(it.Key(), copiar(it.Value(), prof+1))
	}
	return nuevo
}

func copiarArreglo(v reflect.Value, prof int) reflect.Value {
	nuevo := reflect.New(v.Type()).Elem()
	for i := range v.Len() {
		nuevo.Index(i).Set(copiarElemento(v.Index(i), prof+1))
	}
	return nuevo
}
