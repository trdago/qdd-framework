package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoElseRule asegura que ningún archivo .go en el proyecto utilice
// la palabra reservada 'else', cumpliendo con la directriz de QDD CLEAN-01-NO-ELSE.
// Este test previene regresiones (FND-002).
func TestNoElseRule(t *testing.T) {
	// Obtener el directorio raíz del proyecto asumiendo que estamos en cli/cmd
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("No se pudo obtener directorio actual: %v", err)
	}
	
	// Navegar hacia arriba desde cli/cmd a la raíz del proyecto
	rootDir := filepath.Dir(filepath.Dir(cwd))

	violations := 0
	err = filepath.WalkDir(filepath.Join(rootDir, "cli"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		
		// Verificamos " el" + "se " para no disparar el test sobre esta misma línea!
		if strings.Contains(string(content), " el"+"se ") {
			t.Errorf("🚨 Regresión FND-002 detectada: Uso ilegal de 'else' en %s", path)
			violations++
		}
		return nil
	})
	
	if err != nil {
		t.Fatalf("Error al escanear archivos: %v", err)
	}
	
	if violations > 0 {
		t.Fatalf("El test falló con %d violaciones a la regla Cero Else.", violations)
	}
}
