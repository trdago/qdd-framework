package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveQDDPath_ReturnsAbsoluteOrExecutable verifica que resolveQDDPath
// retorne una ruta absoluta (o al menos diferente a simplemente "qdd") si el
// binario o ejecutable está disponible. Esto previene la regresión del bug
// donde el MCP fallaba al inicializar en los IDEs porque "command": "qdd"
// no existía en el $PATH acotado de la interfaz gráfica del editor.
func TestResolveQDDPath_ReturnsAbsoluteOrExecutable(t *testing.T) {
	path := resolveQDDPath()

	if path == "qdd" {
		verifyFallbackBehavior(t)
		return
	}

	verifyResolvedPath(t, path)
}

func verifyFallbackBehavior(t *testing.T) {
	_, err := exec.LookPath("qdd")
	if err == nil {
		t.Errorf("🚨 Regla violada (BUG PREVENT): resolveQDDPath hizo fallback a 'qdd' a pesar de que exec.LookPath encontró el binario.")
		return
	}
	
	t.Logf("Fallback a 'qdd' aceptado porque exec.LookPath no encontró el binario.")
}

func verifyResolvedPath(t *testing.T, path string) {
	if strings.Contains(path, "/tmp/") || strings.Contains(path, "\\Temp\\") {
		t.Logf("El path retornado es temporal: %s", path)
	}

	if !filepath.IsAbs(path) && path != "qdd" {
		t.Logf("Advertencia: El path no parece absoluto, pero es diferente al estático: %s", path)
	}
}
