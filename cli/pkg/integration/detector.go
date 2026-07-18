package integration

import (
	"os"
	"path/filepath"
)

// FindProjectRoot walks upward from startPath looking for the canonical
// project root, so callers resolving from os.Getwd() operate on the real
// project root instead of whatever subdirectory the command happened to be
// invoked from. It prefers the nearest existing QDD project (a directory
// where IsQDDProject is already true) so intentionally-scoped nested
// projects (e.g. an isolated .qdd sandbox created for testing) are respected
// rather than overridden. Only when no .qdd is found anywhere upward does it
// fall back to the nearest .git directory, so a first-time `qdd init` run
// from a subdirectory still targets the real repo root. Falls back to
// startPath if neither is found (standalone/new project, or filesystem root).
func FindProjectRoot(startPath string) string {
	if root, found := walkUpFor(startPath, IsQDDProject); found {
		return root
	}
	if root, found := walkUpFor(startPath, isGitRoot); found {
		return root
	}
	return startPath
}

func isGitRoot(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

func walkUpFor(startPath string, matches func(string) bool) (string, bool) {
	dir := startPath
	for {
		if matches(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

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
