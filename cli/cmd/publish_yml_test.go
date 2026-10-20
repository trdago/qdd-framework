package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishYmlNoElse(t *testing.T) {
	projectRoot := "../.." // Assumes test is run from cli/cmd
	publishYmlPath := filepath.Join(projectRoot, ".github", "workflows", "publish.yml")
	
	content, err := os.ReadFile(publishYmlPath)
	if err != nil {
		t.Fatalf("Failed to read publish.yml: %v", err)
	}

	code := string(content)
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "else" {
			t.Errorf("🚨 Regresión en publish.yml (Línea %d): Se detectó un 'else'. Usa 'exit 0' o guarda explícita.", i+1)
		}
	}
}
