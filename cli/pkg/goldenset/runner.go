package goldenset

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCase representa una definición determinista de un escenario (Golden Set).
type TestCase[In any, Out any] struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Input          In     `json:"input"`
	ExpectedOutput Out    `json:"expected_output"`
	ExpectedError  string `json:"expected_error"`
}

// RunSuite recorre iterativamente los archivos JSON de un feature y los ejecuta.
// handler: Es el código de la funcionalidad a evaluar.
func RunSuite[In any, Out any](t *testing.T, feature string, handler func(In) (Out, error)) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error obteniendo CWD: %v", err)
		return
	}

	rootPath := findProjectRoot(cwd)
	dir := filepath.Join(rootPath, ".qdd", "project", "goldensets", feature)

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("No se pudo leer el directorio de goldenset '%s': %v", dir, err)
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Error leyendo archivo de goldenset %s: %v", f.Name(), err)
			continue
		}

		var tc TestCase[In, Out]
		if err := json.Unmarshal(data, &tc); err != nil {
			t.Errorf("Error parseando JSON de %s: %v", f.Name(), err)
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			out, err := handler(tc.Input)
			
			if tc.ExpectedError != "" {
				if err == nil {
					t.Fatalf("Se esperaba error '%s' pero la ejecución fue exitosa", tc.ExpectedError)
					return
				}
				if !strings.Contains(err.Error(), tc.ExpectedError) {
					t.Fatalf("El error obtenido '%v' no coincide con el esperado '%s'", err, tc.ExpectedError)
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Error inesperado en la funcionalidad: %v", err)
				return
			}

			// Comparación determinista (JSON match)
			outJSON, _ := json.Marshal(out)
			expectedJSON, _ := json.Marshal(tc.ExpectedOutput)
			
			if string(outJSON) != string(expectedJSON) {
				t.Errorf("\n=== Mismatch ===\nEsperado: %s\nObtenido: %s\n", expectedJSON, outJSON)
			}
		})
	}
}

// findProjectRoot escala recursivamente hasta encontrar el directorio .qdd
func findProjectRoot(startPath string) string {
	current := startPath
	for {
		if _, err := os.Stat(filepath.Join(current, ".qdd")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return startPath // Fallback
		}
		current = parent
	}
}
