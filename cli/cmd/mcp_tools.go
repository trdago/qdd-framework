package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
	_ "modernc.org/sqlite"
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

func registerQueryGraphTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_query_graph",
		mcp.WithDescription("Ejecuta consultas SQL directamente en la base de datos de grafos de QDD (.qdd/knowledge.db)."),
		mcp.WithString("query", mcp.Required(), mcp.Description("La consulta SQL a ejecutar (SELECT ... FROM nodes/edges).")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}
		query, ok := argsMap["query"].(string)
		if !ok || query == "" {
			return mcp.NewToolResultError("Argumento 'query' es requerido"), nil
		}

		cwd, _ := os.Getwd()
		dbPath := filepath.Join(cwd, ".qdd", "knowledge.db")
		
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error abriendo DB: %v", err)), nil
		}
		defer db.Close()

		rows, err := db.Query(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error ejecutando query: %v", err)), nil
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		var results []map[string]interface{}

		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}
			
			if err := rows.Scan(valuePtrs...); err != nil {
				continue
			}
			
			rowMap := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				b, isBytes := val.([]byte)
				if isBytes {
					rowMap[col] = string(b)
				}
				if !isBytes {
					rowMap[col] = val
				}
			}
			results = append(results, rowMap)
		}

		jsonResult, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error formateando JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	})
}

func registerSyncGraphTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_sync_graph",
		mcp.WithDescription("Fuerza la sincronización y actualización del motor GraphRAG (Knowledge.db)."),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		gdb, err := graph.InitDB()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error inicializando DB: %v", err)), nil
		}
		
		cwd, _ := os.Getwd()
		err = gdb.SyncToGraph(cwd)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error sincronizando grafos: %v", err)), nil
		}
		
		return mcp.NewToolResultText("GraphRAG sincronizado exitosamente. La topología está actualizada."), nil
	})
}
