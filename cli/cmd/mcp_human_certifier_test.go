package cmd

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestExtractHumanTargetArg(t *testing.T) {
	// Case 1: Empty params
	req1 := mcp.CallToolRequest{}
	if target := extractHumanTargetArg(req1); target != "all" {
		t.Fatalf("Expected 'all' for empty arguments, got: %s", target)
	}

	// Case 2: Explicit target
	req2 := mcp.CallToolRequest{}
	req2.Params.Arguments = map[string]interface{}{
		"target": "accessibility",
	}
	if target := extractHumanTargetArg(req2); target != "accessibility" {
		t.Fatalf("Expected 'accessibility', got: %s", target)
	}
}
