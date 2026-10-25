package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FND-048: el README documentaba un `go install` que Go rechaza, porque el
// módulo se declara como github.com/qdd-framework/qdd y no como la ruta del repo.
func TestReadme_InstalacionGoNoDocumentaRutaInvalida(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("no se pudo leer README.md: %v", err)
	}
	gomod, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatalf("no se pudo leer go.mod: %v", err)
	}
	modulo := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(string(gomod), "\n", 2)[0], "module "))

	for i, linea := range strings.Split(string(readme), "\n") {
		if !strings.Contains(linea, "go install ") {
			continue
		}
		if !strings.Contains(linea, modulo) {
			t.Errorf("README.md:%d documenta %q, pero el módulo es %s y Go rechazará la instalación", i+1, strings.TrimSpace(linea), modulo)
		}
	}
	if !strings.Contains(string(readme), "npm install -g qdd-framework") {
		t.Error("README.md debe documentar la instalación por npm")
	}
}
