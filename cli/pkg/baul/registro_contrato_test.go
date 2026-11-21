package baul

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// El registro solo agrega: cada línea nueva deja intacto lo anterior y lleva la huella de la anterior, así que cambiar
// una línea (no solo borrarla) se nota en esa línea.
// contrato: baul#0bf182a7
func TestRegistro_SoloAgregaYCadaLineaEncadenaLaAnterior(t *testing.T) {
	entorno(t)
	ruta, _ := rutaEn("registro.jsonl")
	var antes []byte
	for _, fin := range []string{"uno", "dos", "tres"} {
		if err := Anotar(Linea{Accion: "usar", Token: "cf", Fin: fin}, ahora); err != nil {
			t.Fatal(err)
		}
		ahoraB, _ := os.ReadFile(ruta)
		if !bytes.HasPrefix(ahoraB, antes) || len(ahoraB) <= len(antes) {
			t.Fatalf("anotar «%s» reescribió lo anterior", fin)
		}
		antes = ahoraB
	}
	lineas, _ := Registro()
	for i, l := range lineas {
		previa := ""
		if i > 0 {
			previa = lineas[i-1].Huella
		}
		if l.Previa != previa || l.Huella != huellaDe(l) {
			t.Fatalf("la línea %d no encadena la anterior: %+v", i+1, l)
		}
	}
	cambiado := strings.Replace(string(antes), `"fin":"dos"`, `"fin":"otro"`, 1)
	if err := os.WriteFile(ruta, []byte(cambiado), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerificarRegistro(); err == nil || !strings.Contains(err.Error(), "línea 2") {
		t.Fatalf("una línea cambiada se nota donde está: %v", err)
	}
}
