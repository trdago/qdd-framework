package estado

import (
	"encoding/json"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-296 (qdd-movile, 2026-11-22): el tablero muestra las 4 pendientes más nuevas; 4 tareas nuevas de verificación
// tapaban 9 ⏳ de la persona y ella vio «no tengo nada en el tablero». Las ocultas que esperan a la persona salen aparte
// (ocultas_tuyas), para una línea «⏳ N esperan por ti» que las despliega (elegido por el modo goal, 2026-11-22).

func TestDatos_LasOcultasQueEsperanALaPersonaSalenAparte_FND296(t *testing.T) {
	yaml := "formato: 1\netapas: [Código]\nfrentes:\n" +
		"  - {id: vieja, nombre: Vieja, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: publicar, de_quien: la persona}}}\n" +
		"  - {id: ci, nombre: CI, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: CI, de_quien: CI}}}\n" +
		"  - {id: n1, nombre: N1, estado: abierto, celdas: {Código: {estado: ❌, falla: rota}}}\n" +
		"  - {id: n2, nombre: N2, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: decidir, de_quien: persona}}}\n"
	d := DatosPendientes(mustParse(t, yaml), ahora, 2)
	if d.Ocultos != 2 {
		t.Fatalf("ocultos sigue contando todas las que no se ven: %d", d.Ocultos)
	}
	if len(d.OcultasTuyas) != 1 || d.OcultasTuyas[0].ID != "vieja" {
		t.Fatalf("solo la oculta que espera a la persona, con su fila: %+v", d.OcultasTuyas)
	}
	for _, f := range d.Frentes {
		if f.ID == "vieja" {
			t.Fatal("no se repite en las visibles")
		}
	}
}

func TestDatos_SinOcultasTuyasLaListaVaVacia_FND296(t *testing.T) {
	yaml := "formato: 1\netapas: [Código]\nfrentes:\n" +
		"  - {id: a, nombre: A, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: decidir, de_quien: persona}}}\n"
	d := DatosPendientes(mustParse(t, yaml), ahora, 4)
	if d.OcultasTuyas == nil || len(d.OcultasTuyas) != 0 {
		t.Fatalf("una lista vacía, no null: %+v", d.OcultasTuyas)
	}
}

// qdd estado ver --json (Datos, sin tope) entregaba ocultas_tuyas null: el contrato es una lista, vacía si no hay.
func TestDatos_SinTopeOcultasTuyasEsUnaListaVacia_FND296(t *testing.T) {
	yaml := "formato: 1\netapas: [Código]\nfrentes:\n" +
		"  - {id: a, nombre: A, estado: abierto, celdas: {Código: {estado: ⏳, bloqueo: decidir, de_quien: persona}}}\n"
	b, err := json.Marshal(Datos(mustParse(t, yaml), ahora))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"ocultas_tuyas":[]`) {
		t.Fatalf("una lista vacía, no null: %s", b)
	}
}
