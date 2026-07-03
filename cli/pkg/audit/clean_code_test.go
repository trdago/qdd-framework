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
	
	os.WriteFile(filepath.Join(tempDir, "clean.go"), []byte(cleanCode), 0644)
	os.WriteFile(filepath.Join(tempDir, "dirty.go"), []byte(dirtyCode), 0644)

	violations := RunCleanCodeCheck(tempDir)
	
	if len(violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(violations))
	}
	
	if violations[0].RuleID != "CLEAN-01-NO-ELSE" {
		t.Errorf("Expected rule ID CLEAN-01-NO-ELSE, got %s", violations[0].RuleID)
	}
}
