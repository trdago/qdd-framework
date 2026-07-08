package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCursorAdapter_Sync_DeepMerge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qdd_cursor_test")
	if err != nil {
		t.Fatalf("Error creando tmpDir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mcpPath := setupCursorTestWorkspace(t, tmpDir)
	
	adapter := &CursorAdapter{}
	err = adapter.Sync(tmpDir)
	if err != nil {
		t.Fatalf("Cursor Sync falló: %v", err)
	}

	verifyCursorDeepMerge(t, mcpPath)
}

func setupCursorTestWorkspace(t *testing.T, tmpDir string) string {
	cursorDir := filepath.Join(tmpDir, ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		t.Fatalf("Error creando .cursor dir: %v", err)
	}

	mcpPath := filepath.Join(cursorDir, "mcp.json")

	initialState := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"otro_server": map[string]interface{}{
				"command": "python",
				"args":    []string{"script.py"},
			},
		},
		"global_setting": true,
	}

	initialBytes, _ := json.Marshal(initialState)
	os.WriteFile(mcpPath, initialBytes, 0644)
	
	return mcpPath
}

func verifyCursorDeepMerge(t *testing.T, mcpPath string) {
	finalState := readFinalState(t, mcpPath)

	verifyGlobalSetting(t, finalState)

	servers := extractServersMap(t, finalState)

	verifyServers(t, servers)
}

func readFinalState(t *testing.T, mcpPath string) map[string]interface{} {
	finalBytes, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("Error leyendo %s: %v", mcpPath, err)
	}

	var finalState map[string]interface{}
	json.Unmarshal(finalBytes, &finalState)
	return finalState
}

func verifyGlobalSetting(t *testing.T, finalState map[string]interface{}) {
	if finalState["global_setting"] != true {
		t.Errorf("Se destruyó la propiedad 'global_setting'")
	}
}

func extractServersMap(t *testing.T, finalState map[string]interface{}) map[string]interface{} {
	servers, ok := finalState["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("La clave mcpServers se corrompió")
	}
	return servers
}

func verifyServers(t *testing.T, servers map[string]interface{}) {
	if _, ok := servers["otro_server"]; !ok {
		t.Errorf("Se eliminó el servidor 'otro_server' ajeno a QDD (Destructive Write!)")
	}

	if _, ok := servers["qdd"]; !ok {
		t.Errorf("No se inyectó el servidor 'qdd'")
	}
}
