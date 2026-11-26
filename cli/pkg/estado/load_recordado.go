package estado

import (
	"bytes"
	"maps"
	"slices"
	"sync"
	"sync/atomic"
)

// Rule: Prohibido usar 'else'

// recordado es el último estado.yaml parseado por este proceso (FND-190): el panel lo carga unas cinco veces en cada
// relectura, varias veces por segundo, y parsear 115 KB de YAML cuesta ~6 ms cada vez.
var recordado struct {
	sync.Mutex
	ruta  string
	datos []byte
	e     Estado
}

// parseos cuenta los parseos de verdad, para la prueba.
var parseos atomic.Int64

// parseRecordado parsea solo si el contenido cambió (compara los bytes: la hora de modificación puede no cambiar entre dos
// escrituras seguidas) y siempre entrega una copia propia, porque quien carga puede modificar lo que recibe.
func parseRecordado(ruta string, data []byte) (Estado, error) {
	recordado.Lock()
	defer recordado.Unlock()
	if recordado.ruta == ruta && bytes.Equal(recordado.datos, data) {
		return recordado.e.clonar(), nil
	}
	parseos.Add(1)
	e, err := Parse(data)
	if err != nil {
		return Estado{}, err
	}
	recordado.ruta, recordado.datos, recordado.e = ruta, data, e
	return e.clonar(), nil
}

// clonar copia todo lo que tiene un slice o un mapa; TestClonar_NoComparteNingunSliceNiMapa avisa si un campo nuevo falta.
func (e Estado) clonar() Estado {
	c := e
	c.Etapas = slices.Clone(e.Etapas)
	c.Objetivos = slices.Clone(e.Objetivos)
	c.Ciclos = clonarCiclos(e.Ciclos)
	c.Frentes = clonarFrentes(e.Frentes)
	return c
}

func clonarCiclos(ciclos map[string][]string) map[string][]string {
	if ciclos == nil {
		return nil
	}
	c := make(map[string][]string, len(ciclos))
	for nombre, etapas := range ciclos {
		c[nombre] = slices.Clone(etapas)
	}
	return c
}

func clonarFrentes(frentes []Frente) []Frente {
	if frentes == nil {
		return nil
	}
	c := make([]Frente, len(frentes))
	for i, f := range frentes {
		f.DependeDe = slices.Clone(f.DependeDe)
		f.Celdas = maps.Clone(f.Celdas)
		c[i] = f
	}
	return c
}
