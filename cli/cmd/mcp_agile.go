package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/integration"
	"gopkg.in/yaml.v3"
)

func registerFindingsTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_findings",
		mcp.WithDescription("Muestra todos los hallazgos técnicos (bugs, vulnerabilidades) del proyecto"),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		qddDir := filepath.Join(".", ".qdd", "project", "findings")
		if _, err := os.Stat(qddDir); os.IsNotExist(err) {
			return mcp.NewToolResultText("No se encontró el directorio de findings (.qdd/project/findings)."), nil
		}

		files, err := os.ReadDir(qddDir)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error leyendo directorio de findings: %v", err)), nil
		}

		out, total := processFindingsFiles(qddDir, files)

		if total == 0 {
			return mcp.NewToolResultText("No hay hallazgos registrados en el proyecto."), nil
		}

		return mcp.NewToolResultText(out), nil
	})
}

func processFindingsFiles(qddDir string, files []os.DirEntry) (string, int) {
	out := "--- QDD FINDINGS REPORT ---\n\n"
	total := 0
	resolved := 0
	pending := 0

	for _, file := range files {
		if shouldProcessFinding(file) {
			processSingleFinding(filepath.Join(qddDir, file.Name()), &out, &total, &resolved, &pending)
		}
	}

	out += "\n-------------------------------\n"
	out += fmt.Sprintf("Total: %d | Resueltos: %d | Pendientes: %d\n", total, resolved, pending)
	return out, total
}

func shouldProcessFinding(file os.DirEntry) bool {
	return !file.IsDir() && strings.HasSuffix(file.Name(), ".yaml")
}

func processSingleFinding(path string, out *string, total, resolved, pending *int) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var f Finding
	if err := yaml.Unmarshal(content, &f); err != nil {
		return
	}

	*total++
	icon := "🔴"
	statusLower := strings.ToLower(f.Status)
	
	if statusLower == "resolved" || statusLower == "closed" {
		icon = "🟢"
		*resolved++
		*out += fmt.Sprintf("%s [%s] [%s] %s - %s\n", icon, f.ID, strings.ToUpper(f.Severity), strings.ToUpper(f.Status), f.Title)
		return
	}
	*pending++
	*out += fmt.Sprintf("%s [%s] [%s] %s - %s\n", icon, f.ID, strings.ToUpper(f.Severity), strings.ToUpper(f.Status), f.Title)
}

func registerSprintTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_sprint",
		mcp.WithDescription("Inicializa un nuevo Sprint interactivo bajo QDD Certification Standard (QCS v1.1). IMPORTANTE: Debes exigir al usuario que defina el Happy Path y explícitamente los Error Paths. Los Edge Cases son opcionales. Para los tests ejecutables, NO se los preguntes al usuario; tú como IA debes deducir y generar los comandos de test adecuados (ej. go test, npm run build) basados en tu conocimiento del stack tecnológico."),
		mcp.WithString("numero", mcp.Required(), mcp.Description("Número del sprint a crear (ej: 1, 2, 3)")),
		mcp.WithString("goal", mcp.Required(), mcp.Description("Objetivo principal claro y sin ambigüedad")),
		mcp.WithString("happy_path", mcp.Required(), mcp.Description("Reglas del camino feliz (ej: Dado X, Cuando Y, Entonces Z) separadas por punto y coma (;)")),
		mcp.WithString("error_paths", mcp.Required(), mcp.Description("Reglas para escenarios de error (ej: Dado un error de DB, Entonces...) separadas por punto y coma (;)")),
		mcp.WithString("edge_cases", mcp.Description("Casos límite opcionales (ej: timeouts, concurrencia) separados por punto y coma (;)")),
		mcp.WithString("technical_constraints", mcp.Required(), mcp.Description("Restricciones técnicas o de arquitectura separadas por punto y coma (;)")),
		mcp.WithString("executable_tests", mcp.Required(), mcp.Description("Comandos de terminal generados POR LA IA (tú) para validar esto automáticamente separados por punto y coma (;). No pidas esto al usuario, genéralo tú.")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}
		
		sprintParams, err := extractSprintParams(argsMap)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		
		sprintsDir := filepath.Join(".", ".qdd", "project", "sprints")
		os.MkdirAll(sprintsDir, 0755)
		
		sprintFile := filepath.Join(sprintsDir, fmt.Sprintf("sprint-%s.md", sprintParams["numero"]))
		certFile := filepath.Join(sprintsDir, fmt.Sprintf("sprint-%s-cert.yaml", sprintParams["numero"]))
		
		if _, err := os.Stat(certFile); !os.IsNotExist(err) {
			return mcp.NewToolResultText(fmt.Sprintf("El sprint/certificado %s ya existe.", sprintParams["numero"])), nil
		}

		generateSprintFiles(sprintFile, certFile, sprintParams)

		return mcp.NewToolResultText(fmt.Sprintf("Sprint %s inicializado correctamente en modo CDD v1.1.\nContrato generado en: %s\nDocumento generado en: %s\n\nEl Agente IA DEBE ahora leer el archivo YAML y comenzar el bucle autónomo programando cada escenario listado hasta que los 'executable_tests' pasen sin errores.", sprintParams["numero"], certFile, sprintFile)), nil
	})
}

