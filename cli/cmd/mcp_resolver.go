package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/findings"
	"gopkg.in/yaml.v3"
)

type DiagnosisPayload struct {
	ID             string   `yaml:"id"`
	IssueType      string   `yaml:"issue_type"`
	Title          string   `yaml:"title"`
	RootCause      string   `yaml:"root_cause"`
	ImpactAnalysis string   `yaml:"impact_analysis"`
	AffectedFiles  []string `yaml:"affected_files"`
	DiagnosedAt    string   `yaml:"diagnosed_at"`
	Status         string   `yaml:"status"`
}

func registerResolveDiagnoseTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_resolve_diagnose",
		mcp.WithDescription("Registra el diagnóstico del Analista (Equipo Resolutor)."),
		mcp.WithString("issue_type", mcp.Required(), mcp.Description("bug | refactor | feature | security")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Título corto")),
		mcp.WithString("root_cause", mcp.Required(), mcp.Description("Causa raíz")),
		mcp.WithString("impact_analysis", mcp.Required(), mcp.Description("Impacto en dependencias vecinas")),
		mcp.WithString("affected_files", mcp.Required(), mcp.Description("Archivos afectados, por ;")),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}

		issueType, _ := argsMap["issue_type"].(string)
		title, _ := argsMap["title"].(string)
		rootCause, _ := argsMap["root_cause"].(string)
		impactAnalysis, _ := argsMap["impact_analysis"].(string)
		affectedFilesRaw, _ := argsMap["affected_files"].(string)

		if issueType == "" || title == "" || rootCause == "" || impactAnalysis == "" {
			return mcp.NewToolResultError("Todos los campos (issue_type, title, root_cause, impact_analysis, affected_files) son obligatorios para el diagnóstico FAANG."), nil
		}

		var filesList []string
		for _, f := range strings.Split(affectedFilesRaw, ";") {
			trimmed := strings.TrimSpace(f)
			if trimmed != "" {
				filesList = append(filesList, trimmed)
			}
		}

		findingID := fmt.Sprintf("FIND-DIAG-%d", time.Now().Unix())
		payload := DiagnosisPayload{
			ID:             findingID,
			IssueType:      issueType,
			Title:          title,
			RootCause:      rootCause,
			ImpactAnalysis: impactAnalysis,
			AffectedFiles:  filesList,
			DiagnosedAt:    time.Now().Format(time.RFC3339),
			Status:         string(findings.Abierto),
		}

		cwd, _ := os.Getwd()
		findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
		os.MkdirAll(findingsDir, 0755)

		targetFile := filepath.Join(findingsDir, fmt.Sprintf("%s.yaml", findingID))

		data, err := yaml.Marshal(payload)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error serializando diagnóstico: %v", err)), nil
		}

		if err := os.WriteFile(targetFile, data, 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error guardando diagnóstico: %v", err)), nil
		}

		msg := fmt.Sprintf("Diagnóstico registrado como finding abierto.\nID: %s\nRuta: %s\n\nFalta la prueba que lo reproduce y el arreglo.", findingID, targetFile)
		return mcp.NewToolResultText(msg), nil
	})
}

var idDeResolucion = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

// goldenSetDelProyecto: el acta nombra un archivo que existe dentro del proyecto.
func goldenSetDelProyecto(cwd, ruta string) error {
	completa := ruta
	if !filepath.IsAbs(ruta) {
		completa = filepath.Join(cwd, ruta)
	}
	rel, err := filepath.Rel(cwd, completa)
	if err != nil || !filepath.IsLocal(rel) {
		return fmt.Errorf("el golden set %s está fuera del proyecto", ruta)
	}
	if !fileExists(completa) {
		return fmt.Errorf("el golden set %s no existe", ruta)
	}
	return nil
}

func registerResolveCertifyTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_resolve_certify",
		mcp.WithDescription("Escribe el acta de una resolución; no corre pruebas."),
		mcp.WithString("resolution_id", mcp.Required(), mcp.Description("Id de la resolución")),
		mcp.WithString("golden_set_path", mcp.Required(), mcp.Description("Ruta del Golden Set")),
		mcp.WithBoolean("qa_approved", mcp.Required(), mcp.Description("Aprobación de QA")),
		mcp.WithString("qa_notes", mcp.Required(), mcp.Description("Notas de QA (zero-mocks, bordes)")),
		mcp.WithBoolean("devops_approved", mcp.Required(), mcp.Description("Aprobación de DevOps/SRE")),
		mcp.WithString("devops_notes", mcp.Required(), mcp.Description("Notas de despliegue (CI/CD, impacto)")),
	)

	RegisterToolWithTelemetry(s, tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("Argumentos inválidos"), nil
		}

		resolutionID, _ := argsMap["resolution_id"].(string)
		goldenSetPath, _ := argsMap["golden_set_path"].(string)
		qaApproved, _ := argsMap["qa_approved"].(bool)
		qaNotes, _ := argsMap["qa_notes"].(string)
		devopsApproved, _ := argsMap["devops_approved"].(bool)
		devopsNotes, _ := argsMap["devops_notes"].(string)

		if resolutionID == "" || goldenSetPath == "" {
			return mcp.NewToolResultError("resolution_id y golden_set_path son requeridos"), nil
		}

		if !idDeResolucion.MatchString(resolutionID) {
			return mcp.NewToolResultError("resolution_id solo lleva letras, números, «-» y «_»"), nil
		}

		cwd, err := os.Getwd()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if err := goldenSetDelProyecto(cwd, goldenSetPath); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if !qaApproved {
			return mcp.NewToolResultError("Certificación RECHAZADA: QA no ha dado su aprobación. Debe regresar a Fase 3 (Implementador)."), nil
		}

		if !devopsApproved {
			return mcp.NewToolResultError("Certificación RECHAZADA: DevOps no ha aprobado el plan de despliegue sin impacto."), nil
		}

		certDir := filepath.Join(cwd, ".qdd", "project", "certification")
		os.MkdirAll(certDir, 0755)

		certFile := filepath.Join(certDir, fmt.Sprintf("resolution_%s.md", resolutionID))
		content := fmt.Sprintf(`# Acta de resolución %s

- **Fecha:** %s
- **Golden set:** %s (el archivo existe; no se ejecutó)
- **Aprobaciones:** las declaró quien llamó a la tool; QDD no las comprueba

## QA
- **Declarado:** aprobado
- **Notas:** %s

## DevOps / SRE
- **Declarado:** aprobado
- **Notas:** %s
`, resolutionID, time.Now().Format(time.RFC3339), goldenSetPath, qaNotes, devopsNotes)

		if err := os.WriteFile(certFile, []byte(content), 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error guardando certificado: %v", err)), nil
		}

		msg := fmt.Sprintf("Acta escrita en %s. Registra lo declarado; el golden set no se ejecutó: corre qdd audit antes de commitear.", certFile)
		return mcp.NewToolResultText(msg), nil
	})
}
