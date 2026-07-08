package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AntigravityAdapter implements integration for the Antigravity IDE.
type AntigravityAdapter struct{}

func (a *AntigravityAdapter) Name() string {
	return "Antigravity"
}

func (a *AntigravityAdapter) Sync(projectPath string) error {
	rulesPath := filepath.Join(projectPath, ".antigravityrules")
	
	err := SafeInjectIdempotent(rulesPath)
	if err != nil {
		return fmt.Errorf("Antigravity adapter failed to sync %s: %w", rulesPath, err)
	}

	if err := syncAntigravityWorkflows(projectPath); err != nil {
		return err
	}

	return syncAntigravityMCP(projectPath)
}

func syncAntigravityWorkflows(projectPath string) error {
	workflowsDir := filepath.Join(projectPath, ".agents", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	qddWorkflowPath := filepath.Join(workflowsDir, "qdd.md")
	
	ensureWorkflowFrontmatter(qddWorkflowPath)

	err := SafeInjectIdempotent(qddWorkflowPath)
	if err != nil {
		return fmt.Errorf("Antigravity adapter failed to sync %s: %w", qddWorkflowPath, err)
	}

	return nil
}

func ensureWorkflowFrontmatter(qddWorkflowPath string) {
	frontmatter := "---\ndescription: QDD Framework native AI commands\n---\n\n"
	if _, err := os.Stat(qddWorkflowPath); os.IsNotExist(err) {
		os.WriteFile(qddWorkflowPath, []byte(frontmatter), 0644)
		return
	}
	
	if _, err := os.Stat(qddWorkflowPath); err == nil {
		content, _ := os.ReadFile(qddWorkflowPath)
		contentStr := string(content)
		if !strings.Contains(contentStr, "description:") {
			os.WriteFile(qddWorkflowPath, []byte(frontmatter+contentStr), 0644)
		}
	}
}

func syncAntigravityMCP(projectPath string) error {
	cursorDir := filepath.Join(projectPath, ".cursor")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		return err
	}
	
	mcpPath := filepath.Join(cursorDir, "mcp.json")
	qddCmd := resolveQDDPath()
	
	mcpData := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"qdd": map[string]interface{}{
				"command": qddCmd,
				"args":    []string{"mcp-server"},
			},
		},
	}

	mcpData = mergeAntigravityMCPData(mcpPath, mcpData)

	finalJSON, err := json.MarshalIndent(mcpData, "", "  ")
	if err == nil {
		os.WriteFile(mcpPath, finalJSON, 0644)
	}

	return nil
}

func mergeAntigravityMCPData(mcpPath string, mcpData map[string]interface{}) map[string]interface{} {
	if _, err := os.Stat(mcpPath); err != nil {
		return mcpData
	}

	existingData, readErr := os.ReadFile(mcpPath)
	if readErr != nil {
		return mcpData
	}

	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err == nil {
		if servers, ok := existing["mcpServers"].(map[string]interface{}); ok {
			servers["qdd"] = mcpData["mcpServers"].(map[string]interface{})["qdd"]
			return existing
		}
	}

	return mcpData
}
