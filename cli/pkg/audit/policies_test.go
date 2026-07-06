package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicies(t *testing.T) {
	tempDir := t.TempDir()

	// 1. No existe el archivo -> Default
	p1 := LoadPolicies(tempDir)
	if !p1.ZeroElse {
		t.Errorf("Expected ZeroElse true as default, got false")
	}

	// 2. Archivo corrupto (yaml inválido) -> Default
	os.MkdirAll(filepath.Join(tempDir, ".qdd"), 0755)
	os.WriteFile(filepath.Join(tempDir, ".qdd", "policies.yaml"), []byte("invalid:\n  yaml: :"), 0644)
	
	p2 := LoadPolicies(tempDir)
	if !p2.ZeroElse {
		t.Errorf("Expected ZeroElse true as default on corrupt yaml")
	}

	// 3. Archivo válido
	validYaml := `
zero_else: false
owasp: false
`
	os.WriteFile(filepath.Join(tempDir, ".qdd", "policies.yaml"), []byte(validYaml), 0644)
	p3 := LoadPolicies(tempDir)
	if p3.ZeroElse {
		t.Errorf("Expected ZeroElse false from yaml")
	}
}

func TestSavePolicies(t *testing.T) {
	tempDir := t.TempDir()
	
	p := DefaultPolicies()
	p.ZeroElse = false
	
	err := SavePolicies(tempDir, p)
	if err != nil {
		t.Errorf("Error saving policies: %v", err)
	}
	
	// Read back
	p2 := LoadPolicies(tempDir)
	if p2.ZeroElse {
		t.Errorf("Expected ZeroElse false after save/load")
	}
}
