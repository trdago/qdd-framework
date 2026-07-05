package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/audit"
)

func registerAuditTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_audit",
		mcp.WithDescription("Ejecuta la auditoría estricta de QDD en el proyecto y devuelve las violaciones."),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cwd, _ := os.Getwd()
		engine := audit.NewEngine(cwd)
		violations := engine.RunAll()

		if len(violations) == 0 {
			return mcp.NewToolResultText("Auditoría exitosa. El código cumple con las certificaciones."), nil
		}

		out := fmt.Sprintf("Se detectaron %d violaciones:\n", len(violations))
		for _, v := range violations {
			out += fmt.Sprintf("- [%s] %s\n", v.Category, v.Format())
		}
		
		return mcp.NewToolResultText(out), nil
	})
}

func registerCertifyTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_certify",
		mcp.WithDescription("Certifica que el código cumple con los estándares listos para release."),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cwd, _ := os.Getwd()
		engine := audit.NewEngine(cwd)
		violations := engine.RunAll()
		
		if len(violations) == 0 {
			return mcp.NewToolResultText("CERTIFICADO! Listo para release."), nil
		}
		
		return mcp.NewToolResultText("RECHAZADO! Existen violaciones de auditoría."), nil
	})
}

