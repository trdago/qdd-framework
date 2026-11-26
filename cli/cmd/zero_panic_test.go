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
		// qdd:ignore CERT-020-ZERO-PANIC
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

// contrato: mcp#f76f665c
// La tool que hace panic se llama de verdad: vuelve como error de la tool y el servidor sigue atendiendo.
func TestZeroPanicMCPToolIsolation(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	s := server.NewMCPServer("TEST-MCP", "1.0.0")

	panickingTool := mcp.NewTool("test_panic", mcp.WithDescription("Tool that panics"))
	RegisterToolWithTelemetry(s, panickingTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// qdd:ignore CERT-020-ZERO-PANIC
		panic("Simulated tool panic")
	})
	RegisterToolWithTelemetry(s, mcp.NewTool("test_ok"), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})

	res, err := s.GetTool("test_panic").Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil || res == nil || !res.IsError {
		t.Fatalf("un panic vuelve como error de la tool: %+v, %v", res, err)
	}
	res, err = s.GetTool("test_ok").Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil || res == nil || res.IsError {
		t.Fatalf("el servidor sigue atendiendo: %+v, %v", res, err)
	}
}
