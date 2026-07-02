package integration

import (
	"fmt"
)

// IntegrationManager handles the detection and synchronization of QDD
// commands across all supported AI platforms.
type IntegrationManager struct {
	adapters []AIAdapter
}

// NewIntegrationManager initializes the manager with all registered adapters.
func NewIntegrationManager() *IntegrationManager {
	return &IntegrationManager{
		adapters: []AIAdapter{
			&CursorAdapter{},
			&ClaudeAdapter{},
			&AntigravityAdapter{},
		},
	}
}

// SyncAll detects if the directory is a QDD project and syncs configurations.
func (m *IntegrationManager) SyncAll(projectPath string) error {
	if !IsQDDProject(projectPath) {
		return fmt.Errorf("not a QDD project: %s", projectPath)
	}

	for _, adapter := range m.adapters {
		err := adapter.Sync(projectPath)
		if err != nil {
			fmt.Printf("[QDD] Error syncing %s adapter: %v\n", adapter.Name(), err)
			continue // Proceed with others even if one fails
		}
		fmt.Printf("[QDD] Successfully synchronized %s integration.\n", adapter.Name())
	}

	return nil
}
