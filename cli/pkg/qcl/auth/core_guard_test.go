package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardCoreWriteAccess_DoctorAuthorized(t *testing.T) {
	// Backup original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Simulamos comando "qdd doctor --fix"
	os.Args = []string{"qdd", "doctor", "--fix"}

	targetPath := filepath.Join(".qdd", "core", "certification", "rule_test.yaml")
	
	err := GuardCoreWriteAccess(targetPath)
	if err != nil {
		t.Errorf("Se esperaba autorización para 'doctor', pero dio error: %v", err)
	}
}

func TestGuardCoreWriteAccess_ValidateDenied(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Simulamos comando "qdd validate" intentando escribir en el core
	os.Args = []string{"qdd", "validate"}

	targetPath := filepath.Join(".qdd", "core", "certification", "rule_test.yaml")
	
	err := GuardCoreWriteAccess(targetPath)
	if err == nil {
		t.Errorf("Se esperaba denegación (Access Denied) para 'validate', pero pasó sin error.")
	}
}

func TestGuardCoreWriteAccess_NonCoreAllowed(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Cualquier comando intentando escribir fuera del core
	os.Args = []string{"qdd", "audit"}

	targetPath := filepath.Join(".qdd", "project", "findings", "finding_test.yaml")
	
	err := GuardCoreWriteAccess(targetPath)
	if err != nil {
		t.Errorf("Se esperaba autorización para escribir fuera del core, pero dio error: %v", err)
	}
}
