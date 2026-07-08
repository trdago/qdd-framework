package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/audit"
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
		totalCerts, certifiedCerts := getCertificationStats()
		openFindings, resolvedFindings := getFindingsStats()

		finalScore, grado := calculateProjectScore(totalCerts, certifiedCerts, openFindings)

		out := formatScoreOutput(finalScore, grado, certifiedCerts, totalCerts, openFindings, resolvedFindings)
		return mcp.NewToolResultText(out), nil
	})
}

func getCertificationStats() (int, int) {
	qddDir := filepath.Join(".", ".qdd")
	certDirs := []string{
		filepath.Join(qddDir, "core", "certification"),
		filepath.Join(qddDir, "project", "certification"),
	}
	
	totalCerts := 0
	certifiedCerts := 0
	for _, certDir := range certDirs {
		t, c := processCertDir(certDir)
		totalCerts += t
		certifiedCerts += c
	}
	return totalCerts, certifiedCerts
}

func processCertDir(certDir string) (int, int) {
	total := 0
	certified := 0
	entries, _ := os.ReadDir(certDir)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			total++
			if isCertCertified(filepath.Join(certDir, entry.Name())) {
				certified++
			}
		}
	}
	return total, certified
}

func isCertCertified(path string) bool {
	content, _ := os.ReadFile(path)
	var cert Certification
	yaml.Unmarshal(content, &cert)
	return cert.Status == "certified"
}

func getFindingsStats() (int, int) {
	findDir := filepath.Join(".", ".qdd", "project", "findings")
	findEntries, _ := os.ReadDir(findDir)
	openFindings := 0
	resolvedFindings := 0
	
	for _, entry := range findEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			processFindingEntry(filepath.Join(findDir, entry.Name()), &openFindings, &resolvedFindings)
		}
	}
	return openFindings, resolvedFindings
}

func processFindingEntry(path string, open, resolved *int) {
	content, _ := os.ReadFile(path)
	var fnd Finding
	yaml.Unmarshal(content, &fnd)
	
	if fnd.Status == "open" {
		*open++
	}
	if fnd.Status == "resolved" {
		*resolved++
	}
}

func calculateProjectScore(totalCerts, certifiedCerts, openFindings int) (int, string) {
	finalScore := computeNumericScore(totalCerts, certifiedCerts, openFindings)
	grado := determineGrade(finalScore)
	return finalScore, grado
}

func computeNumericScore(totalCerts, certifiedCerts, openFindings int) int {
	baseScore := 100
	pendingCerts := totalCerts - certifiedCerts
	certPenalty := pendingCerts * 20
	findingPenalty := openFindings * 30
	finalScore := baseScore - certPenalty - findingPenalty
	if finalScore < 0 {
		return 0
	}
	return finalScore
}

func determineGrade(score int) string {
	if score >= 95 {
		return "A+ (World-Class)"
	}
	if score >= 80 {
		return "B (Bueno)"
	}
	if score >= 60 {
		return "C (Requiere Atención)"
	}
	if score >= 40 {
		return "D (Pobre)"
	}
	return "F"
}

func formatScoreOutput(finalScore int, grado string, certifiedCerts, totalCerts, openFindings, resolvedFindings int) string {
	out := fmt.Sprintf("**Puntaje Global:** %d / 100 (Grado: %s)\n", finalScore, grado)
	out += fmt.Sprintf("Certificaciones Cumplidas: %d/%d\n", certifiedCerts, totalCerts)
	out += fmt.Sprintf("Findings Abiertos: %d\n", openFindings)
	out += fmt.Sprintf("Findings Resueltos: %d\n", resolvedFindings)
	return out
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
	index := buildKnowledgeIndex(cwd)

	if len(index) == 0 {
		return mcp.NewToolResultText("No se encontraron documentos de arquitectura en docs/, rfcs/ o specification/."), nil
	}

	persistKnowledgeIndex(cwd, index)

	uiInstruction := buildUIInstruction(cwd)

	instructions := fmt.Sprintf(`[INSTRUCCIÓN PARA LA IA DEL IDE] (MAP-REDUCE COGNITIVO)
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
%s
`, uiInstruction)

	return mcp.NewToolResultText(instructions), nil
}

func buildKnowledgeIndex(cwd string) []KnowledgeIndexEntry {
	docFolders := []string{"docs", "rfcs", "specification"}
	var index []KnowledgeIndexEntry

	for _, folder := range docFolders {
		folderPath := filepath.Join(cwd, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			continue
		}
		
		index = append(index, scanDocFolder(cwd, folderPath)...)
	}
	return index
}

