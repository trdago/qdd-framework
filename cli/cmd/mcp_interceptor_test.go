package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordMCPActivity(t *testing.T) {
	historyPath, cleanup := setupTestEnv(t)
	defer cleanup()

	recordMCPActivity("start", "qdd_test_telemetry", map[string]interface{}{"arg": "value"})

	verifyHistory(t, historyPath)
}

func setupTestEnv(t *testing.T) (string, func()) {
	cwd, _ := os.Getwd()
	metricsDir := filepath.Join(cwd, ".qdd", "project", "metrics")
	_ = os.MkdirAll(metricsDir, 0755)
	historyPath := filepath.Join(metricsDir, "cognitive_history.json")
	
	os.Remove(historyPath)
	return historyPath, func() { os.RemoveAll(filepath.Join(cwd, ".qdd")) }
}

func verifyHistory(t *testing.T, historyPath string) {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("Expected cognitive_history.json to be created, got error: %v", err)
	}

	var history []map[string]interface{}
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("Failed to parse cognitive_history.json: %v", err)
	}

	validateEvent(t, history)
}

func validateEvent(t *testing.T, history []map[string]interface{}) {
	if len(history) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(history))
	}
	if history[0]["tool"] != "qdd_test_telemetry" {
		t.Errorf("Expected tool qdd_test_telemetry, got %v", history[0]["tool"])
	}
	if history[0]["status"] != "start" {
		t.Errorf("Expected status start, got %v", history[0]["status"])
	}
}
