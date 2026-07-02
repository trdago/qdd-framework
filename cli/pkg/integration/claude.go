package integration

import (
	"fmt"
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

	return nil
}
