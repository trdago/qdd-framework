package cmd

import (
	"context"
	"os"
	
	"github.com/spf13/cobra"
	"github.com/mark3labs/mcp-go/server"
)

type Certification struct {
	ID     string `yaml:"id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`
}

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
		
		registerAuditTool(s)
		registerCertifyTool(s)
		
		registerScoreTool(s)
		registerStatusTool(s)
		registerLearnTool(s)
		registerMapTool(s)
		
		registerFindingsTool(s)
		registerSprintTool(s)
		registerSyncTool(s)
		registerReleaseTool(s)
		
		registerQueryGraphTool(s)
		registerSyncGraphTool(s)
		registerPostgresTunerTool(s)
		
		// Start serving
		return ss.Listen(context.Background(), os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
