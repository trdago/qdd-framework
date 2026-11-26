package cmd

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/qdd-framework/qdd/pkg/qcl/benchmark"
)

// Rule: Prohibido usar 'else'
// FND-059: el MCP escribía sus eventos en cognitive_history.json y lo recortaba a 100 entradas: borraba los
// certificados del benchmark. Sus eventos van a su propio archivo.

func TestMCP_SusEventosNoBorranLosCertificados_FND059(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := benchmark.GenerateCognitiveCertificate(root, benchmark.CognitiveScores{ContextScore: 90}); err != nil {
		t.Fatal(err)
	}
	antes, err := os.ReadFile(benchmark.RutaHistorial(root))
	if err != nil {
		t.Fatal(err)
	}
	for range 150 {
		recordMCPActivity("success", "qdd_audit", nil)
	}
	despues, _ := os.ReadFile(benchmark.RutaHistorial(root))
	if string(antes) != string(despues) {
		t.Fatalf("los eventos del MCP no tocan el historial de certificados:\n%s", despues)
	}
	b, err := os.ReadFile(benchmark.RutaEventosMCP(root))
	if err != nil {
		t.Fatal(err)
	}
	var eventos []map[string]any
	if err := json.Unmarshal(b, &eventos); err != nil || len(eventos) != 100 || eventos[0]["tool"] != "qdd_audit" {
		t.Fatalf("los últimos 100 eventos, en su archivo: %d %v", len(eventos), err)
	}
}
