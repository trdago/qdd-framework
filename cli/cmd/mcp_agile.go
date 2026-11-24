package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/findings"
	"github.com/qdd-framework/qdd/pkg/gitenv"
	"github.com/qdd-framework/qdd/pkg/integration"
)

func registerFindingsTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_findings",
		mcp.WithDescription("Findings del proyecto (bugs, vulnerabilidades)."),
	)
	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if _, err := os.Stat(findings.Dir(".")); os.IsNotExist(err) {
			return mcp.NewToolResultText("No se encontró el directorio de findings (.qdd/project/findings)."), nil
		}
		fs, err := findings.Leer(".")
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error leyendo directorio de findings: %v", err)), nil
		}
		if len(fs) == 0 {
			return mcp.NewToolResultText("No hay hallazgos registrados en el proyecto."), nil
		}
		return mcp.NewToolResultText(reporteFindings(fs)), nil
	})
}

// iconoFinding: cada estado de pkg/findings con el suyo; un sin estado no es abierto ni resuelto (FND-261).
var iconoFinding = map[findings.Estado]string{
	findings.Abierto: "🔴", findings.Resuelto: "🟢", findings.Descartado: "⚪", findings.SinEstado: "❔",
}

// reporteFindings cuenta con pkg/findings, como los demás lectores (FND-261).
func reporteFindings(fs []findings.Finding) string {
	out := "--- QDD FINDINGS REPORT ---\n\n"
	for _, f := range fs {
		estado := strings.ToUpper(f.EstadoOriginal)
		if f.Estado == findings.SinEstado {
			estado = "SIN ESTADO"
		}
		out += fmt.Sprintf("%s [%s] [%s] %s - %s\n", iconoFinding[f.Estado], f.ID, strings.ToUpper(f.Severidad), estado, f.Titulo)
	}
	c := findings.Contar(fs)
	out += "\n-------------------------------\n"
	out += fmt.Sprintf("Total: %d | Resueltos: %d | Pendientes: %d | Descartados: %d | Sin estado: %d\n", len(fs), c.Resueltos, c.Abiertos, c.Descartados, c.SinEstado)
	return out
}

func registerSprintTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_sprint",
		mcp.WithDescription("Crea un sprint (QCS v1.1): exige al usuario Happy Path y Error Paths (Edge Cases opcionales); los tests NO se los preguntes, dedúcelos tú del stack."),
		mcp.WithString("numero", mcp.Required(), mcp.Description("Número del sprint (1, 2, 3…)")),
		mcp.WithString("goal", mcp.Required(), mcp.Description("Objetivo, sin ambigüedad")),
		mcp.WithString("happy_path", mcp.Required(), mcp.Description("Camino feliz: Dado/Cuando/Entonces, por ;")),
		mcp.WithString("error_paths", mcp.Required(), mcp.Description("Escenarios de error, por ;")),
		mcp.WithString("edge_cases", mcp.Description("Opcionales (timeouts, concurrencia), por ;")),
		mcp.WithString("technical_constraints", mcp.Required(), mcp.Description("Restricciones técnicas, por ;")),
		mcp.WithString("executable_tests", mcp.Required(), mcp.Description("Comandos de test que generas TÚ (no el usuario), por ;")),
	)
	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		"numero":                getString("numero"),
		"goal":                  getString("goal"),
		"happy_path":            getString("happy_path"),
		"error_paths":           getString("error_paths"),
		"edge_cases":            getString("edge_cases"),
		"technical_constraints": getString("technical_constraints"),
		"executable_tests":      getString("executable_tests"),
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
		mcp.WithDescription("Sincroniza las reglas de QDD con los asistentes de IA."),
	)
	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		mcp.WithDescription("Corre el gate, compila y crea un tag local."),
		mcp.WithString("version", mcp.Required(), mcp.Description("Versión a liberar (ej: v1.0.0)")),
	)
	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		cwd = integration.FindProjectRoot(cwd)
		if err := publicarRelease(cwd, version, violacionesDelGate, compilarRelease); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Release %s: gate aprobado y tag local creado. Commitea .qdd/state.json; publicar es de la persona.", version)), nil
	})
}

func violacionesDelGate(cwd string) []audit.Violation {
	return audit.NewEngine(cwd).ParandoAlBloquear().Evaluar().Violaciones
}

// publicarRelease etiqueta solo lo que el gate aprueba y compila (FND-310). El tag va antes que state.json: si ya existe,
// la versión anotada no cambia.
func publicarRelease(cwd, version string, gate func(string) []audit.Violation, compilar func(string) error) error {
	if bloqueantes := audit.Bloqueantes(gate(cwd)); len(bloqueantes) > 0 {
		return fmt.Errorf("el gate bloquea la release %s, no se creó el tag:\n%s", version, listarViolaciones(bloqueantes))
	}
	if err := compilar(cwd); err != nil {
		return fmt.Errorf("no se creó el tag %s: %w", version, err)
	}
	if err := correrEn(cwd, "git", "tag", "--", version); err != nil {
		return fmt.Errorf("no se creó el tag %s: %w", version, err)
	}
	return updateReleaseVersion(cwd, version)
}

func compilarRelease(cwd string) error {
	comando := comandoDeCompilar(cwd)
	if len(comando) == 0 {
		return nil
	}
	return correrEn(cwd, comando[0], comando[1:]...)
}

func comandoDeCompilar(cwd string) []string {
	if fileExists(filepath.Join(cwd, "package.json")) {
		return []string{"npm", "run", "build", "--if-present"}
	}
	if fileExists(filepath.Join(cwd, "Makefile")) {
		return []string{"make", "build"}
	}
	return nil
}

func correrEn(cwd, nombre string, args ...string) error {
	cmd := exec.Command(nombre, args...)
	cmd.Dir, cmd.Env = cwd, gitenv.Limpio(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", nombre, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func updateReleaseVersion(cwd, version string) error {
	statePath := filepath.Join(cwd, ".qdd", "state.json")
	stateData, err := os.ReadFile(statePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	state := map[string]interface{}{}
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("el tag %s se creó, pero .qdd/state.json no se pudo leer: %w", version, err)
	}
	state["version"] = version
	newData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, newData, 0644)
}
