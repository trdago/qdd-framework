package cmd

import (
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// El núcleo que QDD instala en cada proyecto (principios y certificaciones) dice la razón de fondo del cero 'else'
// donde la IA lee la regla, en cualquier lenguaje (pedido del usuario, 2026-10-31).
// contrato: metodo#1995b1da
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

// La razón se escribe una vez en Go (modules.PorQueCeroElse) y el núcleo la repite palabra por palabra: si una cambia,
// la otra tiene que cambiar con ella.
// contrato: metodo#fdb6cdd1
func TestCore_RepiteLaRazonDeGoConLasMismasPalabras(t *testing.T) {
	for _, ruta := range []string{
		"core_assets/core/wisdom/principles.md",
		"core_assets/core/certification/CERT-040-EARLY-RETURN.yaml",
		"core_assets/core/certification/CERT-005-CLEAN-CODE.yaml",
	} {
		b, err := coreAssets.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.Join(strings.Fields(string(b)), " "), modules.PorQueCeroElse) {
			t.Errorf("%s no repite modules.PorQueCeroElse con las mismas palabras", ruta)
		}
	}
}
