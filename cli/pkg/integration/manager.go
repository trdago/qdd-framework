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
			&OpenCodeAdapter{},
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
			fmt.Printf("[!] No se pudo configurar %s: %v\n", adapter.Name(), err)
			continue // Proceed with others even if one fails
		}
		fmt.Printf("    [✓] %s\n", adapter.Name())
	}
	imprimirMCP(ConectarMCP(projectPath))
	return nil
}

// imprimirMCP dice qué archivos recibieron el servidor qdd (FND-308) y qué falta para usarlo.
func imprimirMCP(cambios []string, err error) {
	if err != nil {
		fmt.Printf("[!] Servidor MCP de QDD: %v\n", err)
	}
	for _, c := range cambios {
		fmt.Printf("    [+] Servidor MCP de QDD en %s\n", c)
	}
	if len(cambios) > 0 {
		fmt.Println("    [i] Claude Code pide aprobar el servidor qdd de .mcp.json la primera vez que abre el proyecto.")
	}
}
