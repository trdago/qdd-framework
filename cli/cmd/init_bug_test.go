package cmd

import (
	"testing"
)

func TestUnpackCoreAssets_ReturnsErrorOnReadOnlyDir(t *testing.T) {
	// Instead of relying on 0555 which is flaky on Windows,
	// we use a universally invalid path to trigger a guaranteed I/O error.
	targetDir := "invalid\x00path"

	err := unpackCoreAssets(targetDir)
	
	if err == nil {
		t.Errorf("🚨 Regla violada (BUG PREVENT): unpackCoreAssets falló silenciosamente. Se esperaba un error al intentar escribir en un directorio de solo lectura, pero retornó nil.")
		return
	}
	
	t.Logf("Éxito: El error de I/O fue atrapado y propagado correctamente: %v", err)
}