func extractSprintParams(argsMap map[string]interface{}) (map[string]string, error) {
	getString := func(key string) string {
		if val, ok := argsMap[key].(string); ok {
			return val
		}
		return ""
	}

	params := map[string]string{
		"numero": getString("numero"),
		"goal": getString("goal"),
		"happy_path": getString("happy_path"),
		"error_paths": getString("error_paths"),
		"edge_cases": getString("edge_cases"),
		"technical_constraints": getString("technical_constraints"),
		"executable_tests": getString("executable_tests"),
	}

	if err := validateSprintParams(params); err != nil {
		return nil, err
	}

	return params, nil
}

func validateSprintParams(params map[string]string) error {
	requiredParams := []string{"numero", "goal", "happy_path", "error_paths", "executable_tests"}
	for _, req := range requiredParams {
		if params[req] == "" {
			return fmt.Errorf("Faltan parámetros obligatorios (numero, goal, happy_path, error_paths, executable_tests). Vuelve a preguntar al usuario por los escenarios faltantes, pero recuerda generar TÚ MISMO los executable_tests.")
		}
	}
	return nil
}

func generateSprintFiles(sprintFile, certFile string, params map[string]string) {
	certContent := generateCertYAMLContent(params)
	os.WriteFile(certFile, []byte(certContent), 0644)

	content := fmt.Sprintf("# Sprint %s\n\n## Objetivos (Sprint Goal)\n%s\n\n## Certification-Driven Development (CDD v1.1)\nEste sprint se rige por un contrato estricto de escenarios exhaustivos.\nEl agente debe implementar el código y los tests para el **Happy Path**, **Error Paths** y **Edge Cases** definidos en `sprint-%s-cert.yaml`.\n\n---\n*Gobernanza QDD: Este archivo y su YAML asociado guían el desarrollo. Usa 'qdd certify' o ejecuta los tests directamente.*\n", params["numero"], params["goal"], params["numero"])
	os.WriteFile(sprintFile, []byte(content), 0644)
}

func generateCertYAMLContent(params map[string]string) string {
	certContent := fmt.Sprintf(`qdd_certificate:
  version: "1.1"
  sprint_id: "%s"
  status: "FAILING"
  goal: "%s"
  
  scenarios:
    happy_path:`, params["numero"], params["goal"])
	
	certContent += appendRules(params["happy_path"], "      - ")
	certContent += "\n    error_paths:"
	certContent += appendRules(params["error_paths"], "      - ")
	certContent += "\n    edge_cases:"
	if params["edge_cases"] != "" {
		certContent += appendRules(params["edge_cases"], "      - ")
	}
	
	certContent += "\n\n  technical_constraints:"
	certContent += appendRules(params["technical_constraints"], "    - ")
	certContent += "\n\n  executable_tests:"
	certContent += appendRules(params["executable_tests"], "    - ")
	certContent += "\n"

	return certContent
}

func appendRules(ruleString, prefix string) string {
	var result string
	for _, rule := range strings.Split(ruleString, ";") {
		rule = strings.TrimSpace(rule)
		if rule != "" {
			result += fmt.Sprintf("\n%s\"%s\"", prefix, rule)
		}
	}
	return result
}

func registerSyncTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_sync",
		mcp.WithDescription("Sincroniza las reglas nativas (QDD Protocol) con los asistentes de IA"),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error obteniendo directorio actual: %v", err)), nil
		}
		cwd = integration.FindProjectRoot(cwd)

		manager := integration.NewIntegrationManager()
		err = manager.SyncAll(cwd)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error durante la sincronización: %v", err)), nil
		}

		return mcp.NewToolResultText("¡Sincronización completada! Tu asistente de IA ahora comprende comandos nativos QDD."), nil
	})
}

func registerReleaseTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_release",
		mcp.WithDescription("Empaqueta una nueva versión del framework"),
		mcp.WithString("version", mcp.Required(), mcp.Description("Versión a liberar (ej: v1.0.0)")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}
		version, ok := argsMap["version"].(string)
		if !ok {
			return mcp.NewToolResultError("Argumento 'version' requerido"), nil
		}
		
		cwd, err := os.Getwd()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		runReleaseBuilds(cwd)
		updateReleaseVersion(version)
		tagRelease(version)

		return mcp.NewToolResultText(fmt.Sprintf("Release %s finalizado.", version)), nil
	})
}

func runReleaseBuilds(cwd string) {
	if fileExists(filepath.Join(cwd, "package.json")) {
		exec.Command("npm", "run", "build").Run()
	}
	
	if !fileExists(filepath.Join(cwd, "package.json")) && fileExists(filepath.Join(cwd, "Makefile")) {
		exec.Command("make", "build").Run()
	}
}

func updateReleaseVersion(version string) {
	qddDir := filepath.Join(".", ".qdd")
	statePath := filepath.Join(qddDir, "state.json")
	stateData, err := os.ReadFile(statePath)
	if err == nil {
		var state map[string]interface{}
		json.Unmarshal(stateData, &state)
		state["version"] = version
		newData, _ := json.MarshalIndent(state, "", "  ")
		os.WriteFile(statePath, newData, 0644)
	}
}

func tagRelease(version string) {
	execCmd := exec.Command("git", "tag", version)
	execCmd.Run()
}
