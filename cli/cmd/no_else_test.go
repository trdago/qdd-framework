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

func checkSkipDir(name string) error {
	if name == "node_modules" || name == "dist" || name == ".git" || name == ".qdd" || name == "venv" || name == ".venv" || name == "myenv" {
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
	return strings.HasSuffix(name, "_test.go") || name == "scratch.vue"
}

func hasValidExtension(name string) bool {
	return strings.HasSuffix(name, ".go") || 
		strings.HasSuffix(name, ".js") || 
		strings.HasSuffix(name, ".ts") || 
		strings.HasSuffix(name, ".vue")
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
	
	if strings.Contains(code, target1) || strings.Contains(code, target2) || strings.Contains(code, target3) {
		t.Errorf("🚨 Regla violada (CLEAN-01): Se detectó un 'else' en el archivo %s. Debes refactorizar para usar Early Returns o v-if negado.", path)
	}
	return nil
}
