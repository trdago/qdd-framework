package integration

import (
	"os"
	"path/filepath"
)

// FindProjectRoot walks upward from startPath looking for the canonical
// project root, so callers resolving from os.Getwd() operate on the real
// project root instead of whatever subdirectory the command happened to be
// invoked from. The walk stops at the first directory that is either a QDD
// project (nearest .qdd wins, so intentionally nested projects are respected)
// or a git root (.git directory, or .git file for worktrees/submodules). It
// never crosses a repository boundary: a stray .qdd in a folder ABOVE the repo
// (e.g. a workspace folder holding many repos) must not hijack every sibling
// repo's commands. Falls back to startPath if neither is found.
func FindProjectRoot(startPath string) string {
	if root, found := walkUpFor(startPath, isProjectBoundary); found {
		return root
	}
	return startPath
}

func isProjectBoundary(dir string) bool {
	return IsQDDProject(dir) || isGitRoot(dir)
}

// isGitRoot accepts a .git directory (regular clone) or a .git file
// (git worktree or submodule).
func isGitRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
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
