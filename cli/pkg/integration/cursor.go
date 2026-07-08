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
	
	mcpData := buildInitialMCPData(qddCmd)
	mcpData = mergeExistingMCPData(mcpPath, mcpData)

	finalJSON, err := json.MarshalIndent(mcpData, "", "  ")
	if err == nil {
		os.WriteFile(mcpPath, finalJSON, 0644)
	}

	return nil
}

func buildInitialMCPData(qddCmd string) map[string]interface{} {
	return map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"qdd": map[string]interface{}{
				"command": qddCmd,
				"args":    []string{"mcp-server"},
			},
		},
	}
}

func mergeExistingMCPData(mcpPath string, mcpData map[string]interface{}) map[string]interface{} {
	if _, err := os.Stat(mcpPath); err != nil {
		return mcpData
	}

	existingData, readErr := os.ReadFile(mcpPath)
	if readErr != nil {
		return mcpData
	}

	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return mcpData
	}

	return performDeepMerge(existing, mcpData)
}

func performDeepMerge(existing, mcpData map[string]interface{}) map[string]interface{} {
	if _, ok := existing["mcpServers"]; !ok {
		existing["mcpServers"] = map[string]interface{}{}
	}

	if servers, ok := existing["mcpServers"].(map[string]interface{}); ok {
		qddServer := mcpData["mcpServers"].(map[string]interface{})["qdd"]
		servers["qdd"] = qddServer
		existing["mcpServers"] = servers
	}
	
	return existing
}
