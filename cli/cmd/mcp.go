package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

type Finding struct {
	ID       string `yaml:"id"`
	Type     string `yaml:"type"`
	Title    string `yaml:"title"`
	Status   string `yaml:"status"`
	Severity string `yaml:"severity"`
}

var mcpCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Inicia el servidor MCP nativo de QDD (JSON-RPC sobre Stdio)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize the MCP server
		s := server.NewMCPServer("QDD-MCP", "1.4.0")

		// Create the stdio server
		ss := server.NewStdioServer(s)

		registerMCPTools(s)

		// Start serving
		return ss.Listen(context.Background(), os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

// registerMCPTools registra todas las tools que expone 'qdd mcp-server'.
// docs/command-reference.md debe documentar exactamente estas (FND-072).
func registerMCPTools(s *server.MCPServer) {
	registerProjectContextTool(s)
	registerModuleSaveTool(s)
	registerBlueprintTools(s)
	registerEstadoTool(s)
	registerFotoTool(s)
	registerConsolaTools(s)
	registerAuditTool(s)
	registerCertifyTool(s)

	registerScoreTool(s)
	registerStatusTool(s)
	registerLearnTool(s)
	registerEvolutionTool(s)
	registerMapTool(s)

	registerFindingsTool(s)
	registerSprintTool(s)
	registerSyncTool(s)
	registerReleaseTool(s)

	registerQueryGraphTool(s)
	registerSyncGraphTool(s)
	registerPostgresTunerTool(s)

	registerResolveDiagnoseTool(s)
	registerResolveCertifyTool(s)

	registerAIRiskAssessTool(s)
	registerAITestIntegrityTool(s)

	registerReportTool(s)
	registerLocalLLMTools(s)
	registerHumanCertifierTool(s)
}

// soloLeen son las tools que no cambian nada: llevan readOnlyHint. Las demás van sin anotaciones, que en MCP valen lo
// mismo que las que mcp-go pone por defecto (pueden escribir), sin gastar tokens en cada sesión.
var soloLeen = map[string]bool{
	"qdd_project_context": true, "qdd_blueprint_list": true, "qdd_findings": true, "qdd_score": true, "qdd_status": true,
	"qdd_evolution": true, "qdd_map": true, "qdd_local_llm_status": true, "qdd_foto": true,
}

func anotaciones(nombre string) mcp.ToolAnnotation {
	if !soloLeen[nombre] {
		return mcp.ToolAnnotation{}
	}
	si := true
	return mcp.ToolAnnotation{ReadOnlyHint: &si}
}

func RegisterToolWithTelemetry(s *server.MCPServer, tool mcp.Tool, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool.Annotations = anotaciones(tool.Name)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (res *mcp.CallToolResult, err error) {
		defer func() {
			if r := recover(); r != nil {
				recordMCPActivity("panic_recovered", tool.Name, map[string]interface{}{"panic": fmt.Sprintf("%v", r)})
				res = mcp.NewToolResultError(fmt.Sprintf("Zero-Panic Guard: Excepción recuperada de forma segura en herramienta '%s': %v", tool.Name, r))
				err = nil
			}
		}()

		recordMCPActivity("start", tool.Name, request.Params.Arguments)
		cwd, _ := os.Getwd()
		registrarUso(cwd)
		res, err = handler(ctx, request)
		if err != nil {
			recordMCPActivity("error", tool.Name, map[string]interface{}{"error": err.Error()})
			return res, err
		}
		recordMCPActivity("success", tool.Name, nil)
		return res, err
	})
}

func recordMCPActivity(status, toolName string, details interface{}) {
	cwd, _ := os.Getwd()
	metricsDir := filepath.Join(cwd, ".qdd", "project", "metrics")
	_ = os.MkdirAll(metricsDir, 0755)

	event := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"tool":      toolName,
		"status":    status,
		"details":   details,
	}

	historyPath := filepath.Join(metricsDir, "cognitive_history.json")
	var history []interface{}

	if data, err := os.ReadFile(historyPath); err == nil {
		_ = json.Unmarshal(data, &history)
	}

	history = append(history, event)

	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	if out, err := json.MarshalIndent(history, "", "  "); err == nil {
		_ = os.WriteFile(historyPath, out, 0644)
	}
}
