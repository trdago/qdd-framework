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
