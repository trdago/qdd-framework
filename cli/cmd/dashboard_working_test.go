package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestWorkingSemaphore(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "qdd-test-working-*")
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	os.MkdirAll(".qdd", 0755)

	dummyCmd := &cobra.Command{
		Use: "learn",
	}

	statePath := filepath.Join(tempDir, ".qdd", "state.json")
	os.WriteFile(statePath, []byte(`{"version":"v1.1.0"}`), 0644)
	
	dummyRoot := &cobra.Command{Version: "v1.1.0"}
	dummyRoot.AddCommand(dummyCmd)

	err := rootCmd.PersistentPreRunE(dummyCmd, []string{})
	if err != nil {
		t.Fatalf("Error inesperado en PreRun: %v", err)
	}

	workingPath := filepath.Join(tempDir, ".qdd", "working")
	content, err := os.ReadFile(workingPath)
	if err != nil {
		t.Fatalf("No se creó el archivo .qdd/working: %v", err)
	}

	if string(content) != "learn" {
		t.Errorf("Esperaba 'learn' en el archivo working, obtuve '%s'", string(content))
	}

	rootCmd.PersistentPostRun(dummyCmd, []string{})

	if _, err := os.Stat(workingPath); !os.IsNotExist(err) {
		t.Errorf("El archivo .qdd/working no se eliminó después del PostRun")
	}
}

func TestDashboardLifecycleInjection(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "qdd-test-lifecycle-*")
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	os.MkdirAll(".qdd", 0755)

	state := buildState()

	found := false
	for _, doc := range state.Knowledge {
		if doc.Path == "docs/command-reference.md" {
			found = true
			if doc.Content == "" {
				t.Errorf("El contenido del documento virtual está vacío")
			}
			break
		}
	}

	if !found {
		t.Errorf("No se inyectó el documento virtual 'docs/command-reference.md' para el Lifecycle")
	}
}

func TestDashboardProjectName(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "qdd-test-project-name-*")
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)

	os.MkdirAll(".qdd", 0755)

	state := buildState()

	expectedName := filepath.Base(tempDir)
	if state.ProjectName != expectedName {
		t.Errorf("Expected ProjectName to be %s, got %s", expectedName, state.ProjectName)
	}
}
