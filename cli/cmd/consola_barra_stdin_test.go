package cmd

import (
	"io"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

// FND-145: la barra leía stdin hasta su fin. Con un stdin que nadie cierra, el proceso quedaba vivo para siempre.
func TestLeerEntradaBarra_StdinQueNadieCierraNoLaCuelga_FND145(t *testing.T) {
	r, w := io.Pipe()
	defer w.Close()
	inicio := time.Now()
	if datos, ok := leerEntradaBarra(r, 50*time.Millisecond); ok || datos != nil {
		t.Fatalf("sin fin de la entrada no hay barra: %q %v", datos, ok)
	}
	if d := time.Since(inicio); d > 2*time.Second {
		t.Fatalf("tenía que cortar al tope: %v", d)
	}
}

func TestLeerEntradaBarra_EntradaCompleta_FND145(t *testing.T) {
	datos, ok := leerEntradaBarra(strings.NewReader(`{"a":1}`), time.Second)
	if !ok || string(datos) != `{"a":1}` {
		t.Fatalf("entrega lo leído: %q %v", datos, ok)
	}
}

func TestLeerEntradaBarra_NoLeeMasDelLimite_FND145(t *testing.T) {
	datos, ok := leerEntradaBarra(strings.NewReader(strings.Repeat("x", maxEntradaBarra+10)), time.Second)
	if !ok || len(datos) != maxEntradaBarra {
		t.Fatalf("lee hasta el límite: %d %v", len(datos), ok)
	}
}
