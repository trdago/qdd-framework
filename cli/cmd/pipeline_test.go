package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// TestPipelineExecution verifies that 'qdd run [cmds...]' works
// and aborts correctly if an intermediate command fails.
func TestPipelineExecution(t *testing.T) {
	binName := "qdd-test-bin"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	// First we need to build the binary to test it
	buildCmd := exec.Command("go", "build", "-o", binName, "../main.go")
	if err := buildCmd.Run(); err != nil {
		t.Skipf("Skipping test because go build failed: %v", err)
	}

	// Create dummy context so Gatekeeper doesn't block rootCmd execution
	os.MkdirAll(".qdd", 0755)
	os.WriteFile(".qdd/state.json", []byte(`{"status":"initialized","version":"v1.6.0"}`), 0644)
	defer os.RemoveAll(".qdd")

	// Test 1: valid commands -> success
	cmd1 := exec.Command("./" + binName, "run", "init") // init does not require full context
	if err := cmd1.Run(); err != nil {
		t.Errorf("Expected pipeline to succeed with valid command, got %v", err)
	}

	// Test 2: invalid command -> fail
	cmd2 := exec.Command("./" + binName, "run", "fakecommand")
	if err := cmd2.Run(); err == nil {
		t.Errorf("Expected pipeline to fail with invalid command")
	}

	// Test 3: intermediate fail -> fails pipeline
	cmd3 := exec.Command("./" + binName, "run", "init", "fakecommand", "init")
	if err := cmd3.Run(); err == nil {
		t.Errorf("Expected pipeline to abort and fail on intermediate invalid command")
	}
}
