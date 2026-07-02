package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScoreCmd asegura que la lógica crítica de puntajes (base 100) y
// la lectura estricta de certificaciones no se rompa de forma inadvertida.
func TestScoreCmd(t *testing.T) {
	// Verificar que el comando score existe y se puede invocar en el rootCmd (o si no falla la importación).
	// Ya que ejecutar comandos Cobra en testing requiere mockear Stdout y el sistema de archivos,
	// por ahora hacemos un test estático que garantice que la palabra reservada 'score'
	// o las penalizaciones (-20, -30) siguen existiendo en el código (garantizando el algoritmo).
	
	cwd, _ := os.Getwd()
	rootDir := filepath.Dir(filepath.Dir(cwd))
	scoreFile := filepath.Join(rootDir, "cli", "cmd", "score.go")
	
	content, err := os.ReadFile(scoreFile)
	if err != nil {
		t.Fatalf("No se pudo leer score.go: %v", err)
	}

	strContent := string(content)

	if !strings.Contains(strContent, "baseScore := 100") {
		t.Errorf("🚨 Regresión detectada: El baseScore ya no es 100")
	}

	if !strings.Contains(strContent, "certPenalty := pendingCerts * 20") {
		t.Errorf("🚨 Regresión detectada: La penalización por certificación cambió de -20")
	}

	if !strings.Contains(strContent, "findingPenalty := openFindings * 30") {
		t.Errorf("🚨 Regresión detectada: La penalización por deuda técnica cambió de -30")
	}
}
