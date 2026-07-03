package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnpackCoreAssets_ReturnsErrorOnReadOnlyDir(t *testing.T) {
	// Create a temporary read-only directory to simulate MkdirAll/WriteFile failure
	tempDir := t.TempDir()
	
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0555) // Read and execute only, no write
	if err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}

	// Try to unpack core assets inside a directory that lacks write permissions.
	// We point qddDir to a nested path so MkdirAll tries to create a folder inside readOnlyDir.
	targetDir := filepath.Join(readOnlyDir, "nested_qdd")

	err = unpackCoreAssets(targetDir)
	
	if err == nil {
		t.Errorf("🚨 Regla violada (BUG PREVENT): unpackCoreAssets falló silenciosamente. Se esperaba un error al intentar escribir en un directorio de solo lectura, pero retornó nil.")
		return
	}
	
	t.Logf("Éxito: El error de I/O fue atrapado y propagado correctamente: %v", err)
}
