package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoElseInCode scans all .go files in the cmd package
// to ensure that the global rule is strictly followed.
func TestNoElseInCode(t *testing.T) {
	projectRoot := "../.." // Assumes test is run from cli/cmd
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return checkSkipDir(info.Name())
		}

		if !shouldCheckFileForElse(info.Name()) {
			return nil
		}

		return checkFileContentForElse(t, path)
	})
	
	if err != nil {
		t.Fatalf("Error scanning files: %v", err)
	}
}

var skippedDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	".git":         true,
	".qdd":         true,
	"venv":         true,
	".venv":        true,
	"myenv":        true,
	"core_assets":  true,
	".github":      true,
}

func checkSkipDir(name string) error {
	if skippedDirs[name] {
		return filepath.SkipDir
	}
	return nil
}

func shouldCheckFileForElse(name string) bool {
	if isExcludedFile(name) {
		return false
	}
	return hasValidExtension(name)
}

func isExcludedFile(name string) bool {
	return strings.HasSuffix(name, "_test.go") || name == "scratch.vue" || name == "publish.yml"
}

func hasValidExtension(name string) bool {
	return strings.HasSuffix(name, ".go") || 
		strings.HasSuffix(name, ".js") || 
		strings.HasSuffix(name, ".ts") || 
		strings.HasSuffix(name, ".vue") ||
		strings.HasSuffix(name, ".sh") ||
		strings.HasSuffix(name, ".yml") ||
		strings.HasSuffix(name, ".yaml") ||
		strings.HasSuffix(name, ".nsh") ||
		strings.HasSuffix(name, ".nsi")
}

func checkFileContentForElse(t *testing.T, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	code := string(content)
	target1 := "} el" + "se {"
	target2 := " el" + "se "
	target3 := "v-el" + "se"
	target4 := "!el" + "se"
	target5 := "${el" + "se}"
	
	if strings.Contains(code, target1) || strings.Contains(code, target2) || strings.Contains(code, target3) || strings.Contains(code, target4) || strings.Contains(code, target5) {
		t.Errorf("🚨 Regla violada (CLEAN-01): Se detectó un 'else' en el archivo %s. Debes refactorizar para usar Early Returns o v-if negado.", path)
		return nil
	}

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "el" + "se" || trimmed == "} el" + "se" || trimmed == "el" + "se {" || strings.HasPrefix(trimmed, "el" + "se ") || trimmed == target4 || trimmed == target5 {
			if !strings.Contains(line, "\"") && !strings.Contains(line, "'") && !strings.Contains(line, "`") {
				t.Errorf("🚨 Regla violada (CLEAN-01): Se detectó un 'else' en el archivo %s. Debes refactorizar para usar Early Returns o v-if negado.", path)
				return nil
			}
		}
	}
	return nil
}
