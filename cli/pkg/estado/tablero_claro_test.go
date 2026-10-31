package estado

import (
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Pedido del usuario (2026-10-31, «se ve raro el tablero»): el panel mostraba «Pr» y «PR» como dos columnas, y una
// tabla sin tocar desde ayer se veía igual que una al día, con la hora sola («14:59») y sin decir de qué día.

// FND-087: Pruebas y PR quedaban «Pr» y «PR»: distintas para el código, iguales para quien lee.
func TestAbreviaturas_NoChocanAunqueSoloCambieLaMayuscula_FND087(t *testing.T) {
	got := abreviaturas([]string{"Código", "Pruebas", "PR", "Publicación"})
	vistas := map[string]bool{}
	for _, a := range got {
		if vistas[strings.ToLower(a)] {
			t.Fatalf("dos columnas se leen igual: %v", got)
		}
		vistas[strings.ToLower(a)] = true
		if AnchoVisible(a) != 2 {
			t.Fatalf("abreviatura de 2 columnas: %q en %v", a, got)
		}
	}
	if got[0] != "Có" || got[2] != "PR" || got[3] != "Pu" {
		t.Fatalf("lo que no choca no cambia: %v", got)
	}
}

// FND-088: la cabecera decía «Tareas · 14:59» al día siguiente; nada avisaba que el tablero estaba sin actualizar.
func TestDatos_DicenHaceCuantoSeActualizoYSiQuedoViejo_FND088(t *testing.T) {
	e := mustParse(t, valido)
	e.Actualizado = ahora.Add(-9 * time.Hour).Format(time.RFC3339)
	d := DatosPendientes(e, ahora, 4)
	if d.Hace != "hace 9 h" || !d.Viejo {
		t.Fatalf("9 h sin tocar con tareas pendientes es un tablero viejo: hace=%q viejo=%v", d.Hace, d.Viejo)
	}
	e.Actualizado = ahora.Add(-20 * time.Minute).Format(time.RFC3339)
	d = DatosPendientes(e, ahora, 4)
	if d.Hace != "hace 20 min" || d.Viejo {
		t.Fatalf("recién actualizado: hace=%q viejo=%v", d.Hace, d.Viejo)
	}
	e.Actualizado = ahora.Add(-50 * time.Hour).Format(time.RFC3339)
	if d = DatosPendientes(e, ahora, 4); d.Hace != "hace 2 d" {
		t.Fatalf("en días: %q", d.Hace)
	}
	e.Actualizado = "no es una fecha"
	if d = DatosPendientes(e, ahora, 4); d.Hace != "" || d.Viejo {
		t.Fatalf("sin fecha legible no se inventa: hace=%q viejo=%v", d.Hace, d.Viejo)
	}
}

func TestDatos_SinPendientesNuncaEsViejo(t *testing.T) {
	e := mustParse(t, valido)
	for i := range e.Frentes {
		e.Frentes[i].Estado = EstadoCerrado
	}
	e.Actualizado = ahora.Add(-72 * time.Hour).Format(time.RFC3339)
	if d := DatosPendientes(e, ahora, 4); d.Viejo {
		t.Fatal("un tablero sin nada pendiente no tiene nada que actualizar")
	}
}
