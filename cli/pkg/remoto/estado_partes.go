package remoto

import (
	"encoding/json"
	"slices"
)

// Rule: Prohibido usar 'else'
// FND-209 (acordado con qdd-movile, 2026-11-22): Realtime descarta sin aviso un broadcast que pasa su tope (256 KB en
// el plan Free) y el estado completo crece con cada proyecto. A los teléfonos que declaran estado_por_partes, un estado
// completo que pasa TopeParte les llega en partes. Cada parte lleva la cabecera completa (se valida sola: versión y
// máquina), el mismo seq, parte (desde 1) y de, y solo sus proyectos, en orden. La app lo arma con las de partes del
// mismo seq; si falta una, descarta lo parcial y pide sync.

// CapacidadEstadoPorPartes la declara en su presence un teléfono que sabe armar un estado en partes.
const CapacidadEstadoPorPartes = "estado_por_partes"

// TopeParte es cuánto pesa como mucho cada parte: la mitad del tope de Realtime, para que un proyecto que crece entre
// dos mensajes no lo pase.
var TopeParte = TopeBroadcast / 2

// partirEstado reparte los proyectos en partes de hasta tope bytes, en orden. Un estado que cabe va entero, sin parte
// ni de. Un proyecto que solo ya pasa el tope va en su propia parte: no se recorta.
func partirEstado(e Estado, tope int) []Estado {
	if pesoJSON(e) <= tope {
		return []Estado{e}
	}
	cabecera := e
	cabecera.Proyectos = []ProyectoEstado{}
	var partes []Estado
	actual := cabecera
	for _, p := range e.Proyectos {
		con := actual
		con.Proyectos = append(slices.Clone(actual.Proyectos), p)
		if len(actual.Proyectos) > 0 && pesoJSON(con) > tope {
			partes = append(partes, actual)
			con = cabecera
			con.Proyectos = []ProyectoEstado{p}
		}
		actual = con
	}
	partes = append(partes, actual)
	for i := range partes {
		partes[i].Parte, partes[i].De = i+1, len(partes)
	}
	return partes
}

func pesoJSON(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return len(b)
}

// todosArmanPartes: hay teléfonos y todos declaran la capacidad.
func (b *Bucle) todosArmanPartes() bool {
	if len(b.telefonos) == 0 {
		return false
	}
	for _, t := range b.telefonos {
		if !slices.Contains(t.Capacidades, CapacidadEstadoPorPartes) {
			return false
		}
	}
	return true
}
