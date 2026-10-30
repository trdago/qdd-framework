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

// Canal preview (pedido del usuario, 2026-10-30): un tag vX.Y.Z-rc.N sale en npm con --tag preview y como pre-release
// en GitHub; latest es solo para versiones estables (VER-05 exige su uso medido).
func TestPublishYml_UnaCandidataSaleComoPreview(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "publish.yml"))
	if err != nil {
		t.Fatal(err)
	}
	code := string(content)
	for _, s := range []string{`*-*) CANAL=preview`, `npm publish --access public --tag "$CANAL"`, "prerelease: ${{ contains(github.ref_name, '-') }}"} {
		if !strings.Contains(code, s) {
			t.Fatalf("publish.yml no tiene %q", s)
		}
	}
	if strings.Count(code, "prerelease: ${{ contains(github.ref_name, '-') }}") != 2 {
		t.Fatal("las dos subidas a la GitHub Release la marcan como pre-release")
	}
}
