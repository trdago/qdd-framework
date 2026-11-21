package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Contrato de las fichas: un invariante de una ficha aprobada que ninguna prueba cita se recomienda probar, sin
// bloquear el gate y una sola vez por proyecto (resta 2 al puntaje, no 2 por ficha); citado, deja de aparecer.
func TestHeart_InvarianteSinPruebaSeRecomiendaYCitadoDesaparece(t *testing.T) {
	root := t.TempDir()
	ficha := `id: pagos
nombre: Pagos
estado: aprobado
rutas: [pkg/pagos]
objetivo_operativo: {problema: p, criterio_de_exito: c}
objetivo_tecnologico:
    arquitectura: a
    invariantes: ["Cobra una sola vez."]
pruebas: {comandos: [go test ./pkg/pagos]}
`
	os.MkdirAll(filepath.Join(root, modules.Dir), 0o755)
	os.WriteFile(filepath.Join(root, modules.Dir, "pagos.yaml"), []byte(ficha), 0o644)
	os.MkdirAll(filepath.Join(root, "pkg", "pagos"), 0o755)
	v := RunModuleHeartCheck(root, false)
	if len(v) != 1 || v[0].RuleID != "HEART-04-INVARIANTE-SIN-PRUEBA" || !v[0].Recomendacion {
		t.Fatalf("una recomendación que no bloquea: %+v", v)
	}
	id := modules.IDInvariante("Cobra una sola vez.")
	if !strings.Contains(v[0].Description, "1 de 1 invariantes") || !strings.Contains(v[0].Description, "pagos 1/1") {
		t.Fatalf("dice cuántos faltan y en qué ficha: %s", v[0].Description)
	}
	if _, ok := PrincipioDeRegla(v[0].RuleID); !ok {
		t.Fatal("la regla pertenece a un principio")
	}
	os.WriteFile(filepath.Join(root, "pkg", "pagos", "cobro_test.go"), []byte("package pagos\n// "+modules.CitaInvariante("pagos", id)+"\n"), 0o644)
	if v := RunModuleHeartCheck(root, false); len(v) != 0 {
		t.Fatalf("citado, ya no aparece: %+v", v)
	}
}
