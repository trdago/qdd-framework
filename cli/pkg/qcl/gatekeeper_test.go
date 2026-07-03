package qcl

import (
	"os"
	"testing"
)

func TestCheckMinimumAlignment(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "qdd-test-gatekeeper-*")
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	err := CheckMinimumAlignment()
	if err == nil {
		t.Errorf("Expected error when .qdd is missing")
	}

	os.MkdirAll(".qdd", 0755)
	configYAML := `
project: test
languages:
  - go
databases: []
architecture: serverless
`
	os.WriteFile(".qdd/config.yaml", []byte(configYAML), 0644)
	err = CheckMinimumAlignment()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
    
    configYAMLMissing := `
project: test
languages: []
architecture: serverless
`
	os.WriteFile(".qdd/config.yaml", []byte(configYAMLMissing), 0644)
	err = CheckMinimumAlignment()
	if err == nil {
		t.Errorf("Expected error when languages are missing")
	}
}
