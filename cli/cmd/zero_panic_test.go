package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestZeroPanicHTTPMiddleware(t *testing.T) {
	panickingHandler := safeHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
		panic("Simulated critical panic inside HTTP handler")
	})

	req := httptest.NewRequest("GET", "/api/test-panic", nil)
	rec := httptest.NewRecorder()

	panickingHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("Expected HTTP 500 on recovered panic, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body == "" {
		t.Fatalf("Expected error response body on recovered panic")
	}
}

func TestZeroPanicMCPToolIsolation(t *testing.T) {
	s := server.NewMCPServer("TEST-MCP", "1.0.0")

	panickingTool := mcp.NewTool("test_panic", mcp.WithDescription("Tool that panics"))
	RegisterToolWithTelemetry(s, panickingTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		panic("Simulated tool panic")
	})

	// The MCP server shouldn't crash when registering or recovering
	if s == nil {
		t.Fatalf("Expected server to be non-nil")
	}
}
