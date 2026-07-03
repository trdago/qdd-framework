package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// TestFND004StateVersionSynchronization verifica que createStateFile asigne la versión dinámica correcta del CLI
// y no escriba un valor estático/hardcodeado en state.json.
// Cumple con la Regla Global: "todos los bugs encontrados se tienen que documentar y generar test unitario".
func TestFND004StateVersionSynchronization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-test-fnd004-*")
	if err != nil {
		t.Fatalf("No se pudo crear directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	qddDir := filepath.Join(tempDir, ".qdd")
	err = os.MkdirAll(qddDir, 0755)
	if err != nil {
		t.Fatalf("No se pudo crear directorio .qdd: %v", err)
	}

	expectedVersion := "v9.9.9-test"
	originalVersion := rootCmd.Version
	rootCmd.Version = expectedVersion
	defer func() { rootCmd.Version = originalVersion }()

	err = createStateFile(qddDir)
	if err != nil {
		t.Fatalf("createStateFile falló: %v", err)
	}

	statePath := filepath.Join(qddDir, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("No se pudo leer state.json: %v", err)
	}

	// Basic check using string matching since we haven't imported encoding/json
	importCheck := string(data)
	if !strings.Contains(importCheck, expectedVersion) {
		t.Errorf("🚨 Regresión FND-004: La versión en state.json no contiene la dinámica '%v'", expectedVersion)
	}
}

// TestFND006InitVersionUpdate asegura que si state.json ya existe, qdd init actualiza la versión y mantiene el estado.
func TestFND006InitVersionUpdate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-test-fnd006-*")
	if err != nil {
		t.Fatalf("No se pudo crear directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	qddDir := filepath.Join(tempDir, ".qdd")
	err = os.MkdirAll(qddDir, 0755)
	if err != nil {
		t.Fatalf("No se pudo crear directorio .qdd: %v", err)
	}

	statePath := filepath.Join(qddDir, "state.json")
	oldState := []byte(`{"status":"old","version":"v0.1.0"}`)
	err = os.WriteFile(statePath, oldState, 0644)
	if err != nil {
		t.Fatalf("No se pudo escribir state antiguo: %v", err)
	}

	expectedVersion := "v2.0.0-test"
	originalVersion := rootCmd.Version
	rootCmd.Version = expectedVersion
	defer func() { rootCmd.Version = originalVersion }()

	err = createStateFile(qddDir)
	if err != nil {
		t.Fatalf("createStateFile falló: %v", err)
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("No se pudo leer state.json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, expectedVersion) {
		t.Errorf("🚨 Regresión FND-006: La versión no fue actualizada. Contenido: %s", content)
	}
	if !strings.Contains(content, "old") {
		t.Errorf("🚨 Regresión FND-006: Se borró el status anterior. Contenido: %s", content)
	}
}
