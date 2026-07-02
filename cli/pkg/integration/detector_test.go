package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsQDDProject(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "qdd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Should return false for empty directory
	if IsQDDProject(tempDir) {
		t.Errorf("expected IsQDDProject to return false for empty dir")
	}

	// Create a .qdd marker directory
	qddDir := filepath.Join(tempDir, ".qdd")
	err = os.Mkdir(qddDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .qdd dir: %v", err)
	}

	// Should return true now
	if !IsQDDProject(tempDir) {
		t.Errorf("expected IsQDDProject to return true after creating .qdd dir")
	}
}
