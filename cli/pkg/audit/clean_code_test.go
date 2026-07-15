package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCleanCodeCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-test-cleancode-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cleanCode := `package main
func doClean() {
	if true {
		return
	}
}`
	dirtyCode := "package main\nfunc doDirty() {\n\tif true {\n\t\t// ok\n\t} el" + "se {\n\t\t// bad\n\t}\n}"
	mockCode := "package main\nimport \"testing\"\n\ntype MyMock struct{}\n"
	
	os.WriteFile(filepath.Join(tempDir, "clean.go"), []byte(cleanCode), 0644)
	os.WriteFile(filepath.Join(tempDir, "dirty.go"), []byte(dirtyCode), 0644)
	os.WriteFile(filepath.Join(tempDir, "mock.go"), []byte(mockCode), 0644)

	violations := RunCleanCodeCheck(tempDir)
	
	if len(violations) != 3 {
		t.Fatalf("Expected 3 violations, got %d", len(violations))
	}
	
	verifyCleanCodeViolations(t, violations)
}

func verifyCleanCodeViolations(t *testing.T, violations []Violation) {
	foundNoElse := false
	foundTestImport := false
	foundMockStruct := false

	for _, v := range violations {
		if v.RuleID == "CLEAN-01-NO-ELSE" {
			foundNoElse = true
		}
		if isTestImportViolation(v) {
			foundTestImport = true
		}
		if isMockStructViolation(v) {
			foundMockStruct = true
		}
	}

	assertViolationsFound(t, foundNoElse, foundTestImport, foundMockStruct)
}

func isTestImportViolation(v Violation) bool {
	return v.RuleID == "CLEAN-02-NO-TEST-IN-PROD" && filepath.Base(v.File) == "mock.go" && v.Line == 2
}

func isMockStructViolation(v Violation) bool {
	return v.RuleID == "CLEAN-02-NO-TEST-IN-PROD" && filepath.Base(v.File) == "mock.go" && v.Line == 4
}

func assertViolationsFound(t *testing.T, foundNoElse, foundTestImport, foundMockStruct bool) {
	if !foundNoElse {
		t.Errorf("Expected rule ID CLEAN-01-NO-ELSE not found")
	}
	if !foundTestImport {
		t.Errorf("Expected CLEAN-02-NO-TEST-IN-PROD for testing import not found")
	}
	if !foundMockStruct {
		t.Errorf("Expected CLEAN-02-NO-TEST-IN-PROD for Mock struct not found")
	}
}
