package integration

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// MockAdapter is used to test the manager's error handling
type MockAdapter struct {
	ShouldFail bool
}

func (m *MockAdapter) Name() string { return "Mock" }
func (m *MockAdapter) Sync(projectPath string) error {
	if m.ShouldFail {
		return errors.New("mock error")
	}
	return nil
}

func TestSyncAll_NotQDD(t *testing.T) {
	tempDir := t.TempDir()
	
	manager := NewIntegrationManager()
	err := manager.SyncAll(tempDir)
	if err == nil {
		t.Errorf("Expected error for non-QDD project, got nil")
	}
}

func TestSyncAll_SuccessAndFailures(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, ".qdd"), 0755) // Make it a QDD project
	
	manager := &IntegrationManager{
		adapters: []AIAdapter{
			&MockAdapter{ShouldFail: false},
			&MockAdapter{ShouldFail: true}, // Should not panic and should continue
			&MockAdapter{ShouldFail: false},
		},
	}
	
	err := manager.SyncAll(tempDir)
	if err != nil {
		t.Errorf("Unexpected error from SyncAll: %v", err)
	}
}
