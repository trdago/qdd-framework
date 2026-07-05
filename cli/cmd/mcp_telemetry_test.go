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
	// Setup a temporary directory to act as the project root
	tempDir, err := os.MkdirTemp("", "qdd-learn-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save current working directory and restore it after test
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		os.Chdir(originalWD)
	}()

	// Change to the temporary directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}

	// Create docs directory and a dummy markdown file
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}
	
	dummyContent := "# Arquitectura de Prueba\n\nEste es un documento de prueba para validar que el contexto se lee correctamente."
	if err := os.WriteFile(filepath.Join(docsDir, "arquitectura.md"), []byte(dummyContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy markdown file: %v", err)
	}

	// Call handleLearnTool
	result, err := handleLearnTool(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handleLearnTool returned unexpected error: %v", err)
	}

	// Validate the result
	if result == nil {
		t.Fatalf("Expected non-nil CallToolResult")
	}

	if len(result.Content) == 0 {
		t.Fatalf("Expected non-empty content in CallToolResult")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected first content element to be mcp.TextContent")
	}

	// Verify the result contains the IDE instructions for Map-Reduce
	if !strings.Contains(textContent.Text, "(MAP-REDUCE COGNITIVO)") {
		t.Errorf("Result is missing Map-Reduce instructions. Got: %s", textContent.Text)
	}

	if strings.Contains(textContent.Text, "Arquitectura de Prueba") {
		t.Errorf("Result should NOT contain the file content directly. Got: %s", textContent.Text)
	}

	// Verify the knowledge_index.json was created
	indexPath := filepath.Join(tempDir, ".qdd", "knowledge_index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("Expected knowledge_index.json to be created at %s", indexPath)
	}
}
