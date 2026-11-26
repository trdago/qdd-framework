package flujo

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/materializar"
)

// Rule: Prohibido usar 'else'
// FND-181: una rama lado a lado con ramas con sangría anidadas tres o cuatro niveles pasaba del ancho de su rama: en
// 24 000 dibujos al azar (60 semillas × 400 rondas) se pasaba en 2. Caso: semilla 23, ronda 220, ancho 50 (51
// columnas). Ahora una bifurcación cuya rama no cabe en su columna va entera con sangría. El barrido de las 60
// semillas pasa (15 s; se corrió al arreglarlo); aquí queda el caso, en todos los anchos de 40 a 80.

func casoFND181() Flujo {
	azar := rand.New(rand.NewSource(23))
	for range 220 {
		flujoAlAzar(azar)
		azar.Intn(41)
	}
	return flujoAlAzar(azar)
}

func TestDibujar_RamasConSangriaAnidadasCabenEnSuRama_FND181(t *testing.T) {
	f := casoFND181()
	for ancho := 40; ancho <= 80; ancho++ {
		lineas := Dibujar(f, ancho, sinEstado)
		dibujo := strings.Join(lineas, "\n")
		for _, l := range lineas {
			if materializar.Ancho(l) > ancho {
				t.Fatalf("ancho %d: %d columnas:\n%s", ancho, materializar.Ancho(l), dibujo)
			}
		}
		for _, n := range f.Nodos {
			if !strings.Contains(dibujo, n.Nombre) {
				t.Fatalf("ancho %d: falta %s:\n%s", ancho, n.Nombre, dibujo)
			}
		}
	}
}
