package flujo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/materializar"
)

// Rule: Prohibido usar 'else'
// Flujos grandes legibles, parte A (decisión de la persona, 2026-11-07; #186): cada desvío (un rechazo, una vuelta
// atrás, un atajo) partía el ancho, y el camino principal nunca lo recuperaba: tras seis filtros, sus cajas quedaban de
// 9 columnas y 27 líneas. Un desvío va como etiqueta al costado y el camino principal conserva todo el ancho.

// filtros es una cadena de n filtros; cada uno rechaza al costado y sigue al próximo. Al final, un paso de nombre largo.
func filtros(n int) Flujo {
	var nodos []Nodo
	for i := 0; i < n; i++ {
		nodos = append(nodos, paso(fmt.Sprintf("f%d", i), fmt.Sprintf("Filtro %d", i), fmt.Sprintf("r%d", i), fmt.Sprintf("f%d", i+1)))
		nodos = append(nodos, resultado(fmt.Sprintf("r%d", i), fmt.Sprintf("Rechazo %d", i)))
	}
	nodos = append(nodos, paso(fmt.Sprintf("f%d", n), "Genera el informe final con todo lo que pasó los filtros"))
	return deNodos(nodos...)
}

func lineaCon(lineas []string, parte string) string {
	for _, l := range lineas {
		if strings.Contains(l, parte) {
			return l
		}
	}
	return ""
}

func TestDibujar_UnRechazoVaAlCostadoYElCaminoConservaElAncho(t *testing.T) {
	lineas := Dibujar(filtros(6), 60, sinEstado)
	s := strings.Join(lineas, "\n")
	for i := 0; i < 6; i++ {
		l := lineaCon(lineas, fmt.Sprintf("( Rechazo %d )", i))
		if !strings.Contains(l, "├─") || strings.Contains(l, "Filtro") {
			t.Fatalf("el rechazo %d cuelga al costado, en su propia línea:\n%s", i, s)
		}
	}
	final := lineaCon(lineas, "Genera el informe")
	if !strings.Contains(final, "Genera el informe final con todo lo que pasó") {
		t.Fatalf("tras seis filtros el último paso sigue a todo el ancho, sin partirse:\n%s", s)
	}
	if len(lineas) > 40 {
		t.Fatalf("seis filtros y sus rechazos caben en 40 líneas, no en %d:\n%s", len(lineas), s)
	}
	for _, l := range lineas {
		if materializar.Ancho(l) > 60 {
			t.Fatalf("se pasa del ancho:\n%s", s)
		}
	}
}

func TestDibujar_UnaVueltaAtrasCortaEsUnaEtiqueta(t *testing.T) {
	f := deNodos(paso("cola", "Cola", "gate"), paso("gate", "Doc-Gate", "bloq", "ctx"), paso("bloq", "Bloqueada esperando", "cola"), paso("ctx", "Contexto", "fin"), resultado("fin", "Listo"))
	lineas := Dibujar(f, 60, sinEstado)
	l := lineaCon(lineas, "Bloqueada esperando")
	if !strings.Contains(l, "├─") || !strings.Contains(strings.Join(lineas, "\n"), "↺ vuelve a Cola") {
		t.Fatalf("la vuelta atrás va al costado y dice adónde vuelve:\n%s", strings.Join(lineas, "\n"))
	}
	if strings.Contains(lineaCon(lineas, "Contexto"), "Bloqueada") {
		t.Fatalf("no van lado a lado:\n%s", strings.Join(lineas, "\n"))
	}
}

func TestDibujar_UnAtajoAlEncuentroEsUnaEtiqueta(t *testing.T) {
	f := deNodos(paso("u", "UNSPSC", "rech", "reg", "cola"), resultado("rech", "Rechazado"), paso("reg", "Región", "cola"), paso("cola", "Cola profunda", "fin"), resultado("fin", "Listo"))
	lineas := Dibujar(f, 60, sinEstado)
	s := strings.Join(lineas, "\n")
	if !strings.Contains(lineaCon(lineas, "salta a Cola profunda"), "├─") {
		t.Fatalf("el atajo se dice al costado:\n%s", s)
	}
	if strings.Count(s, "│ Cola profunda") != 1 || strings.Index(s, "Región") > strings.Index(s, "│ Cola profunda") {
		t.Fatalf("la cola se dibuja una vez, después de la región:\n%s", s)
	}
}

func TestDibujar_UnDesvioElegidoSeDibujaComoCaja(t *testing.T) {
	f := deNodos(paso("a", "Filtra", "r", "b"), resultado("r", "Rechazado"), resultado("b", "Sigue"))
	f.Nodos[2] = paso("b", "Sigue", "c")
	f.Nodos = append(f.Nodos, resultado("c", "Fin"))
	lineas := DibujarElegido(f, 60, sinEstado, "r")
	if !strings.Contains(lineaCon(lineas, "Rechazado"), "▶") {
		t.Fatalf("el nodo elegido se resalta, aunque sea un desvío:\n%s", strings.Join(lineas, "\n"))
	}
}

// Dos ramas que se alcanzan entre sí no pueden ser atajo la una de la otra: ninguna se dibujaba (azar, ronda 165).
func TestDibujar_DosRamasQueSeAlcanzanEntreSiSeDibujan(t *testing.T) {
	f := deNodos(paso("n0", "N00", "n1", "n2", "n5", "n3"), paso("n1", "N01", "n3"), paso("n2", "N02", "n4", "n5"), paso("n3", "N03"), paso("n4", "N04", "n4"), paso("n5", "N05", "n2"))
	s := texto(f, 73)
	for _, n := range f.Nodos {
		if !strings.Contains(s, n.Nombre) {
			t.Fatalf("falta %s:\n%s", n.Nombre, s)
		}
	}
}

// FND-180: el dibujo anterior dejaba fuera un nodo cuando una rama llegaba al encuentro por otra (azar, semilla 9).
func TestDibujar_CadaNodoApareceAunqueLasRamasSeCrucen_FND180(t *testing.T) {
	f := deNodos(paso("n0", "N00", "n1", "n3"), paso("n1", "N01", "n2", "n3", "n4"), paso("n2", "N02", "n4"), paso("n3", "N03", "n2"), paso("n4", "N04"))
	s := texto(f, 70)
	for _, n := range f.Nodos {
		if !strings.Contains(s, n.Nombre) {
			t.Fatalf("falta %s:\n%s", n.Nombre, s)
		}
	}
}
