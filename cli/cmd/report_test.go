package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportGeneration(t *testing.T) {
	tempDir := t.TempDir()
	setupReportProject(t, tempDir)

	report := GenerateComprehensiveReport(tempDir)

	testReportDataIntegrity(t, report)
	testReportTextFormat(t, report)
	testReportMarkdownFormat(t, report)
	testReportJSONFormat(t, report)
	testReportFileSaving(t, tempDir, report)
}

func setupReportProject(t *testing.T, cwd string) {
	t.Helper()
	mustMkdir(t, filepath.Join(cwd, ".qdd", "project", "findings"))
	mustMkdir(t, filepath.Join(cwd, ".qdd", "project", "certification"))
	mustMkdir(t, filepath.Join(cwd, ".qdd", "project", "evidence", "reports"))

	configContent := "version: 1.0.0\ngovernance:\n  enabled: true\n"
	mustWrite(t, filepath.Join(cwd, ".qdd", "config.yaml"), configContent)

	findingContent := "id: FND-001\ntitle: Example Bug\nstatus: resolved\n"
	mustWrite(t, filepath.Join(cwd, ".qdd", "project", "findings", "FND-001.yaml"), findingContent)

	certContent := "id: CERT-001\ntitle: Base Structure\nstatus: certified\n"
	mustWrite(t, filepath.Join(cwd, ".qdd", "project", "certification", "CERT-001.yaml"), certContent)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("Failed to create dir %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func testReportDataIntegrity(t *testing.T, r ComprehensiveReport) {
	t.Helper()
	if r.Score < 0 || r.Score > 100 {
		t.Errorf("Invalid score in report: %d", r.Score)
	}
	if r.TotalCertifications != 1 {
		t.Errorf("Expected 1 total certification, got %d", r.TotalCertifications)
	}
	if r.CertifiedCount != 1 {
		t.Errorf("Expected 1 certified cert, got %d", r.CertifiedCount)
	}
	if r.ResolvedFindings != 1 {
		t.Errorf("Expected 1 resolved finding, got %d", r.ResolvedFindings)
	}
	if !r.DoctorHealthy {
		t.Errorf("Expected Doctor to report healthy state")
	}
	if r.LocalLLM.Provider == "" {
		t.Errorf("Expected non-empty LocalLLM provider")
	}
}

func testReportTextFormat(t *testing.T, r ComprehensiveReport) {
	t.Helper()
	textOut := formatReportOutput(r, "text")
	if !strings.Contains(textOut, "REPORTE INTEGRAL QDD FRAMEWORK") {
		t.Errorf("Text report missing header banner")
	}
	if !strings.Contains(textOut, "Lo que no se mide no se mejora") {
		t.Errorf("Text report missing philosophical motto")
	}
	if !strings.Contains(textOut, "Score Global:") {
		t.Errorf("Text report missing Score Global")
	}
}

func testReportMarkdownFormat(t *testing.T, r ComprehensiveReport) {
	t.Helper()
	mdOut := formatReportOutput(r, "md")
	if !strings.Contains(mdOut, "# Reporte Integral de Calidad y Gobernanza QDD") {
		t.Errorf("Markdown report missing H1 header")
	}
	if !strings.Contains(mdOut, "Lo que no se mide no se mejora") {
		t.Errorf("Markdown report missing philosophical motto")
	}
	if !strings.Contains(mdOut, "## Resumen Ejecutivo de Métricas") {
		t.Errorf("Markdown report missing executive table")
	}
	if !strings.Contains(mdOut, "Poder Cognitivo Local (LLM)") {
		t.Errorf("Markdown report missing Poder Cognitivo Local section")
	}
}

func testReportJSONFormat(t *testing.T, r ComprehensiveReport) {
	t.Helper()
	jsonOut := formatReportOutput(r, "json")
	var parsed ComprehensiveReport
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("JSON report is not valid JSON: %v", err)
	}
	if parsed.Score != r.Score {
		t.Errorf("Parsed JSON score %d does not match original %d", parsed.Score, r.Score)
	}
}

func testReportFileSaving(t *testing.T, cwd string, r ComprehensiveReport) {
	t.Helper()
	customPath := filepath.Join(cwd, "custom_report.json")
	jsonContent := formatReportOutput(r, "json")
	handleReportFileSaving(cwd, jsonContent, customPath, "json")

	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("Failed to save report to custom path %s", customPath)
	}
}
