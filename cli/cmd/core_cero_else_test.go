package cmd

import (
	"strings"
	"testing"
)

// El núcleo que QDD instala en cada proyecto (principios y certificaciones) dice la razón de fondo del cero 'else'
// donde la IA lee la regla, en cualquier lenguaje (pedido del usuario, 2026-10-31).
func TestCore_CeroElseConSuRazon(t *testing.T) {
	for _, ruta := range []string{
		"core_assets/core/wisdom/principles.md",
		"core_assets/core/certification/CERT-040-EARLY-RETURN.yaml",
		"core_assets/core/certification/CERT-005-CLEAN-CODE.yaml",
	} {
		b, err := coreAssets.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		texto := string(b)
		for _, debe := range []string{"línea a línea", "de un vistazo"} {
			if !strings.Contains(texto, debe) {
				t.Errorf("%s debe decir la razón del cero 'else' (%q)", ruta, debe)
			}
		}
	}
}
