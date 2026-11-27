package cmd

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/integration"
)

// Rule: Prohibido usar 'else'
// Servidor MCP de QDD en Claude Code y OpenCode (FND-308): sin él la IA tiene las reglas pero no las tools de QDD.

const (
	itemMCPClaude   = "Servidor MCP de QDD en Claude Code (.mcp.json)"
	itemMCPOpenCode = "Servidor MCP de QDD en OpenCode (opencode.json)"
)

func itemsMCP(projectPath string) []*CheckItem {
	nombres := map[string]string{"Claude Code": itemMCPClaude, "OpenCode": itemMCPOpenCode}
	var items []*CheckItem
	for _, e := range integration.EstadoMCP(projectPath) {
		items = append(items, itemMCP(nombres[e.IA], e))
	}
	return items
}

func itemMCP(nombre string, e integration.EstadoArchivoMCP) *CheckItem {
	item := &CheckItem{Name: nombre, Success: e.Estado == integration.MCPConectado}
	switch e.Estado {
	case integration.MCPFalta:
		item.Error = fmt.Sprintf("falta qdd en %s: qdd doctor --fix lo agrega sin tocar lo demás", e.Archivo)
	case integration.MCPIlegible:
		item.Error = fmt.Sprintf("%s no es JSON válido: arréglalo; --fix no lo pisa", e.Archivo)
	case integration.MCPAMano:
		item.Success = true
		item.Error = "el proyecto usa opencode.jsonc: agrega a mano \"mcp\": {\"qdd\": {\"type\": \"local\", \"command\": [\"qdd\", \"mcp-server\"]}}"
	}
	return item
}
