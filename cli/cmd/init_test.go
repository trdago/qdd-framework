package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectLanguagesRecursively(t *testing.T) {
	// Setup temporal directory
	tempDir, err := os.MkdirTemp("", "qdd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a nested project structure
	// tempDir/
	//   my-backend/
	//     go.mod
	//   my-frontend/
	//     package.json

	backendDir := filepath.Join(tempDir, "my-backend")
	err = os.Mkdir(backendDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}

	frontendDir := filepath.Join(tempDir, "my-frontend")
	err = os.Mkdir(frontendDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create frontend dir: %v", err)
	}

	err = os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	err = os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte("{}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Execution: scan from the root of tempDir
	languages := detectLanguages(tempDir)

	// Validation
	expected := []string{"Go", "Node"}
	
	// We sort or just check if it contains both since order doesn't strictly matter
	// but for simplicity, detectLanguages usually appends in a deterministic order (Go, then Node, then Java)
	if !reflect.DeepEqual(languages, expected) {
		t.Errorf("Expected detected languages %v, but got %v", expected, languages)
	}
}
