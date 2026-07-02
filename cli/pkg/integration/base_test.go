package integration

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeInjectIdempotent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "qdd-inject-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, ".cursorrules")

	// 1. Inject into new file
	err = SafeInjectIdempotent(testFile)
	if err != nil {
		t.Fatalf("failed to inject into new file: %v", err)
	}

	content1, _ := ioutil.ReadFile(testFile)
	if !strings.Contains(string(content1), markerBegin) {
		t.Errorf("expected markerBegin in content")
	}

	// 2. Inject again (Idempotency)
	err = SafeInjectIdempotent(testFile)
	if err != nil {
		t.Fatalf("failed second injection: %v", err)
	}

	content2, _ := ioutil.ReadFile(testFile)
	// Size should be roughly the same, block should not be duplicated
	if len(content1) != len(content2) {
		t.Errorf("content size changed after second injection (expected %d, got %d)", len(content1), len(content2))
	}

	// 3. Inject with existing user content
	userContent := "User custom rule 1\n" + string(content2) + "\nUser custom rule 2"
	ioutil.WriteFile(testFile, []byte(userContent), 0644)

	err = SafeInjectIdempotent(testFile)
	if err != nil {
		t.Fatalf("failed third injection: %v", err)
	}

	content3, _ := ioutil.ReadFile(testFile)
	if !strings.Contains(string(content3), "User custom rule 1") || !strings.Contains(string(content3), "User custom rule 2") {
		t.Errorf("injection destroyed user content")
	}
	if strings.Count(string(content3), markerBegin) != 1 {
		t.Errorf("expected exactly one markerBegin in content, got %d", strings.Count(string(content3), markerBegin))
	}
}
