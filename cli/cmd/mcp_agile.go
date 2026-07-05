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
		mcp.WithDescription("Inicializa un nuevo Sprint de trabajo"),
		mcp.WithString("numero", mcp.Required(), mcp.Description("Número del sprint a crear (ej: 1, 2, 3)")),
	)
	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}
		sprintNum, ok := argsMap["numero"].(string)
		if !ok {
			return mcp.NewToolResultError("Argumento 'numero' es requerido como string"), nil
		}
		
		qddDir := filepath.Join(".", ".qdd")
		sprintsDir := filepath.Join(qddDir, "project", "sprints")
		os.MkdirAll(sprintsDir, 0755)
		sprintFile := filepath.Join(sprintsDir, fmt.Sprintf("sprint-%s.md", sprintNum))
		
		if _, err := os.Stat(sprintFile); !os.IsNotExist(err) {
			return mcp.NewToolResultText(fmt.Sprintf("El sprint %s ya existe.", sprintNum)), nil
		}

		content := fmt.Sprintf("# Sprint %s\n\n## Objetivos (Sprint Goal)\n- [ ] Definir objetivos aquí\n\n## Tareas (Backlog)\n- [ ] Tarea 1\n- [ ] Tarea 2\n\n## Métricas de Calidad Iniciales\n- **QDD Score de Entrada:** (Ejecuta 'qdd score')\n- **Deuda Técnica Inicial:** (Ejecuta 'qdd status')\n\n---\n*Gobernanza QDD: Todo código añadido en este sprint debe contar con evidencia (EV-FND) y pruebas unitarias.*\n", sprintNum)
		os.WriteFile(sprintFile, []byte(content), 0644)

		return mcp.NewToolResultText(fmt.Sprintf("Archivo de Sprint creado exitosamente en: %s", sprintFile)), nil
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
