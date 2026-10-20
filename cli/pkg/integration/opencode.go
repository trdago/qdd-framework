package integration

import (
	"fmt"
	"os"
	"path/filepath"
)

// OpenCodeAdapter implements integration for OpenCode, Aider, Cline, and generic AI agents.
// Strictly follows Zero-Else and Early Return.
type OpenCodeAdapter struct{}

func (o *OpenCodeAdapter) Name() string {
	return "OpenCode & Generic Agents (AGENTS.md)"
}

func (o *OpenCodeAdapter) Sync(projectPath string) error {
	// 1. Sync root AGENTS.md (Universal Standard)
	agentsMDPath := filepath.Join(projectPath, "AGENTS.md")
	err := SafeInjectIdempotent(agentsMDPath)
	if err != nil {
		return fmt.Errorf("OpenCode adapter failed to sync %s: %w", agentsMDPath, err)
	}

	// 2. Sync .agentrules
	agentRulesPath := filepath.Join(projectPath, ".agentrules")
	err = SafeInjectIdempotent(agentRulesPath)
	if err != nil {
		return fmt.Errorf("OpenCode adapter failed to sync %s: %w", agentRulesPath, err)
	}

	// 3. Sync .opencode/ configuration directory
	opencodeDir := filepath.Join(projectPath, ".opencode")
	if errMkdir := os.MkdirAll(opencodeDir, 0755); errMkdir != nil {
		return fmt.Errorf("failed to create .opencode directory: %w", errMkdir)
	}

	opencodeRules := filepath.Join(opencodeDir, "qdd.md")
	if _, errStat := os.Stat(opencodeRules); os.IsNotExist(errStat) {
		frontmatter := "---\ndescription: QDD Framework Universal Harness for OpenCode\n---\n\n"
		_ = os.WriteFile(opencodeRules, []byte(frontmatter), 0644)
	}

	err = SafeInjectIdempotent(opencodeRules)
	if err != nil {
		return fmt.Errorf("OpenCode adapter failed to sync %s: %w", opencodeRules, err)
	}

	return nil
}
