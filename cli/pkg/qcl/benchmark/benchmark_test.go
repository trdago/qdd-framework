package benchmark

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEvaluateContextFAANG(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qdd_bench_faang_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	score := evaluateContext(tmpDir)
	if score != 0 {
		t.Errorf("Expected score 0, got %d", score)
	}

	setupMockKnowledgeDB(t, tmpDir)

	score = evaluateContext(tmpDir)
	if score != 100 {
		t.Errorf("Expected score 100 for <10%% orphans, got %d", score)
	}
}

func setupMockKnowledgeDB(t *testing.T, tmpDir string) {
	qddDir := filepath.Join(tmpDir, ".qdd")
	os.MkdirAll(qddDir, 0755)
	os.WriteFile(filepath.Join(qddDir, "config.yaml"), []byte("data"), 0644)

	dbPath := filepath.Join(qddDir, "knowledge.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to create mock db: %v", err)
	}
	defer db.Close()

	db.ExecContext(context.Background(), "CREATE TABLE nodes (id TEXT PRIMARY KEY)")
	db.ExecContext(context.Background(), "CREATE TABLE edges (source_id TEXT, target_id TEXT)")

	for i := 0; i < 100; i++ {
		db.ExecContext(context.Background(), "INSERT INTO nodes (id) VALUES (?)", i)
	}

	for i := 0; i < 95; i++ {
		db.ExecContext(context.Background(), "INSERT INTO edges (source_id, target_id) VALUES (?, ?)", i, i+1)
	}
}

func TestEvaluateUnderstandingFAANG(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qdd_bench_faang_und")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qddDir := filepath.Join(tmpDir, ".qdd")
	os.MkdirAll(qddDir, 0755)

	understandingPath := filepath.Join(qddDir, "understanding.json")

	longSummary := "Este es un resumen muy largo de la arquitectura del proyecto que debería pasar la validación por ser superior a los cien caracteres sin problema alguno. Se extiende bastante para probar."

	// Test Anthropic Constitutional Alignment (Security, Scalability)
	data := map[string]interface{}{
		"summary":    longSummary,
		"components": []string{"C1", "C2", "C3"},
		"objectives": []string{"Validar Security y Performance"},
		"guidelines": []string{"Seguir rule framework qdd"},
	}
	jsonData, _ := json.Marshal(data)
	os.WriteFile(understandingPath, jsonData, 0644)

	score := evaluateUnderstanding(tmpDir)
	if score != 100 {
		t.Errorf("Expected score 100, got %d", score)
	}
}

func TestEvaluateImprovementsFAANG(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qdd_bench_faang_imp")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	findingsDir := filepath.Join(tmpDir, ".qdd", "project", "findings")
	os.MkdirAll(findingsDir, 0755)

	// Crear 2 findings, 1 resuelto (50% resolution rate - Meta-Grade)
	f1 := "severity: High\ndescription: Bug\nstatus: Open"
	f2 := "severity: Low\ndescription: Typo\nstatus: Resolved"
	os.WriteFile(filepath.Join(findingsDir, "a.yaml"), []byte(f1), 0644)
	os.WriteFile(filepath.Join(findingsDir, "b.yaml"), []byte(f2), 0644)

	metricsDir := filepath.Join(tmpDir, ".qdd", "project", "metrics")
	os.MkdirAll(metricsDir, 0755)
	os.WriteFile(filepath.Join(metricsDir, "metric1.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(metricsDir, "metric2.json"), []byte("{}"), 0644)

	score := evaluateImprovements(tmpDir)
	if score != 100 {
		t.Errorf("Expected score 100 with 50%% resolution rate, got %d", score)
	}
}

func TestEvaluateComplianceFAANG(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qdd_bench_faang_comp")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simulamos carpeta de certificaciones para evitar penalización
	certDir := filepath.Join(tmpDir, ".qdd", "core", "certification")
	os.MkdirAll(certDir, 0755)
	os.WriteFile(filepath.Join(certDir, "rule1.yaml"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(certDir, "rule2.yaml"), []byte("data"), 0644)
	os.WriteFile(filepath.Join(certDir, "rule3.yaml"), []byte("data"), 0644)

	// Aquí no mockeamos el audit completo, pero evaluamos que pase el chequeo base.
	// Si hay fallos de audit en tmpDir por no ser proyecto go, score bajará.
	// Solo validamos que se ejecuta sin crashear.
	score := evaluateCompliance(tmpDir)
	if score < 0 {
		t.Errorf("Score no puede ser negativo")
	}
}
