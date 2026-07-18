package bugreport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFile_CreatesOpenFindingAndEvidence(t *testing.T) {
	cwd := t.TempDir()

	filed, err := File(cwd, Report{
		Command:  "node",
		Args:     []string{"server.js"},
		ExitCode: 1,
		Output:   "TypeError: cannot read property 'x' of undefined",
	})
	if err != nil {
		t.Fatalf("File returned unexpected error: %v", err)
	}

	if filed.Finding.Status != "OPEN" {
		t.Errorf("Status = %q, want OPEN", filed.Finding.Status)
	}
	if filed.Finding.ID != "FND-001" {
		t.Errorf("ID = %q, want FND-001 for the first finding in an empty project", filed.Finding.ID)
	}

	assertFileContains(t, filed.FindingPath, "status: OPEN")
	assertFileContains(t, filed.EvidencePath, "TypeError: cannot read property")
}

func TestFile_TestPendingIsFlaggedInMetadata(t *testing.T) {
	cwd := t.TempDir()

	filed, err := File(cwd, Report{Command: "myapp", ExitCode: 2, Output: "panic"})
	if err != nil {
		t.Fatalf("File returned unexpected error: %v", err)
	}

	pending, ok := filed.Finding.Metadata["test_pending"].(bool)
	if !ok || !pending {
		t.Errorf("expected metadata.test_pending=true so a human/AI knows a regression test is still owed, got: %v", filed.Finding.Metadata["test_pending"])
	}
}

func seedExistingFindings(t *testing.T, findingsDir string, names []string) {
	t.Helper()
	if err := os.MkdirAll(findingsDir, 0755); err != nil {
		t.Fatalf("failed to create findings dir: %v", err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(findingsDir, name), []byte("id: x"), 0644); err != nil {
			t.Fatalf("failed to seed %q: %v", name, err)
		}
	}
}

func TestFile_NextIDContinuesExistingSequence(t *testing.T) {
	cwd := t.TempDir()
	findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
	seedExistingFindings(t, findingsDir, []string{"FND-001.yaml", "FND-002-SOME-SLUG.yaml", "FND-017-OTHER.yaml"})

	filed, err := File(cwd, Report{Command: "svc", ExitCode: 1, Output: "boom"})
	if err != nil {
		t.Fatalf("File returned unexpected error: %v", err)
	}

	if filed.Finding.ID != "FND-018" {
		t.Errorf("ID = %q, want FND-018 (continuing after the highest existing FND-017)", filed.Finding.ID)
	}
}

func TestFile_WrittenYAMLIsValid(t *testing.T) {
	cwd := t.TempDir()

	filed, err := File(cwd, Report{Command: "svc", Args: []string{"--flag"}, ExitCode: 3, Output: "trace"})
	if err != nil {
		t.Fatalf("File returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(filed.FindingPath)
	if err != nil {
		t.Fatalf("failed to read finding file: %v", err)
	}
	var parsed Finding
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("finding YAML is not valid: %v", err)
	}
	if parsed.ID != filed.Finding.ID {
		t.Errorf("parsed ID = %q, want %q", parsed.ID, filed.Finding.ID)
	}
}

func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %q: %v", path, err)
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("expected %q to contain %q, got:\n%s", path, substr, data)
	}
}
