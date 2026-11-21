package listas

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-045 y MEJ-050 (FND-189): 'qdd consola blueprints --json' entregó "entradas": null, un slice nil de Go, y el
// panel quedó en negro al hacer inv.entradas.filter. Lo que qdd le entrega al panel nunca lleva una lista null: un
// slice o un mapa sin valor sale como [] o {}, a cualquier profundidad.

type entrada struct{ ID string }

// Base se incrusta con sus campos a la vista; un tipo incrustado sin exportar no se puede copiar sin unsafe.
type Base struct {
	Base []string `json:"base"`
}

type inventario struct {
	Entradas  []entrada         `json:"entradas"`
	Problemas []string          `json:"problemas"`
	Por       map[string]int    `json:"por"`
	Sub       *inventario       `json:"sub"`
	Cuando    *time.Time        `json:"cuando"`
	Crudo     json.RawMessage   `json:"crudo"`
	Bytes     []byte            `json:"bytes"`
	Varios    []inventario      `json:"varios"`
	Algo      any               `json:"algo"`
	Mapa      map[string][]bool `json:"mapa"`
	oculto    []string
}

func TestMarshal_NingunaListaNiMapaSaleNull_FND189(t *testing.T) {
	v := inventario{
		Sub:    &inventario{},
		Varios: []inventario{{}},
		Algo:   inventario{},
		Mapa:   map[string][]bool{"a": nil},
	}
	b, err := Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, debe := range []string{`"entradas":[]`, `"problemas":[]`, `"por":{}`, `"mapa":{"a":[]}`, `"sub":{"entradas":[]`, `"varios":[{"entradas":[]`, `"algo":{"entradas":[]`} {
		if !strings.Contains(s, debe) {
			t.Fatalf("falta %s en %s", debe, s)
		}
	}
	for _, sigue := range []string{`"cuando":null`, `"crudo":null`, `"bytes":null`} {
		if !strings.Contains(s, sigue) {
			t.Fatalf("un puntero, un JSON crudo o unos bytes sin valor siguen null (%s): %s", sigue, s)
		}
	}
	if v.Entradas != nil || v.Algo.(inventario).Entradas != nil {
		t.Fatal("no cambia el valor de quien llama")
	}
}

func TestMarshal_ValoresSueltos(t *testing.T) {
	for _, caso := range []struct {
		v    any
		sale string
	}{
		{nil, "null"},
		{[]string(nil), "[]"},
		{map[string]int(nil), "{}"},
		{(*inventario)(nil), "null"},
		{[]*inventario{nil}, "[null]"},
		{[][]string{nil, {"a"}}, `[null,["a"]]`},
		{[1][]string{}, `[null]`},
		{json.RawMessage(`{"a":null}`), `{"a":null}`},
		{3, "3"},
	} {
		b, err := Marshal(caso.v)
		if err != nil || string(b) != caso.sale {
			t.Fatalf("%#v: %s %v", caso.v, b, err)
		}
	}
}

func TestMarshal_UnCicloNoLoCuelga(t *testing.T) {
	type nodo struct {
		Hijos []string `json:"hijos"`
		Otro  *nodo    `json:"otro,omitempty"`
	}
	n := &nodo{}
	n.Otro = n
	if _, err := Marshal(n); err == nil {
		t.Fatal("un ciclo es un error de json, como siempre, y no se cuelga")
	}
}

func TestNulas_DiceDondeHayUnaListaNull(t *testing.T) {
	type fila struct {
		Celdas []string `json:"celdas"`
	}
	type dato struct {
		Base
		Filas  []fila          `json:"filas"`
		Por    map[string]fila `json:"por_fila"`
		Puntos *[]int          `json:"puntos"`
		Nada   []string        `json:"-"`
		Tal    []string        `json:"tal,omitempty"`
	}
	crudo := []byte(`{"entradas":null,"problemas":[],"base":null,"filas":[{"celdas":null},null],"por_fila":{"x":{"celdas":null}},"puntos":null,"algo":null}`)
	nulas := Nulas(crudo, reflect.TypeFor[dato]())
	quiere := []string{".base", ".filas[0].celdas", ".por_fila.x.celdas"} // en orden
	if strings.Join(nulas, " ") != strings.Join(quiere, " ") {
		t.Fatalf("nulas: %v, quiere %v", nulas, quiere)
	}
	b, _ := Marshal(dato{})
	if n := Nulas(b, reflect.TypeFor[dato]()); len(n) != 0 {
		t.Fatalf("lo que entrega Marshal no tiene ninguna: %v en %s", n, b)
	}
}
