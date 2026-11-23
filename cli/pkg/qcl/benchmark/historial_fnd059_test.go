package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
)

// Rule: Prohibido usar 'else'
// FND-059: cognitive_history.json lo escribían el MCP (eventos de herramientas) y el benchmark (certificados), con
// esquemas distintos. El MCP recortaba el archivo a 100 entradas y borraba certificados; el benchmark leía los eventos
// como certificados en 0 y la tendencia salía falsa. Cada fuente tiene su archivo; uno mezclado se separa sin perder nada.

func escribirJSON(t *testing.T, ruta, texto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

func leerLista(t *testing.T, ruta string) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var l []map[string]any
	if err := json.Unmarshal(b, &l); err != nil {
		t.Fatal(err)
	}
	return l
}

const evento = `{"timestamp":"2026-10-29T10:00:00Z","tool":"qdd_audit","status":"success","details":null}`
const certificado = `{"timestamp":"2026-10-28T10:00:00Z","scores":{"context_score":80,"understanding_score":80,"improvement_score":80,"compliance_score":80},"total":80,"tendency":"stable"}`

func TestHistorial_MezcladoApartaLosEventosYCalculaSoloConCertificados_FND059(t *testing.T) {
	root := t.TempDir()
	escribirJSON(t, RutaHistorial(root), "["+evento+","+certificado+"]")
	cert, err := GenerateCognitiveCertificate(root, CognitiveScores{80, 80, 80, 80})
	if err != nil {
		t.Fatal(err)
	}
	if cert.Tendency != audit.TendencyStable {
		t.Fatalf("80 contra 80 es estable; el evento no cuenta como un certificado en 0: %s", cert.Tendency)
	}
	historial := leerLista(t, RutaHistorial(root))
	if len(historial) != 2 || historial[0]["scores"] == nil || historial[1]["scores"] == nil {
		t.Fatalf("el historial solo tiene certificados: %+v", historial)
	}
	eventos := leerLista(t, RutaEventosMCP(root))
	if len(eventos) != 1 || eventos[0]["tool"] != "qdd_audit" {
		t.Fatalf("el evento queda en el archivo del MCP, no se pierde: %+v", eventos)
	}
}

func TestHistorial_SepararDiceSiHabiaMezclaYNoTocaUnoLimpio_FND059(t *testing.T) {
	root := t.TempDir()
	escribirJSON(t, RutaHistorial(root), "["+certificado+"]")
	if mezclado, err := Mezclado(root); err != nil || mezclado {
		t.Fatalf("uno con solo certificados está limpio: %v %v", mezclado, err)
	}
	escribirJSON(t, RutaHistorial(root), "["+evento+","+certificado+"]")
	if mezclado, _ := Mezclado(root); !mezclado {
		t.Fatal("con un evento del MCP está mezclado")
	}
	if err := SepararEventosMCP(root); err != nil {
		t.Fatal(err)
	}
	if mezclado, _ := Mezclado(root); mezclado {
		t.Fatal("separado queda limpio")
	}
	if len(leerLista(t, RutaHistorial(root))) != 1 || len(leerLista(t, RutaEventosMCP(root))) != 1 {
		t.Fatal("cada entrada en su archivo")
	}
}
