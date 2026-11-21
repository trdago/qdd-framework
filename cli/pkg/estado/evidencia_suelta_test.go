package estado

import "testing"

// Un ✅ dice cómo se sabe que está hecho: un commit, una versión, n/n pruebas, una corrida, o lo que confirmó la persona
// y dónde. Vacía o una palabra suelta («listo», «ok») no dice nada y no pasa (EPF-C02). Medido 2026-11-21: ninguna de
// las 1045 celdas ✅ de los proyectos de este PC era una palabra suelta.
// contrato: estado-por-tareas#4a92ed45
func TestCheck_EPFC02_UnOkConUnaPalabraSueltaNoPasa(t *testing.T) {
	for _, ev := range []string{"", "  ", "listo", "OK", "Hecho.", "done", "✅"} {
		if vs := checkCelda("t", "Código", Celda{Estado: Hecho, Evidencia: ev}); len(vs) == 0 || vs[0].ID != "EPF-C02" {
			t.Fatalf("%q no dice cómo se sabe: %v", ev, vs)
		}
	}
	for _, ev := range []string{"a1b2c3d", "v2.10.0-rc.108", "12/12 pruebas", "corrida 4421", "La persona lo confirmó desde el panel (2026-11-21)"} {
		if vs := checkCelda("t", "Código", Celda{Estado: Hecho, Evidencia: ev}); len(vs) != 0 {
			t.Fatalf("%q dice cómo se sabe: %v", ev, vs)
		}
	}
}
