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
	
	mcpData := buildInitialMCPData(qddCmd, projectPath)
	mcpData = mergeExistingMCPData(mcpPath, mcpData)

	finalJSON, err := json.MarshalIndent(mcpData, "", "  ")
	if err == nil {
		os.WriteFile(mcpPath, finalJSON, 0644)
	}

	return nil
}

func buildInitialMCPData(qddCmd, projectPath string) map[string]interface{} {
	serverData := map[string]interface{}{
		"command": qddCmd,
		"args":    []string{"mcp-server"},
	}

	envVars := make(map[string]string)
	
	localPaths := []string{
		filepath.Join(projectPath, "venv", "bin"),
		filepath.Join(projectPath, ".venv", "bin"),
		filepath.Join(projectPath, "myenv", "bin"),
		filepath.Join(projectPath, "node_modules", ".bin"),
	}

	for _, p := range localPaths {
		if _, err := os.Stat(p); err == nil {
			appendToPath(envVars, p)
		}
	}

	discoverGlobalShims(envVars)

	if len(envVars) > 0 {
		serverData["env"] = envVars
	}

	return map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"qdd": serverData,
		},
	}
}

func discoverGlobalShims(envVars map[string]string) {
	home, _ := os.UserHomeDir()
	if home == "" {
		return
	}

	pyenvShims := filepath.Join(home, ".pyenv", "shims")
	if _, err := os.Stat(pyenvShims); err == nil {
		appendToPath(envVars, pyenvShims)
	}
	
	nvmPath := filepath.Join(home, ".nvm", "versions", "node")
	if entries, err := os.ReadDir(nvmPath); err == nil && len(entries) > 0 {
		appendToPath(envVars, filepath.Join(nvmPath, entries[0].Name(), "bin"))
	}
}

func appendToPath(envVars map[string]string, newPath string) {
	current := envVars["PATH"]
	if current == "" {
		current = os.Getenv("PATH")
	}
	envVars["PATH"] = newPath + string(os.PathListSeparator) + current
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
