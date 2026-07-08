package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDetectLanguagesRecursively(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	setupNestedProjectStructure(t, tempDir)

	languages := detectLanguages(tempDir)

	validateDetectedLanguages(t, languages)
}

func setupNestedProjectStructure(t *testing.T, tempDir string) {
	backendDir := filepath.Join(tempDir, "my-backend")
	if err := os.Mkdir(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create backend dir: %v", err)
	}

	frontendDir := filepath.Join(tempDir, "my-frontend")
	if err := os.Mkdir(frontendDir, 0755); err != nil {
		t.Fatalf("Failed to create frontend dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(backendDir, "go.mod"), []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	if err := os.WriteFile(filepath.Join(frontendDir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
}

func validateDetectedLanguages(t *testing.T, languages []string) {
	expected := []string{"Go", "Node"}
	
	if !reflect.DeepEqual(languages, expected) {
		t.Errorf("Expected detected languages %v, but got %v", expected, languages)
	}
}

// TestFND004StateVersionSynchronization verifica que createStateFile asigne la versión dinámica correcta del CLI
// y no escriba un valor estático/hardcodeado en state.json.
// Cumple con la Regla Global: "todos los bugs encontrados se tienen que documentar y generar test unitario".
func TestFND004StateVersionSynchronization(t *testing.T) {
	tempDir, qddDir := setupInitTestDir(t, "fnd004")
	defer os.RemoveAll(tempDir)

	expectedVersion := "v9.9.9-test"
	originalVersion := rootCmd.Version
	rootCmd.Version = expectedVersion
	defer func() { rootCmd.Version = originalVersion }()

	if err := createStateFile(qddDir); err != nil {
		t.Fatalf("createStateFile falló: %v", err)
	}

	validateStateFileContains(t, qddDir, expectedVersion, "FND-004")
}

// TestFND006InitVersionUpdate asegura que si state.json ya existe, qdd init actualiza la versión y mantiene el estado.
func TestFND006InitVersionUpdate(t *testing.T) {
	tempDir, qddDir := setupInitTestDir(t, "fnd006")
	defer os.RemoveAll(tempDir)

	statePath := filepath.Join(qddDir, "state.json")
	oldState := []byte(`{"status":"old","version":"v0.1.0"}`)
	if err := os.WriteFile(statePath, oldState, 0644); err != nil {
		t.Fatalf("No se pudo escribir state antiguo: %v", err)
	}

	expectedVersion := "v2.0.0-test"
	originalVersion := rootCmd.Version
	rootCmd.Version = expectedVersion
	defer func() { rootCmd.Version = originalVersion }()

	if err := createStateFile(qddDir); err != nil {
		t.Fatalf("createStateFile falló: %v", err)
	}

	validateStateFileContains(t, qddDir, expectedVersion, "FND-006")
	validateStateFileContains(t, qddDir, "old", "FND-006 (status)")
}

func setupInitTestDir(t *testing.T, prefix string) (string, string) {
	tempDir, err := os.MkdirTemp("", "qdd-test-"+prefix+"-*")
	if err != nil {
		t.Fatalf("No se pudo crear directorio temporal: %v", err)
	}

	qddDir := filepath.Join(tempDir, ".qdd")
	if err := os.MkdirAll(qddDir, 0755); err != nil {
		t.Fatalf("No se pudo crear directorio .qdd: %v", err)
	}
	return tempDir, qddDir
}

func validateStateFileContains(t *testing.T, qddDir, expectedStr, errorCode string) {
	statePath := filepath.Join(qddDir, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("No se pudo leer state.json: %v", err)
	}

	if !strings.Contains(string(data), expectedStr) {
		t.Errorf("🚨 Regresión %s: state.json no contiene '%v'", errorCode, expectedStr)
	}
}
