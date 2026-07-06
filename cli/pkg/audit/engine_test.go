package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEngineFormat(t *testing.T) {
	v1 := Violation{
		RuleID:      "TEST-01",
		Description: "Test without file",
	}
	if v1.Format() != "[TEST-01] Test without file" {
		t.Errorf("Format failed for base violation")
	}

	v2 := Violation{
		RuleID:      "TEST-02",
		Description: "Test with file",
		File:        "main.go",
	}
	if v2.Format() != "[TEST-02] Test with file -> main.go" {
		t.Errorf("Format failed for file violation")
	}

	v3 := Violation{
		RuleID:      "TEST-03",
		Description: "Test with line",
		File:        "main.go",
		Line:        42,
	}
	if v3.Format() != "[TEST-03] Test with line -> main.go:42" {
		t.Errorf("Format failed for line violation")
	}
}

func TestEngineCoordinator_RunAll(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create dummy policies to disable things and test conditional logic
	p := DefaultPolicies()
	p.ZeroElse = false // Disable NO-ELSE
	p.Enterprise = false
	p.BeyondLimits = false
	p.Traceability = false
	p.OWASP = false
	
	SavePolicies(tempDir, p)

	// Create a dummy file with an "else" statement
	os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
	os.WriteFile(filepath.Join(tempDir, "src", "bad.go"), []byte(`package main
func test() {
	if true {
	} else {
	}
}`), 0644)

	engine := NewEngine(tempDir)
	violations := engine.RunAll()
	
	// Because ZeroElse is false in policy, it should skip the CLEAN-01-NO-ELSE violation
	for _, v := range violations {
		if v.RuleID == "CLEAN-01-NO-ELSE" {
			t.Errorf("Expected CLEAN-01-NO-ELSE to be skipped when policy is false")
		}
	}
}
