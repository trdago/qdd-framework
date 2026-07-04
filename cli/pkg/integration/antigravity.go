package integration

import (
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

	// Antigravity slash command workflow (/qdd)
	workflowsDir := filepath.Join(projectPath, ".agents", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	qddWorkflowPath := filepath.Join(workflowsDir, "qdd.md")
	
	// Ensure frontmatter exists
	frontmatter := "---\ndescription: QDD Framework native AI commands\n---\n\n"
	if _, err := os.Stat(qddWorkflowPath); os.IsNotExist(err) {
		os.WriteFile(qddWorkflowPath, []byte(frontmatter), 0644)
	}
	
	if _, err := os.Stat(qddWorkflowPath); err == nil {
		content, _ := os.ReadFile(qddWorkflowPath)
		contentStr := string(content)
		if !strings.Contains(contentStr, "description:") {
			os.WriteFile(qddWorkflowPath, []byte(frontmatter+contentStr), 0644)
		}
	}

	err = SafeInjectIdempotent(qddWorkflowPath)
	if err != nil {
		return fmt.Errorf("Antigravity adapter failed to sync %s: %w", qddWorkflowPath, err)
	}

	return nil
}
