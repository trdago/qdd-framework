package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateCertificate(t *testing.T) {
	// Setup a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "qdd_cert_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test 1: No previous history, 0 violations (Perfect Score)
	var violations []Violation // 0 violations
	cert, err := GenerateCertificate(tmpDir, violations)
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}
	
	if cert.Score != 100 {
		t.Errorf("Expected score 100, got %d", cert.Score)
	}
	if cert.Tendency != TendencyStable {
		t.Errorf("Expected tendency Stable, got %s", cert.Tendency)
	}

	// Test 2: Worsening trend (adding a violation)
	violations = append(violations, Violation{
		Category:    "OWASP",
		RuleID:      "TEST-01",
		Description: "Test violation",
	})
	
	cert2, err := GenerateCertificate(tmpDir, violations)
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}
	
	if cert2.Score != 98 {
		t.Errorf("Expected score 98, got %d", cert2.Score)
	}
	if cert2.Tendency != TendencyWorsening {
		t.Errorf("Expected tendency Worsening, got %s", cert2.Tendency)
	}

	// Test 3: Improving trend (removing a violation)
	var betterViolations []Violation
	cert3, err := GenerateCertificate(tmpDir, betterViolations)
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}
	
	if cert3.Score != 100 {
		t.Errorf("Expected score 100, got %d", cert3.Score)
	}
	if cert3.Tendency != TendencyImproving {
		t.Errorf("Expected tendency Improving, got %s", cert3.Tendency)
	}
	
	// Test 4: Check if history file was written correctly
	historyPath := filepath.Join(tmpDir, ".qdd", "project", "metrics", "certificate_history.json")
	data, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("Failed to read history file: %v", err)
	}
	
	var history []Certificate
	if err := json.Unmarshal(data, &history); err != nil {
		t.Fatalf("Failed to parse history file: %v", err)
	}
	
	if len(history) != 3 {
		t.Errorf("Expected 3 history items, got %d", len(history))
	}
}
