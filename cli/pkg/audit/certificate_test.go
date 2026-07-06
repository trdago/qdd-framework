package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCertificate(t *testing.T) {
	tempDir := t.TempDir()

	// 1. First run, no history -> Stable
	cert1, err := GenerateCertificate(tempDir, []Violation{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cert1.Score != 100 {
		t.Errorf("Expected score 100, got %d", cert1.Score)
	}
	if cert1.Tendency != TendencyStable {
		t.Errorf("Expected tendency Stable, got %v", cert1.Tendency)
	}

	// 2. Second run, worse score -> Worsening
	cert2, err := GenerateCertificate(tempDir, []Violation{{RuleID: "V1"}, {RuleID: "V2"}})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cert2.Score >= 100 {
		t.Errorf("Expected lower score, got %d", cert2.Score)
	}
	if cert2.Tendency != TendencyWorsening {
		t.Errorf("Expected tendency Worsening, got %v", cert2.Tendency)
	}

	// 3. Third run, better score -> Improving
	cert3, err := GenerateCertificate(tempDir, []Violation{{RuleID: "V1"}})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cert3.Tendency != TendencyImproving {
		t.Errorf("Expected tendency Improving, got %v", cert3.Tendency)
	}

	// 4. Corrupt history file -> Error parsing
	historyPath := filepath.Join(tempDir, ".qdd", "project", "metrics", "certificate_history.json")
	os.WriteFile(historyPath, []byte("invalid json"), 0644)
	_, err = GenerateCertificate(tempDir, []Violation{})
	if err == nil {
		t.Errorf("Expected error when parsing invalid json")
	}

	// 5. Test negative score caps at 0
	violations := make([]Violation, 60) // 60 * 2 = 120 reduction -> score < 0
	os.Remove(historyPath)
	cert4, err := GenerateCertificate(tempDir, violations)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cert4.Score != 0 {
		t.Errorf("Expected negative score to cap at 0, got %d", cert4.Score)
	}
}

func TestGenerateCertificate_MkdirError(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a file where the directory should be, simulating an MkdirAll error
	metricsDir := filepath.Join(tempDir, ".qdd", "project", "metrics")
	os.MkdirAll(filepath.Dir(metricsDir), 0755)
	os.WriteFile(metricsDir, []byte("file instead of dir"), 0644)
	
	_, err := GenerateCertificate(tempDir, []Violation{})
	if err == nil {
		t.Errorf("Expected mkdir error, got nil")
	}
}
