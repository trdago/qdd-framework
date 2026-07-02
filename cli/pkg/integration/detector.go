package integration

import (
	"os"
	"path/filepath"
)

// IsQDDProject checks if the given directory contains QDD configuration.
// It looks for .qdd directory or common config files.
func IsQDDProject(projectPath string) bool {
	markers := []string{
		".qdd",
		"qdd.yaml",
		"qdd.yml",
		"qdd.json",
		"QDD.md",
	}

	for _, marker := range markers {
		fullPath := filepath.Join(projectPath, marker)
		if _, err := os.Stat(fullPath); err == nil {
			return true
		}
	}

	return false
}
