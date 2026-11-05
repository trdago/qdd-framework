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
	if !strings.Contains(string(readme), instalarConNPM) {
		t.Error("README.md debe documentar la instalación por npm")
	}
}

// instalarConNPM: npm 12 bloquea el postinstall por defecto, y sin él qdd queda sin binario aunque npm termine bien.
const instalarConNPM = "npm install -g qdd-framework --allow-scripts=qdd-framework"

// FND-143: toda instrucción de instalación tiene que dejar qdd funcionando también con npm 12.
func TestInstrucciones_InstalanConLaBanderaDeNPM12_FND143(t *testing.T) {
	for _, ruta := range []string{"../../README.md", "../../npm/README.md", "../../docs/getting-started.md"} {
		b, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), instalarConNPM) {
			t.Errorf("%s no documenta %q", ruta, instalarConNPM)
		}
	}
	e2e, err := os.ReadFile("../../.github/workflows/e2e.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(e2e), "--allow-scripts=qdd-framework") {
		t.Error("el e2e instala como le decimos a la persona")
	}
}
