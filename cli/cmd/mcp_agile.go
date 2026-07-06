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

		out := "--- QDD FINDINGS REPORT ---\n\n"
		total := 0
		resolved := 0
		pending := 0

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			path := filepath.Join(qddDir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var f Finding // Definiendo 'Finding' de findings.go
			if err := yaml.Unmarshal(content, &f); err != nil {
				continue
			}

			total++
			icon := "🔴"
			statusLower := strings.ToLower(f.Status)
			
			if statusLower == "resolved" || statusLower == "closed" {
				icon = "🟢"
				resolved++
				out += fmt.Sprintf("%s [%s] [%s] %s - %s\n", icon, f.ID, strings.ToUpper(f.Severity), strings.ToUpper(f.Status), f.Title)
				continue
			}

			pending++
			out += fmt.Sprintf("%s [%s] [%s] %s - %s\n", icon, f.ID, strings.ToUpper(f.Severity), strings.ToUpper(f.Status), f.Title)
		}

		if total == 0 {
			return mcp.NewToolResultText("No hay hallazgos registrados en el proyecto."), nil
		}

		out += "\n-------------------------------\n"
		out += fmt.Sprintf("Total: %d | Resueltos: %d | Pendientes: %d\n", total, resolved, pending)
		return mcp.NewToolResultText(out), nil
	})
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
		
		getString := func(key string) string {
			if val, ok := argsMap[key].(string); ok {
				return val
			}
			return ""
		}

		sprintNum := getString("numero")
		goal := getString("goal")
		happyPath := getString("happy_path")
		errorPaths := getString("error_paths")
		edgeCases := getString("edge_cases")
		technicalConstraints := getString("technical_constraints")
		executableTests := getString("executable_tests")

		if sprintNum == "" || goal == "" || happyPath == "" || errorPaths == "" || executableTests == "" {
			return mcp.NewToolResultError("Faltan parámetros obligatorios (numero, goal, happy_path, error_paths, executable_tests). Vuelve a preguntar al usuario por los escenarios faltantes, pero recuerda generar TÚ MISMO los executable_tests."), nil
		}
		
		qddDir := filepath.Join(".", ".qdd")
		sprintsDir := filepath.Join(qddDir, "project", "sprints")
		os.MkdirAll(sprintsDir, 0755)
		
		sprintFile := filepath.Join(sprintsDir, fmt.Sprintf("sprint-%s.md", sprintNum))
		certFile := filepath.Join(sprintsDir, fmt.Sprintf("sprint-%s-cert.yaml", sprintNum))
		
		if _, err := os.Stat(certFile); !os.IsNotExist(err) {
			return mcp.NewToolResultText(fmt.Sprintf("El sprint/certificado %s ya existe.", sprintNum)), nil
		}

		// Crear el YAML de certificación
		certContent := fmt.Sprintf(`qdd_certificate:
  version: "1.1"
  sprint_id: "%s"
  status: "FAILING"
  goal: "%s"
  
  scenarios:
    happy_path:`, sprintNum, goal)
        
		for _, rule := range strings.Split(happyPath, ";") {
			rule = strings.TrimSpace(rule)
			if rule != "" {
				certContent += fmt.Sprintf("\n      - \"%s\"", rule)
			}
		}

		certContent += "\n    error_paths:"
		for _, rule := range strings.Split(errorPaths, ";") {
			rule = strings.TrimSpace(rule)
			if rule != "" {
				certContent += fmt.Sprintf("\n      - \"%s\"", rule)
			}
		}

		certContent += "\n    edge_cases:"
		if edgeCases != "" {
			for _, rule := range strings.Split(edgeCases, ";") {
				rule = strings.TrimSpace(rule)
				if rule != "" {
					certContent += fmt.Sprintf("\n      - \"%s\"", rule)
				}
			}
		}
        
		certContent += "\n\n  technical_constraints:"
		for _, rule := range strings.Split(technicalConstraints, ";") {
			rule = strings.TrimSpace(rule)
			if rule != "" {
				certContent += fmt.Sprintf("\n    - \"%s\"", rule)
			}
		}
        
		certContent += "\n\n  executable_tests:"
		for _, cmd := range strings.Split(executableTests, ";") {
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				certContent += fmt.Sprintf("\n    - \"%s\"", cmd)
			}
		}
		certContent += "\n"
        
		os.WriteFile(certFile, []byte(certContent), 0644)

		content := fmt.Sprintf("# Sprint %s\n\n## Objetivos (Sprint Goal)\n%s\n\n## Certification-Driven Development (CDD v1.1)\nEste sprint se rige por un contrato estricto de escenarios exhaustivos.\nEl agente debe implementar el código y los tests para el **Happy Path**, **Error Paths** y **Edge Cases** definidos en `sprint-%s-cert.yaml`.\n\n---\n*Gobernanza QDD: Este archivo y su YAML asociado guían el desarrollo. Usa 'qdd certify' o ejecuta los tests directamente.*\n", sprintNum, goal, sprintNum)
		os.WriteFile(sprintFile, []byte(content), 0644)

		return mcp.NewToolResultText(fmt.Sprintf("Sprint %s inicializado correctamente en modo CDD v1.1.\nContrato generado en: %s\nDocumento generado en: %s\n\nEl Agente IA DEBE ahora leer el archivo YAML y comenzar el bucle autónomo programando cada escenario listado hasta que los 'executable_tests' pasen sin errores.", sprintNum, certFile, sprintFile)), nil
	})
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

		if fileExists(filepath.Join(cwd, "package.json")) {
			exec.Command("npm", "run", "build").Run()
		}
		
		if !fileExists(filepath.Join(cwd, "package.json")) && fileExists(filepath.Join(cwd, "Makefile")) {
			exec.Command("make", "build").Run()
		}

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

		execCmd := exec.Command("git", "tag", version)
		execCmd.Run()

		return mcp.NewToolResultText(fmt.Sprintf("Release %s finalizado.", version)), nil
	})
}
