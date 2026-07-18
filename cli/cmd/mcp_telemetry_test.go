package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestHandleLearnTool(t *testing.T) {
	tempDir, originalWD := setupTestHandleLearnToolEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer os.Chdir(originalWD)

	setupTestDocs(t, tempDir)

	result, err := handleLearnTool(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handleLearnTool returned unexpected error: %v", err)
	}

	validateHandleLearnToolResult(t, result)
	validateKnowledgeIndexCreated(t, tempDir)
}

func setupTestHandleLearnToolEnvironment(t *testing.T) (string, string) {
	tempDir, err := os.MkdirTemp("", "qdd-learn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}

	return tempDir, originalWD
}

func setupTestDocs(t *testing.T, tempDir string) {
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}
	
	dummyContent := "# Arquitectura de Prueba\n\nEste es un documento de prueba para validar que el contexto se lee correctamente."
	if err := os.WriteFile(filepath.Join(docsDir, "arquitectura.md"), []byte(dummyContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy markdown file: %v", err)
	}
}

func validateHandleLearnToolResult(t *testing.T, result *mcp.CallToolResult) {
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("Expected non-empty CallToolResult")
	}

	textContent := extractTextContent(t, result.Content[0])

	if !strings.Contains(textContent.Text, "(MAP-REDUCE COGNITIVO)") {
		t.Errorf("Result is missing Map-Reduce instructions. Got: %s", textContent.Text)
	}

	if strings.Contains(textContent.Text, "Arquitectura de Prueba") {
		t.Errorf("Result should NOT contain the file content directly. Got: %s", textContent.Text)
	}
}

func extractTextContent(t *testing.T, content interface{}) mcp.TextContent {
	textContent, ok := content.(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected first content element to be mcp.TextContent")
	}
	return textContent
}

func validateKnowledgeIndexCreated(t *testing.T, tempDir string) {
	indexPath := filepath.Join(tempDir, ".qdd", "knowledge_index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("Expected knowledge_index.json to be created at %s", indexPath)
	}
}

// FND: a naive re-run of qdd_learn overwrote .qdd/understanding.json from
// scratch every time, discarding anything previously learned. It must instead
// read the existing understanding first and instruct the AI to refine it.
func TestHandleLearnTool_FirstRun_HasNoPriorUnderstanding(t *testing.T) {
	tempDir, originalWD := setupTestHandleLearnToolEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer os.Chdir(originalWD)
	setupTestDocs(t, tempDir)

	result, err := handleLearnTool(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handleLearnTool returned unexpected error: %v", err)
	}

	text := extractTextContent(t, result.Content[0]).Text
	if !strings.Contains(text, "primera vez que se ejecuta") {
		t.Errorf("expected first-run instructions when no prior understanding.json exists, got: %s", text)
	}
}

func writePriorUnderstanding(t *testing.T, tempDir, content string) {
	t.Helper()
	qddDir := filepath.Join(tempDir, ".qdd")
	if err := os.MkdirAll(qddDir, 0755); err != nil {
		t.Fatalf("failed to create .qdd dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(qddDir, "understanding.json"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prior understanding.json: %v", err)
	}
}

func TestHandleLearnTool_ReRun_RefinesExistingUnderstanding(t *testing.T) {
	tempDir, originalWD := setupTestHandleLearnToolEnvironment(t)
	defer os.RemoveAll(tempDir)
	defer os.Chdir(originalWD)
	setupTestDocs(t, tempDir)

	priorContent := `{"summary": "Entendimiento previo del proyecto"}`
	writePriorUnderstanding(t, tempDir, priorContent)

	result, err := handleLearnTool(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handleLearnTool returned unexpected error: %v", err)
	}

	text := extractTextContent(t, result.Content[0]).Text
	if !strings.Contains(text, "REFÍNALO") {
		t.Errorf("expected refine instructions when understanding.json already exists, got: %s", text)
	}
	if !strings.Contains(text, priorContent) {
		t.Errorf("expected prior understanding.json content to be embedded in the instructions, got: %s", text)
	}
}

func TestLoadExistingUnderstanding_MissingFile(t *testing.T) {
	cwd := t.TempDir()

	content, found := loadExistingUnderstanding(cwd)
	if found {
		t.Errorf("expected found=false when understanding.json does not exist, got content: %s", content)
	}
}

func TestLearnModeInstruction_FirstRunVsReRun(t *testing.T) {
	firstRun := learnModeInstruction(false, "")
	if !strings.Contains(firstRun, "primera vez") {
		t.Errorf("expected first-run message to mention it's the first pass, got: %s", firstRun)
	}

	reRun := learnModeInstruction(true, "prior data")
	if !strings.Contains(reRun, "prior data") || !strings.Contains(reRun, "REFÍNALO") {
		t.Errorf("expected re-run message to embed prior data and ask to refine, got: %s", reRun)
	}
}
