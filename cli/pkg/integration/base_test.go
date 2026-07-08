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

	content1 := testInjectionNewFile(t, testFile)
	testInjectionIdempotency(t, testFile, content1)
	testInjectionWithUserContent(t, testFile, content1)
}

func testInjectionNewFile(t *testing.T, testFile string) []byte {
	err := SafeInjectIdempotent(testFile)
	if err != nil {
		t.Fatalf("failed to inject into new file: %v", err)
	}

	content1, _ := ioutil.ReadFile(testFile)
	if !strings.Contains(string(content1), markerBegin) {
		t.Errorf("expected markerBegin in content")
	}
	return content1
}

func testInjectionIdempotency(t *testing.T, testFile string, content1 []byte) {
	err := SafeInjectIdempotent(testFile)
	if err != nil {
		t.Fatalf("failed second injection: %v", err)
	}

	content2, _ := ioutil.ReadFile(testFile)
	if len(content1) != len(content2) {
		t.Errorf("content size changed after second injection (expected %d, got %d)", len(content1), len(content2))
	}
}

func testInjectionWithUserContent(t *testing.T, testFile string, content1 []byte) {
	userContent := "User custom rule 1\n" + string(content1) + "\nUser custom rule 2"
	ioutil.WriteFile(testFile, []byte(userContent), 0644)

	err := SafeInjectIdempotent(testFile)
	if err != nil {
		t.Fatalf("failed third injection: %v", err)
	}

	content3, _ := ioutil.ReadFile(testFile)
	strContent := string(content3)
	if !strings.Contains(strContent, "User custom rule 1") || !strings.Contains(strContent, "User custom rule 2") {
		t.Errorf("injection destroyed user content")
	}
	if strings.Count(strContent, markerBegin) != 1 {
		t.Errorf("expected exactly one markerBegin in content, got %d", strings.Count(strContent, markerBegin))
	}
}
