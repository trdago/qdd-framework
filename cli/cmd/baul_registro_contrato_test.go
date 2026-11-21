package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
)

// Rule: Prohibido usar 'else'

// doctor --fix aparta el registro roto tal cual (la evidencia no se toca) y empieza otro que lo anota.
// contrato: baul#0bf182a7
func TestDoctorBaul_FixApartaElRegistroRotoSinReescribirlo(t *testing.T) {
	baulDePrueba(t)
	root := t.TempDir()
	ahora := time.Now()
	if err := baul.Guardar(baul.Entrada{Nombre: "cf", Tipo: "cloudflare", Donde: baul.EnLlavero}, "valor-de-prueba-123", ahora); err != nil {
		t.Fatal(err)
	}
	for _, fin := range []string{"uno", "dos", "tres"} {
		if err := baul.Anotar(baul.Linea{Accion: "usar", Token: "cf", Fin: fin}, ahora); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(os.Getenv("QDD_HOME"), "baul")
	ruta := filepath.Join(dir, "registro.jsonl")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("el registro vive en %s: %v", ruta, err)
	}
	lineas := strings.SplitAfter(string(b), "\n")
	roto := lineas[0] + strings.Join(lineas[2:], "")
	os.WriteFile(ruta, []byte(roto), 0o600)

	g := buildBaulChecks(root)
	if item := itemMovil(t, g, baul.ItemRegistro); item.Success {
		t.Fatalf("doctor ve el registro roto: %+v", item)
	}
	repairBaul(g)
	apartados, _ := filepath.Glob(filepath.Join(dir, "registro-roto-*.jsonl"))
	if len(apartados) != 1 {
		t.Fatalf("queda un apartado: %v", apartados)
	}
	if got, _ := os.ReadFile(apartados[0]); string(got) != roto {
		t.Fatalf("el apartado es el roto, byte a byte:\n%s", got)
	}
	nuevas, _ := baul.Registro()
	if len(nuevas) != 1 || nuevas[0].Accion != "apartar" || nuevas[0].Previa != "" || baul.VerificarRegistro() != nil {
		t.Fatalf("empieza otra cadena que anota el apartado: %+v", nuevas)
	}
	if item := itemMovil(t, buildBaulChecks(root), baul.ItemRegistro); !item.Success {
		t.Fatalf("después de --fix, intacto: %+v", item)
	}
}
