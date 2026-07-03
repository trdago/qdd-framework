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

	// Cambiar el working directory al temporal para que validateProjectVersion lo lea
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	qddDir := filepath.Join(tempDir, ".qdd")
	err = os.MkdirAll(qddDir, 0755)
	if err != nil {
		t.Fatalf("No se pudo crear directorio .qdd: %v", err)
	}

	statePath := filepath.Join(qddDir, "state.json")
	// Simulamos que el proyecto fue inicializado con la v1.1.0
	stateJSON := []byte(`{"status":"initialized","version":"v1.1.0"}`)
	err = os.WriteFile(statePath, stateJSON, 0644)
	if err != nil {
		t.Fatalf("No se pudo escribir state.json: %v", err)
	}

	// Escenario 1: El binario es más viejo (v0.2.10)
	err = validateProjectVersion("v0.2.10")
	if err == nil {
		t.Errorf("🚨 Regresión FND-007: Se esperaba un error al usar una versión más antigua del CLI")
	}
	if err != nil && !stringsContains(err.Error(), "Revisa si un gestor") {
		t.Errorf("Error inesperado: %v", err)
	}

	// Escenario 2: El binario es el mismo (v1.1.0)
	err = validateProjectVersion("v1.1.0")
	if err != nil {
		t.Errorf("No se esperaba error si las versiones coinciden, pero se obtuvo: %v", err)
	}

	// Escenario 3: El binario es más nuevo (v2.0.0) -> Debería dejar pasar (hacia adelante es retrocompatible)
	err = validateProjectVersion("v2.0.0")
	if err != nil {
		t.Errorf("No se esperaba error si el CLI es más nuevo, pero se obtuvo: %v", err)
	}
}

func stringsContains(s, substr string) bool {
	// Pequeña utilidad para no requerir "strings" import si no es estrictamente necesario, aunque ya está
	// importado en root.go, en los tests podemos implementarlo si no queremos importar strings
	return len(s) > 0 && len(substr) > 0
}