func scanDocFolder(cwd, folderPath string) []KnowledgeIndexEntry {
	var index []KnowledgeIndexEntry
	filepath.WalkDir(folderPath, func(path string, d os.DirEntry, err error) error {
		if err == nil && isMarkdownFile(d) {
			if entry := createKnowledgeEntry(cwd, path, d); entry != nil {
				index = append(index, *entry)
			}
		}
		return nil
	})
	return index
}

func isMarkdownFile(d os.DirEntry) bool {
	return !d.IsDir() && strings.HasSuffix(d.Name(), ".md")
}

func createKnowledgeEntry(cwd, path string, d os.DirEntry) *KnowledgeIndexEntry {
	relPath, _ := filepath.Rel(cwd, path)
	info, errInfo := d.Info()
	if errInfo == nil {
		return &KnowledgeIndexEntry{
			Path:      relPath,
			SizeBytes: info.Size(),
			ModTime:   info.ModTime().Format("2006-01-02 15:04:05"),
		}
	}
	return nil
}

func persistKnowledgeIndex(cwd string, index []KnowledgeIndexEntry) {
	indexData, _ := json.MarshalIndent(index, "", "  ")
	indexPath := filepath.Join(cwd, ".qdd", "knowledge_index.json")
	os.MkdirAll(filepath.Dir(indexPath), 0755)
	os.WriteFile(indexPath, indexData, 0644)
}

func isAutoUIEnabled(cwd string) bool {
	if configData, err := os.ReadFile(filepath.Join(cwd, ".qdd", "config.yaml")); err == nil {
		var config struct {
			AutoUICertification *bool `yaml:"auto_ui_certification"`
		}
		if yaml.Unmarshal(configData, &config) == nil && config.AutoUICertification != nil {
			return *config.AutoUICertification
		}
	}
	return true // default yes
}

func buildUIInstruction(cwd string) string {
	if !isAutoUIEnabled(cwd) {
		return ""
	}

	uiComponents := discoverUIComponents(cwd)
	
	sort.Slice(uiComponents, func(i, j int) bool {
		return uiComponents[i].SizeBytes > uiComponents[j].SizeBytes
	})
	
	var topUIComponents []string
	for i := 0; i < len(uiComponents) && i < 3; i++ {
		topUIComponents = append(topUIComponents, uiComponents[i].Path)
	}
	
	if len(topUIComponents) == 0 {
		return ""
	}

	return fmt.Sprintf(`
[AUTO-UI CERTIFICATION]
Se han detectado componentes de UI en el proyecto. Debes leer los 3 componentes más grandes:
%v

Tu tarea es:
1. Extraer el patrón de diseño (Tokens CSS, estructura, peso cognitivo).
2. Generar el archivo YAML técnico estricto en ".qdd/certifications/ui-consistency.yml" que use este patrón (ej. exigir var(--bg-panel)). El método de evaluación será por Percentage Score (umbral 80%%).
3. Generar el catálogo de prevención de duplicados en ".qdd/core-components.json" mapeando por intención analítica (ej. {"feedback_visual_carga": "LoadingSpinner.vue"}).
4. Fusionar esto con la documentación existente y escribir ".qdd/knowledge/design_system.md".
`, topUIComponents)
}

func discoverUIComponents(cwd string) []KnowledgeIndexEntry {
	var uiComponents []KnowledgeIndexEntry
	uiExtensions := map[string]bool{".vue": true, ".tsx": true, ".jsx": true, ".svelte": true, ".ts": true, ".js": true}
	
	filepath.WalkDir(cwd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		return processUIComponentEntry(cwd, path, d, uiExtensions, &uiComponents)
	})
	return uiComponents
}

func processUIComponentEntry(cwd, path string, d os.DirEntry, uiExtensions map[string]bool, uiComponents *[]KnowledgeIndexEntry) error {
	if shouldSkipUIDir(d) {
		return filepath.SkipDir
	}
	if isUIComponentFile(d, uiExtensions) {
		if entry := createKnowledgeEntry(cwd, path, d); entry != nil && entry.SizeBytes > 0 {
			*uiComponents = append(*uiComponents, *entry)
		}
	}
	return nil
}

func shouldSkipUIDir(d os.DirEntry) bool {
	if !d.IsDir() {
		return false
	}
	name := d.Name()
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "dist"
}

func isUIComponentFile(d os.DirEntry, uiExtensions map[string]bool) bool {
	return !d.IsDir() && uiExtensions[filepath.Ext(d.Name())]
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
		cwd, _ := os.Getwd()
		p := audit.LoadPolicies(cwd)
		prompt := harness.GenerateSystemPrompt(p.AllowExecution)
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
