package estado

// Rule: Prohibido usar 'else'
// MEJ-010 (sfrc, 2026-11-04): las tareas cambian con el tiempo y el tablero queda atrás sin que nada avise. qdd estado check
// pasaba con 15 tareas terminadas y abiertas, y con 6 cerradas que conservaban un ⏳. Esto solo mira la tabla: no adivina
// si un nombre o un criterio siguen siendo ciertos, y no bloquea, porque terminar una tarea lo decide quien trabaja en ella.

// TareasDesfasadas son los ids de las tareas cuya tabla ya no dice lo que pasa.
type TareasDesfasadas struct {
	Listas            []string `json:"listas"`
	CerradasConEspera []string `json:"cerradas_con_espera"`
}

// Desfasadas revisa cada tarea: abierta con todo listo, o cerrada con un ⏳ que ya no espera a nadie.
func Desfasadas(e Estado) TareasDesfasadas {
	d := TareasDesfasadas{Listas: []string{}, CerradasConEspera: []string{}}
	for _, f := range e.Frentes {
		etapas := e.EtapasDe(f)
		if f.Estado == EstadoAbierto && f.Terminado(etapas) {
			d.Listas = append(d.Listas, f.ID)
		}
		if f.Estado == EstadoCerrado && conservaEspera(f, etapas) {
			d.CerradasConEspera = append(d.CerradasConEspera, f.ID)
		}
	}
	return d
}

func conservaEspera(f Frente, etapas []string) bool {
	for _, etapa := range etapas {
		if NormalizarEstado(f.Celdas[etapa].Estado) == Espera {
			return true
		}
	}
	return false
}
