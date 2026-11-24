package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
	_ "modernc.org/sqlite"
)

func registerAuditTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_audit",
		mcp.WithDescription("Gate de QDD (pruebas + reglas): devuelve las violaciones."),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cwd, _ := os.Getwd()
		engine := audit.NewEngine(cwd).ParandoAlBloquear()
		return mcp.NewToolResultText(formatAuditResult(engine.Evaluar())), nil
	})
}

func registerCertifyTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_certify",
		mcp.WithDescription("Certifica el código para release."),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cwd, _ := os.Getwd()
		engine := audit.NewEngine(cwd)
		r := engine.EvaluarCompleto()
		violations := r.Violaciones
		cobertura := "Cobertura: " + resumenDeCobertura(r.Cobertura) + "\n" + formatCobertura(r.Cobertura)

		if len(audit.Bloqueantes(violations)) == 0 {
			return mcp.NewToolResultText("CERTIFICADO! Listo para release.\n" + cobertura + formatRecomendaciones(audit.Recomendaciones(violations))), nil
		}

		return mcp.NewToolResultText("RECHAZADO! Existen violaciones de auditoría.\n" + cobertura), nil
	})
}

func registerQueryGraphTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_query_graph",
		mcp.WithDescription("SQL de solo lectura en .qdd/knowledge.db."),
		mcp.WithString("query", mcp.Required(), mcp.Description("SELECT … FROM nodes/edges.")),
	)

	RegisterToolWithTelemetry(s, tool, consultarGrafo)
}

// consultarGrafo atiende qdd_query_graph y qdd_graph_query: las dos leen lo mismo.
func consultarGrafo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

var empiezaComoConsulta = regexp.MustCompile(`(?i)^(select|with|explain)\b`)

// unaConsulta admite una sola sentencia que lee: mode=ro protege la base del grafo, no una que se adjunte con ATTACH.
func unaConsulta(query string) (string, error) {
	q := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(query), ";"))
	if strings.Contains(q, ";") || !empiezaComoConsulta.MatchString(q) {
		return "", errors.New("solo se admite una consulta (SELECT, WITH o EXPLAIN), sin «;» en medio")
	}
	return q, nil
}

// executeGraphQuery abre la base solo para leer (FND-312).
func executeGraphQuery(query string) ([]map[string]interface{}, error) {
	consulta, err := unaConsulta(query)
	if err != nil {
		return nil, err
	}
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, ".qdd", "knowledge.db")
	if !fileExists(dbPath) {
		return nil, errors.New("no hay grafo todavía: créalo con qdd_sync_graph")
	}

	soloLectura := url.URL{Scheme: "file", Path: filepath.ToSlash(dbPath), RawQuery: "mode=ro&_pragma=query_only(1)"}
	db, err := sql.Open("sqlite", soloLectura.String())
	if err != nil {
		return nil, fmt.Errorf("Error abriendo DB: %v", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(context.Background(), consulta)
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
	return results, rows.Err()
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
		mcp.WithDescription("Sincroniza el grafo de conocimiento (knowledge.db)."),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		mcp.WithDescription("Consejos generales de PostgreSQL; no analiza la consulta."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Consulta SQL lenta.")),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}
		query, ok := argsMap["query"].(string)
		if !ok || query == "" {
			return mcp.NewToolResultError("Argumento 'query' es requerido"), nil
		}

		analysis := fmt.Sprintf("Consejos generales de PostgreSQL: qdd no analiza la consulta ni se conecta a la base.\n```sql\n%s\n```\n\n", query)
		analysis += "1. **Conteo Masivo**: Si estás haciendo SELECT COUNT(*), recuerda que PostgreSQL bloquea y escanea toda la tabla. Usa `SELECT reltuples AS estimate FROM pg_class WHERE relname = 'tu_tabla';`.\n"
		analysis += "2. **Índices Faltantes**: Asegúrate de tener índices B-Tree en las columnas que usas en el WHERE o JOIN.\n"
		analysis += "3. **Parámetros Engine**: Si ves operaciones de ordenamiento en disco, considera subir `work_mem`.\n"
		analysis += "4. **Particionamiento**: Si la tabla tiene más de 100GB, evalúa usar particionamiento por rangos de fecha.\n\n"

		return mcp.NewToolResultText(analysis), nil
	})
}
