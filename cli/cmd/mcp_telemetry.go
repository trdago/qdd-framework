package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
	"github.com/qdd-framework/qdd/pkg/qcl/harness"
	"github.com/qdd-framework/qdd/pkg/topology"
	"gopkg.in/yaml.v3"
)

func registerScoreTool(s *server.MCPServer) {
	registerGraphQueryTool(s)
	registerHarnessTool(s)
	tool := mcp.NewTool("qdd_score",
		mcp.WithDescription("Calcula y devuelve el puntaje de calidad del proyecto basado en certificaciones y findings."),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		qddDir := filepath.Join(".", ".qdd")
		certDirs := []string{
			filepath.Join(qddDir, "core", "certification"),
			filepath.Join(qddDir, "project", "certification"),
		}
		
		totalCerts := 0
		certifiedCerts := 0
		for _, certDir := range certDirs {
			entries, _ := os.ReadDir(certDir)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
					totalCerts++
					content, _ := os.ReadFile(filepath.Join(certDir, entry.Name()))
					var cert Certification
					yaml.Unmarshal(content, &cert)
					if cert.Status == "certified" {
						certifiedCerts++
					}
				}
			}
		}

		findDir := filepath.Join(qddDir, "project", "findings")
		findEntries, _ := os.ReadDir(findDir)
		openFindings := 0
		resolvedFindings := 0
		for _, entry := range findEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				content, _ := os.ReadFile(filepath.Join(findDir, entry.Name()))
				var fnd Finding
				yaml.Unmarshal(content, &fnd)
				if fnd.Status == "open" {
					openFindings++
					continue
				}
				if fnd.Status == "resolved" {
					resolvedFindings++
				}
			}
		}

		baseScore := 100
		pendingCerts := totalCerts - certifiedCerts
		certPenalty := pendingCerts * 20
		findingPenalty := openFindings * 30
		finalScore := baseScore - certPenalty - findingPenalty
		if finalScore < 0 {
			finalScore = 0
		}

		grado := "F"
		if finalScore >= 95 {
			grado = "A+ (World-Class)"
		}
		if finalScore >= 80 && finalScore < 95 {
			grado = "B (Bueno)"
		}
		if finalScore >= 60 && finalScore < 80 {
			grado = "C (Requiere Atención)"
		}
		if finalScore >= 40 && finalScore < 60 {
			grado = "D (Pobre)"
		}

		out := fmt.Sprintf("**Puntaje Global:** %d / 100 (Grado: %s)\n", finalScore, grado)
		out += fmt.Sprintf("Certificaciones Cumplidas: %d/%d\n", certifiedCerts, totalCerts)
		out += fmt.Sprintf("Findings Abiertos: %d\n", openFindings)
		out += fmt.Sprintf("Findings Resueltos: %d\n", resolvedFindings)
		
		return mcp.NewToolResultText(out), nil
	})
}

func registerStatusTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_status",
		mcp.WithDescription("Muestra el estado de gobernanza del proyecto."),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		qddDir := filepath.Join(".", ".qdd")
		statePath := filepath.Join(qddDir, "state.json")
		stateData, err := os.ReadFile(statePath)
		out := "--- QDD PROJECT STATUS ---\n"
		if err == nil {
			var state map[string]interface{}
			json.Unmarshal(stateData, &state)
			out += fmt.Sprintf("Versión: %v\nEstado: %v\n", state["version"], state["status"])
		}
		
		return mcp.NewToolResultText(out), nil
	})
}

func registerLearnTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_learn",
		mcp.WithDescription("Aprende la arquitectura y documentación del proyecto (Fast Path)"),
	)
	s.AddTool(tool, handleLearnTool)
}

type KnowledgeIndexEntry struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	ModTime   string `json:"mod_time"`
}

func handleLearnTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cwd, _ := os.Getwd()
	docFolders := []string{"docs", "rfcs", "specification"}
	var index []KnowledgeIndexEntry

	for _, folder := range docFolders {
		folderPath := filepath.Join(cwd, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			continue
		}
		filepath.WalkDir(folderPath, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
				relPath, _ := filepath.Rel(cwd, path)
				info, errInfo := d.Info()
				if errInfo == nil {
					index = append(index, KnowledgeIndexEntry{
						Path:      relPath,
						SizeBytes: info.Size(),
						ModTime:   info.ModTime().Format("2006-01-02 15:04:05"),
					})
				}
			}
			return nil
		})
	}

	if len(index) == 0 {
		return mcp.NewToolResultText("No se encontraron documentos de arquitectura en docs/, rfcs/ o specification/."), nil
	}

	// Persist the knowledge index for the IDE to read iteratively
	indexData, _ := json.MarshalIndent(index, "", "  ")
	indexPath := filepath.Join(cwd, ".qdd", "knowledge_index.json")
	os.MkdirAll(filepath.Dir(indexPath), 0755)
	os.WriteFile(indexPath, indexData, 0644)

	instructions := `[INSTRUCCIÓN PARA LA IA DEL IDE] (MAP-REDUCE COGNITIVO)
Como Arquitecto Principal del Proyecto, debes leer los documentos listados en .qdd/knowledge_index.json y generar el reporte de inteligencia (Intelligence Report).
Por favor genera un archivo JSON estrictamente válido en la ruta ".qdd/understanding.json" con la siguiente estructura exacta:
{
  "summary": "Resumen ejecutivo del proyecto",
  "components": ["Componente 1", "Componente 2"],
  "objectives": ["Objetivo 1", "Objetivo 2"],
  "guidelines": ["Regla 1", "Regla 2"],
  "next_steps": "Siguientes pasos recomendados"
}

1. NO intentes leer todos los documentos de golpe.
2. Usa tus herramientas nativas para leer los archivos uno por uno según el índice.
3. Al finalizar, genera el archivo .qdd/understanding.json.
`

	return mcp.NewToolResultText(instructions), nil
}

func registerGraphQueryTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_graph_query",
		mcp.WithDescription("Consultar el Grafo de Conocimiento (Knowledge Graph) mediante consultas SQLite (Pure-Go) embebidas."),
		mcp.WithString("query", mcp.Required(), mcp.Description("La consulta SQL a ejecutar sobre las tablas 'nodes' y 'edges'.")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid arguments format")
		}
		query, ok := args["query"].(string)
		if !ok {
			return nil, fmt.Errorf("argument 'query' is missing or not a string")
		}

		db, err := graph.InitDB()
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error inicializando la base de datos de grafos: %v", err)), nil
		}
		defer db.Close()

		return mcp.NewToolResultText(fmt.Sprintf("Recibida consulta para el grafo: %s\n(Funcionalidad en desarrollo para la fase 2)", query)), nil
	})
}

func registerHarnessTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_harness_generate",
		mcp.WithDescription("Genera el QDD Agentic Harness (System Prompt) combinando Claude, Antigravity, Cursor y Hermes."),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prompt := harness.GenerateSystemPrompt()
		return mcp.NewToolResultText(prompt), nil
	})
}

func registerMapTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_map",
		mcp.WithDescription("Genera el mapa topológico de certificación del proyecto."),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cwd, _ := os.Getwd()
		projTopology, err := topology.MapProject(cwd)
		if err != nil {
			return mcp.NewToolResultText("Error al mapear el proyecto: " + err.Error()), nil
		}
		
		qddDir := filepath.Join(cwd, ".qdd", "project")
		os.MkdirAll(qddDir, 0755)
		outPath := filepath.Join(qddDir, "topology.json")
		data, _ := json.MarshalIndent(projTopology, "", "  ")
		os.WriteFile(outPath, data, 0644)
		
		return mcp.NewToolResultText(fmt.Sprintf("Topología generada. Score Global: %d%%", projTopology.GlobalScore)), nil
	})
}
