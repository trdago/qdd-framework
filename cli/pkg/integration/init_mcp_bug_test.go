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

	// Si no retorna algo que contiene separadores (como '/' en linux/mac o '\' en windows),
	// significa que hizo fallback a "qdd", lo cual solo es aceptable si exec.LookPath falla
	// y el entorno es un build temporal.
	if path == "qdd" {
		_, err := exec.LookPath("qdd")
		if err == nil {
			t.Errorf("🚨 Regla violada (BUG PREVENT): resolveQDDPath hizo fallback a 'qdd' a pesar de que exec.LookPath encontró el binario.")
		} else {
			// Es aceptable en un entorno CI donde 'qdd' no está en el PATH ni ejecutamos como binario
			t.Logf("Fallback a 'qdd' aceptado porque exec.LookPath no encontró el binario.")
		}
		return
	}

	// Si retornó una ruta, no debe ser de tmp de "go run" a menos que todo falle.
	// Pero el método ya filtra "tmp" en caso de os.Executable(), cayendo a LookPath.
	// Si cayó a LookPath y encontró un tmp, es raro, pero la validación principal es que no retorne "qdd" estático sin motivo.
	if strings.Contains(path, "/tmp/") || strings.Contains(path, "\\Temp\\") {
		// Validar si es porque estamos forzosamente en "go test" 
		// y LookPath falló. "go test" usa /tmp/, pero la función lo filtra,
		// excepto si exec.LookPath retorna algo en tmp.
		t.Logf("El path retornado es temporal: %s", path)
	}

	if !filepath.IsAbs(path) && path != "qdd" {
		// exec.LookPath y os.Executable generalmente retornan rutas absolutas
		t.Logf("Advertencia: El path no parece absoluto, pero es diferente al estático: %s", path)
	}
}
