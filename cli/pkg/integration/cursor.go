package integration

import (
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
	if _, err := os.Stat(mcpPath); os.IsNotExist(err) {
		mcpContent := `{
  "mcpServers": {
    "qdd": {
      "command": "qdd",
      "args": ["mcp-server"]
    }
  }
}`
		os.WriteFile(mcpPath, []byte(mcpContent), 0644)
	}

	return nil
}
