package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoreTemplatesUnpackAndContent(t *testing.T) {
	tempDir := t.TempDir()
	qddDir := filepath.Join(tempDir, ".qdd")
	err := os.MkdirAll(qddDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create qdd dir: %v", err)
	}

	err = unpackCoreAssets(qddDir)
	if err != nil {
		t.Fatalf("Failed to unpack core assets: %v", err)
	}

	expectedTemplates := []string{
		"finding.yaml",
		"adr.md",
		"goldenset.json",
		"resolution_certificate.md",
		"ai_risk_assessment.yaml",
		"soc2_processing_integrity.yaml",
	}

	for _, tmpl := range expectedTemplates {
		verifyTemplateExists(t, qddDir, tmpl)
	}
}

func verifyTemplateExists(t *testing.T, qddDir, templateName string) {
	templatePath := filepath.Join(qddDir, "core", "templates", templateName)
	info, err := os.Stat(templatePath)
	if err != nil {
		t.Fatalf("Expected core template %s to exist at %s: %v", templateName, templatePath, err)
	}

	if info.Size() == 0 {
		t.Fatalf("Core template %s is empty (0 bytes)", templateName)
	}
}
