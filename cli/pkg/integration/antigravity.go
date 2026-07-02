package integration

import (
	"fmt"
	"path/filepath"
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

	return nil
}
