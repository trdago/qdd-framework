package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-08): «dentro de los nodos tiene que existir un título visible para guiarme, porque si
// no veo una lista de cosas y no sé qué significa […] debería estar solo el título primero y, si quiero ver el detalle,
// verlo». Cada caja muestra su título; la descripción va con --detalle, separada del título por una línea.

func conDescripciones() Flujo {
	a := paso("a", "Barra inferior", "b")
	a.Descripcion = "Siempre a mano. La Bandeja muestra cuántas cosas te esperan."
	b := paso("b", "Proyecto en vivo", "c")
	b.Descripcion = "Entrar a un proyecto es mirarlo trabajar."
	c := resultado("c", "Resuelto")
	c.Descripcion = "Baja a «Ya resuelto»."
	return deNodos(a, b, c)
}

func TestTitulos_PorDefectoCadaNodoMuestraSoloSuTitulo(t *testing.T) {
	lineas, _ := DibujarConOpciones(conDescripciones(), 80, sinEstado, Opciones{})
	dibujo := strings.Join(lineas, "\n")
	for _, titulo := range []string{"Barra inferior", "Proyecto en vivo", "Resuelto"} {
		if !strings.Contains(dibujo, titulo) {
			t.Fatalf("falta el título %q:\n%s", titulo, dibujo)
		}
	}
	for _, detalle := range []string{"Siempre a mano", "mirarlo trabajar", "Ya resuelto"} {
		if strings.Contains(dibujo, detalle) {
			t.Fatalf("sin --detalle no va la descripción %q:\n%s", detalle, dibujo)
		}
	}
	if len(lineas) > 9 {
		t.Fatalf("tres títulos caben en pocas líneas: %d\n%s", len(lineas), dibujo)
	}
}

func TestTitulos_ConDetalleElTituloQuedaSeparadoDeLaDescripcion(t *testing.T) {
	lineas, segmentos := DibujarConOpciones(conDescripciones(), 80, sinEstado, Opciones{Detalle: true})
	dibujo := strings.Join(lineas, "\n")
	if !strings.Contains(dibujo, "Siempre a mano") || !strings.Contains(dibujo, "mirarlo trabajar") {
		t.Fatalf("con --detalle va la descripción:\n%s", dibujo)
	}
	titulo := indiceCon(lineas, "Barra inferior")
	if titulo < 0 || !strings.Contains(lineas[titulo+1], "├") || !strings.Contains(lineas[titulo+1], "┤") {
		t.Fatalf("una línea separa el título de su descripción:\n%s", dibujo)
	}
	if !strings.Contains(lineas[titulo+2], "Siempre a mano") {
		t.Fatalf("la descripción va debajo de la línea:\n%s", dibujo)
	}
	if len(segmentos) != len(lineas) {
		t.Fatalf("un renglón de segmentos por línea: %d %d", len(segmentos), len(lineas))
	}
	sinDescripcion, _ := DibujarConOpciones(deNodos(cadena()...), 80, sinEstado, Opciones{Detalle: true})
	if strings.Contains(strings.Join(sinDescripcion, "\n"), "┤") {
		t.Fatalf("un nodo sin descripción no lleva línea:\n%s", strings.Join(sinDescripcion, "\n"))
	}
}

func indiceCon(lineas []string, texto string) int {
	for i, l := range lineas {
		if strings.Contains(l, texto) {
			return i
		}
	}
	return -1
}
