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
