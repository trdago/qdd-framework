package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/qdd-framework/qdd/pkg/qcl/benchmark"
)

// Rule: Prohibido usar 'else'
// FND-304: el servidor MCP de una consola abierta sigue con el binario viejo después de actualizar, y nadie lo avisaba.

func textoDeTool(t *testing.T, s *server.MCPServer, nombre string) string {
	t.Helper()
	res, err := s.GetTool(nombre).Handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatal(err)
	}
	texto := ""
	for _, c := range res.Content {
		texto += c.(mcp.TextContent).Text + "\n"
	}
	return texto
}

func servidorConBinario(t *testing.T, cambio *bool) *server.MCPServer {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	anterior, dicho := binarioCambio, avisoDado
	binarioCambio = func() bool { return *cambio }
	avisoDado = false
	t.Cleanup(func() { binarioCambio, avisoDado = anterior, dicho })
	s := server.NewMCPServer("prueba", "1")
	responde := func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("resultado"), nil
	}
	RegisterToolWithTelemetry(s, mcp.NewTool("qdd_project_context"), responde)
	RegisterToolWithTelemetry(s, mcp.NewTool("qdd_score"), responde)
	return s
}

func TestMCP_AvisaQueCorreUnBinarioViejo_FND304(t *testing.T) {
	cambio := false
	s := servidorConBinario(t, &cambio)
	if got := textoDeTool(t, s, "qdd_score"); got != "resultado\n" {
		t.Fatalf("sin actualización no cambia la respuesta: %q", got)
	}
	cambio = true
	got := textoDeTool(t, s, "qdd_score")
	if !strings.HasPrefix(got, "resultado\n") || !strings.Contains(got, "/mcp") || !strings.Contains(got, Version) {
		t.Fatalf("conserva el resultado y dice qué versión corre y cómo reconectar: %q", got)
	}
	if got := textoDeTool(t, s, "qdd_score"); got != "resultado\n" {
		t.Fatalf("no se repite en cada llamada: %q", got)
	}
	if got := textoDeTool(t, s, "qdd_project_context"); !strings.Contains(got, "/mcp") {
		t.Fatalf("antes de cambiar código se vuelve a decir: %q", got)
	}
}

func TestDoctor_DiceQueUnaConsolaCorreUnMCPViejo_FND304(t *testing.T) {
	cambio := false
	s := servidorConBinario(t, &cambio)
	root, _ := os.Getwd()
	if item := checkMCPAlDia(root); item != nil {
		t.Fatalf("sin eventos no hay qué revisar: %+v", item)
	}
	cambio = true
	textoDeTool(t, s, "qdd_score")
	item := checkMCPAlDia(root)
	if item == nil || !item.Success || !strings.Contains(item.Error, "/mcp") || !strings.Contains(item.Error, Version) {
		t.Fatalf("lo informa sin fallar, con la versión y qué hacer: %+v", item)
	}
	cambio = false
	textoDeTool(t, s, "qdd_score")
	if item := checkMCPAlDia(root); item != nil {
		t.Fatalf("al reconectar deja de informarlo: %+v", item)
	}
}

func TestDoctor_UnMCPViejoQueYaTerminoNoSeInforma_FND304(t *testing.T) {
	root := t.TempDir()
	if err := benchmark.AnotarEventoMCP(root, eventoMCP{Tool: "qdd_score", PID: 1 << 30, Version: "v1.0.0", Atrasado: true}); err != nil {
		t.Fatal(err)
	}
	if item := checkMCPAlDia(root); item != nil {
		t.Fatalf("la consola ya se cerró: %+v", item)
	}
}
