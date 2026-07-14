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

		out := fmt.Sprintf("Se detectaron %d violaciones (CWD: %s):\n", len(violations), cwd)
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
		query, err := extractQueryArg(request)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		results, err := executeGraphQuery(query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonResult, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error formateando JSON: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	})
}

func extractQueryArg(request mcp.CallToolRequest) (string, error) {
	argsMap, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Argumentos inválidos")
	}
	query, ok := argsMap["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("Argumento 'query' es requerido")
	}
	return query, nil
}

func executeGraphQuery(query string) ([]map[string]interface{}, error) {
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, ".qdd", "knowledge.db")
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Error abriendo DB: %v", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("Error ejecutando query: %v", err)
	}
	defer rows.Close()

	return scanGraphRows(rows)
}

func scanGraphRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		if rowMap, err := scanSingleRow(rows, columns); err == nil {
			results = append(results, rowMap)
		}
	}
	return results, nil
}

func scanSingleRow(rows *sql.Rows, columns []string) (map[string]interface{}, error) {
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}
	
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}
	
	rowMap := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		if b, isBytes := val.([]byte); isBytes {
			rowMap[col] = string(b)
			continue
		}
		rowMap[col] = val
	}
	return rowMap, nil
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

func registerPostgresTunerTool(s *server.MCPServer) {
        tool := mcp.NewTool("qdd_postgres_tuner",
                mcp.WithDescription("Especialista en tuning de base de datos PostgreSQL. Analiza consultas lentas (SQL) y propone optimizaciones (índices, pg_class, parámetros de BD)."),
                mcp.WithString("query", mcp.Required(), mcp.Description("La consulta SQL que presenta problemas de rendimiento.")),
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

                analysis := fmt.Sprintf("Análisis de Especialista Tuning para Query:\n```sql\n%s\n```\n\n", query)
                analysis += "Recomendaciones de Performance (Dry-Run):\n"
                analysis += "1. **Conteo Masivo**: Si estás haciendo SELECT COUNT(*), recuerda que PostgreSQL bloquea y escanea toda la tabla. Usa `SELECT reltuples AS estimate FROM pg_class WHERE relname = 'tu_tabla';`.\n"
                analysis += "2. **Índices Faltantes**: Asegúrate de tener índices B-Tree en las columnas que usas en el WHERE o JOIN.\n"
                analysis += "3. **Parámetros Engine**: Si ves operaciones de ordenamiento en disco, considera subir `work_mem`.\n"
                analysis += "4. **Particionamiento**: Si la tabla tiene más de 100GB, evalúa usar particionamiento por rangos de fecha.\n\n"
                analysis += "NOTA: Esta es una herramienta estática y de análisis seguro. No modificó tu base de datos en producción."

                return mcp.NewToolResultText(analysis), nil
        })
}
