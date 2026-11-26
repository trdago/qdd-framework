package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FND-308: doctor revisa que el servidor qdd esté en .mcp.json y en opencode.json, y --fix lo agrega.
func TestDoctor_ServidorMCPEnClaudeYOpenCode_FND308(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".qdd"), 0o755)
	items := itemsMCP(dir)
	if len(items) != 2 {
		t.Fatalf("un ítem por IA: %+v", items)
	}
	for _, it := range items {
		if it.Success || !strings.Contains(it.Error, "doctor --fix") {
			t.Fatalf("sin el servidor, doctor lo dice y cómo arreglarlo: %+v", it)
		}
	}
	for _, it := range items {
		dispatchRepairCommand(it.Name, dir, filepath.Join(dir, ".qdd"))
	}
	for _, it := range itemsMCP(dir) {
		if !it.Success {
			t.Errorf("después de --fix: %+v", it)
		}
	}
}

func TestDoctor_ServidorMCPIlegibleNoSePisa_FND308(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte("{ roto"), 0o644)
	it := itemsMCP(dir)[0]
	if it.Success || !strings.Contains(it.Error, "JSON") {
		t.Fatalf("un .mcp.json roto se ve: %+v", it)
	}
}
