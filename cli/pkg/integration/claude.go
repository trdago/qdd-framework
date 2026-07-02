package integration

import (
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeAdapter implements integration for Claude Code.
type ClaudeAdapter struct{}

func (c *ClaudeAdapter) Name() string {
	return "Claude Code"
}

func (c *ClaudeAdapter) Sync(projectPath string) error {
	rulesPath := filepath.Join(projectPath, ".clauderc")
	
	err := SafeInjectIdempotent(rulesPath)
	if err != nil {
		return fmt.Errorf("Claude adapter failed to sync %s: %w", rulesPath, err)
	}

	// Sync the actual Claude Command workflow
	commandsDir := filepath.Join(projectPath, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create claude commands directory: %w", err)
	}

	qddCommandPath := filepath.Join(commandsDir, "qdd.md")
	
	// Create with frontmatter if it doesn't exist
	if _, err := os.Stat(qddCommandPath); os.IsNotExist(err) {
		frontmatter := "---\ndescription: QDD Framework native AI commands\n---\n\n"
		os.WriteFile(qddCommandPath, []byte(frontmatter), 0644)
	}

	err = SafeInjectIdempotent(qddCommandPath)
	if err != nil {
		return fmt.Errorf("Claude adapter failed to sync %s: %w", qddCommandPath, err)
	}

	return nil
}
