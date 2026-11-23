package baul

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-070 (qdd-movile, 2026-11-10): lo copiado se rechazó («no parece un token») y el reintento encontró el
// portapapeles vacío, sin saber por qué. Se vacía igual por seguridad (baul#6058d327); el rechazo lo dice, para que la
// persona copie de nuevo solo el token.

func TestPortapapeles_ElRechazoDiceQueLoVacioYQueHayQueCopiarDeNuevo_MEJ070(t *testing.T) {
	entorno(t)
	for _, copiado := range []string{"una frase con espacios", "dos\nlíneas", strings.Repeat("x", 5000)} {
		_, err := GuardarDelPortapapeles(otro("mio"), &portapapelesFalso{valor: copiado}, ahora)
		if err == nil || !strings.Contains(err.Error(), "lo vacié") || !strings.Contains(err.Error(), "cópialo de nuevo") {
			t.Fatalf("el rechazo dice que se vació y qué hacer: %v", err)
		}
	}
}
