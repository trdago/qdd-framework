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
	
	foundNoElse := false
	foundTestImport := false
	foundMockStruct := false

	for _, v := range violations {
		if v.RuleID == "CLEAN-01-NO-ELSE" {
			foundNoElse = true
		}
		if v.RuleID == "CLEAN-02-NO-TEST-IN-PROD" && filepath.Base(v.File) == "mock.go" {
			if v.Line == 2 {
				foundTestImport = true
			}
			if v.Line == 4 {
				foundMockStruct = true
			}
		}
	}

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
