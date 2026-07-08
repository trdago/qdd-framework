package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CursorAdapter implements integration for the Cursor IDE AI.
type CursorAdapter struct{}

func (c *CursorAdapter) Name() string {
	return "Cursor"
}

func (c *CursorAdapter) Sync(projectPath string) error {
	rulesPath := filepath.Join(projectPath, ".cursorrules")
	
	err := SafeInjectIdempotent(rulesPath)
	if err != nil {
		return fmt.Errorf("Cursor adapter failed to sync %s: %w", rulesPath, err)
	}

	cursorDir := filepath.Join(projectPath, ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return err
	}
	
	mcpPath := filepath.Join(cursorDir, "mcp.json")
	qddCmd := resolveQDDPath()
	
	// Define the base structure
	mcpData := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"qdd": map[string]interface{}{
				"command": qddCmd,
				"args":    []string{"mcp-server"},
			},
		},
	}

	if _, err := os.Stat(mcpPath); err == nil {
		// Try to read and merge if it already exists
		existingData, readErr := os.ReadFile(mcpPath)
		if readErr == nil {
			var existing map[string]interface{}
			if err := json.Unmarshal(existingData, &existing); err == nil {
				if servers, ok := existing["mcpServers"].(map[string]interface{}); ok {
					servers["qdd"] = mcpData["mcpServers"].(map[string]interface{})["qdd"]
					mcpData = existing
				}
			}
		}
	}

	finalJSON, err := json.MarshalIndent(mcpData, "", "  ")
	if err == nil {
		os.WriteFile(mcpPath, finalJSON, 0644)
	}

	return nil
}
