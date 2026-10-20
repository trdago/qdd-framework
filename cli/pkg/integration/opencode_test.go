package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenCodeAdapter_Sync(t *testing.T) {
	tempDir := t.TempDir()

	adapter := &OpenCodeAdapter{}
	err := adapter.Sync(tempDir)
	if err != nil {
		t.Fatalf("Expected no error syncing OpenCode adapter, got: %v", err)
	}

	agentsMD := filepath.Join(tempDir, "AGENTS.md")
	if _, err := os.Stat(agentsMD); os.IsNotExist(err) {
		t.Fatalf("AGENTS.md should have been created")
	}

	agentRules := filepath.Join(tempDir, ".agentrules")
	if _, err := os.Stat(agentRules); os.IsNotExist(err) {
		t.Fatalf(".agentrules should have been created")
	}

	opencodeQDD := filepath.Join(tempDir, ".opencode", "qdd.md")
	if _, err := os.Stat(opencodeQDD); os.IsNotExist(err) {
		t.Fatalf(".opencode/qdd.md should have been created")
	}
}
