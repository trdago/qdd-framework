package cmd

import "testing"

// Rule: Prohibido usar 'else'
// Solo un binario instalado deja su ruta en settings.json: uno compilado a mano o de una prueba no es la barra de nadie.
func TestBinarioDeLaBarra_SoloElInstaladoConNPM(t *testing.T) {
	if got := binarioDeLaBarra("/opt/lib/node_modules/qdd-framework/bin/qdd"); got != "/opt/lib/node_modules/qdd-framework/bin/qdd" {
		t.Fatalf("el de npm va directo: %q", got)
	}
	for _, bin := range []string{"/home/ana/.cache/qdd-297", "/tmp/go-build1/qdd.test", ""} {
		if got := binarioDeLaBarra(bin); got != "" {
			t.Fatalf("%q no se escribe en la barra: %q", bin, got)
		}
	}
}
