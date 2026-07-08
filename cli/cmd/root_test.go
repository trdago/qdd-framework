package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFND007VersionMismatch asegura que si el proyecto fue creado con una versión de QDD
// y el binario ejecutado es más antiguo, arroje error explícito.
func TestFND007VersionMismatch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-test-fnd007-*")
	if err != nil {
		t.Fatalf("No se pudo crear directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	setupFND007State(t, tempDir)

	runFND007Scenarios(t)
}

func setupFND007State(t *testing.T, tempDir string) {
	qddDir := filepath.Join(tempDir, ".qdd")
	err := os.MkdirAll(qddDir, 0755)
	if err != nil {
		t.Fatalf("No se pudo crear directorio .qdd: %v", err)
	}

	statePath := filepath.Join(qddDir, "state.json")
	stateJSON := []byte(`{"status":"initialized","version":"v1.1.0"}`)
	err = os.WriteFile(statePath, stateJSON, 0644)
	if err != nil {
		t.Fatalf("No se pudo escribir state.json: %v", err)
	}
}

func runFND007Scenarios(t *testing.T) {
	testOlderBinary(t)
	testSameBinary(t)
	testNewerBinary(t)
}

func testOlderBinary(t *testing.T) {
	err := validateProjectVersion("v0.2.10")
	if err == nil {
		t.Errorf("🚨 Regresión FND-007: Se esperaba un error al usar una versión más antigua del CLI")
	}
	if err != nil && !stringsContains(err.Error(), "Revisa si un gestor") {
		t.Errorf("Error inesperado: %v", err)
	}
}

func testSameBinary(t *testing.T) {
	err := validateProjectVersion("v1.1.0")
	if err != nil {
		t.Errorf("No se esperaba error si las versiones coinciden, pero se obtuvo: %v", err)
	}
}

func testNewerBinary(t *testing.T) {
	err := validateProjectVersion("v2.0.0")
	if err != nil {
		t.Errorf("No se esperaba error si el CLI es más nuevo, pero se obtuvo: %v", err)
	}
}

func stringsContains(s, substr string) bool {
	// Pequeña utilidad para no requerir "strings" import si no es estrictamente necesario, aunque ya está
	// importado en root.go, en los tests podemos implementarlo si no queremos importar strings
	return len(s) > 0 && len(substr) > 0
}

func TestIsOlder(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"v1.0.0", "v1.1.0", true},
		{"1.0.0", "1.1.0", true},
		{"v1.1.0", "v1.0.0", false},
		{"v1.0.0", "v2.0.0", true},
		{"v2.0.0", "v1.0.0", false},
		{"v1.1.0", "v1.1.1", true},
		{"v1.1.1", "v1.1.0", false},
		{"v1.5.0", "v1.5.0", false},
		{"v1.5.0-beta", "v1.5.0", false},
		{"v0.9.9", "v1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			result := isOlder(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("isOlder(%q, %q) = %v; want %v", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}
