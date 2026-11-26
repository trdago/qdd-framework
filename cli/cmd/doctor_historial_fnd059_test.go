package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/qcl/benchmark"
)

// Rule: Prohibido usar 'else'
// FND-059: doctor validaba un «Contrato de Historial» que solo pedía un array. Uno con eventos del MCP mezclados entre
// los certificados falla, y --fix los pasa a mcp_eventos.json sin perder ninguno.

func TestDoctor_HistorialConEventosDelMCPFallaYFixLosSepara_FND059(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	mezcla := `[{"timestamp":"2026-10-29T10:00:00Z","tool":"qdd_audit","status":"success"},` +
		`{"timestamp":"2026-10-28T10:00:00Z","scores":{"context_score":80},"total":80,"tendency":"Estable"}]`
	if err := os.MkdirAll(filepath.Dir(benchmark.RutaHistorial(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(benchmark.RutaHistorial(root), []byte(mezcla), 0o644); err != nil {
		t.Fatal(err)
	}
	item := buildHistoryCheck(root)
	if item.Success || !strings.Contains(item.Error, "MCP") {
		t.Fatalf("un historial mezclado falla y dice por qué: %+v", item)
	}
	repairHistory(root, filepath.Join(root, ".qdd"))
	if item := buildHistoryCheck(root); !item.Success {
		t.Fatalf("después de --fix queda bien: %+v", item)
	}
	eventos, err := benchmark.LeerEventosMCP(root)
	if err != nil || len(eventos) != 1 {
		t.Fatalf("el evento queda en el registro del MCP: %d %v", len(eventos), err)
	}
	b, _ := os.ReadFile(benchmark.RutaHistorial(root))
	if !strings.Contains(string(b), `"scores"`) || strings.Contains(string(b), `"tool"`) {
		t.Fatalf("el certificado se queda; el evento sale: %s", b)
	}
}
