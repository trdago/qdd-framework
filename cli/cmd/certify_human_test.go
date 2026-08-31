package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHumanTarget(t *testing.T) {
	if res := resolveHumanTarget([]string{}); res != "all" {
		t.Fatalf("Expected 'all' for empty args, got: %s", res)
	}

	if res := resolveHumanTarget([]string{"usability"}); res != "usability" {
		t.Fatalf("Expected 'usability', got: %s", res)
	}

	if res := resolveHumanTarget([]string{""}); res != "all" {
		t.Fatalf("Expected 'all' for empty string, got: %s", res)
	}
}

func TestFindHumanCertifierRunner(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd_test_runner_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Should fail when no runner is found
	_, err = findHumanCertifierRunner(tempDir)
	if err == nil {
		t.Fatalf("Expected error when no runner exists, but got nil")
	}

	// 2. Should find in .qdd/core/plugins/qdd-human-certifier/runtime/index.mjs
	pluginRuntimeDir := filepath.Join(tempDir, ".qdd", "core", "plugins", "qdd-human-certifier", "runtime")
	if err := os.MkdirAll(pluginRuntimeDir, 0755); err != nil {
		t.Fatalf("Failed to create mock runtime dir: %v", err)
	}
	indexFile := filepath.Join(pluginRuntimeDir, "index.mjs")
	if err := os.WriteFile(indexFile, []byte("// mock runner"), 0644); err != nil {
		t.Fatalf("Failed to write mock index.mjs: %v", err)
	}

	foundPath, err := findHumanCertifierRunner(tempDir)
	if err != nil {
		t.Fatalf("Expected to find runner, got error: %v", err)
	}
	if foundPath != indexFile {
		t.Fatalf("Expected path %s, got: %s", indexFile, foundPath)
	}
}
