package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
)

type DiagnosisPayload struct {
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
		mcp.WithDescription("Registra el diagnóstico formal del Analista Integral en el Equipo Resolutor."),
		mcp.WithString("issue_type", mcp.Required(), mcp.Description("Tipo de problema: bug, refactor, feature, security")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Título conciso del problema diagnosticado")),
		mcp.WithString("root_cause", mcp.Required(), mcp.Description("Causa raíz identificada tras el análisis forense")),
		mcp.WithString("impact_analysis", mcp.Required(), mcp.Description("Análisis de impacto en dependencias adyacentes")),
		mcp.WithString("affected_files", mcp.Required(), mcp.Description("Rutas de archivos afectados separadas por punto y coma (;)")),
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

		payload := DiagnosisPayload{
			IssueType:      issueType,
			Title:          title,
			RootCause:      rootCause,
			ImpactAnalysis: impactAnalysis,
			AffectedFiles:  filesList,
			DiagnosedAt:    time.Now().Format(time.RFC3339),
			Status:         "diagnosed",
		}

		cwd, _ := os.Getwd()
		findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
		os.MkdirAll(findingsDir, 0755)

		findingID := fmt.Sprintf("FIND-DIAG-%d", time.Now().Unix())
		targetFile := filepath.Join(findingsDir, fmt.Sprintf("%s.yaml", findingID))

		data, err := yaml.Marshal(payload)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error serializando diagnóstico: %v", err)), nil
		}

		if err := os.WriteFile(targetFile, data, 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error guardando diagnóstico: %v", err)), nil
		}

		msg := fmt.Sprintf("✅ Diagnóstico registrado exitosamente.\nID: %s\nRuta: %s\n\nEl testigo pasa al Implementador para crear el Golden Set (Fase 3) y la solución Cero-Else (Fase 4).", findingID, targetFile)
		return mcp.NewToolResultText(msg), nil
	})
}

func registerResolveCertifyTool(s *server.MCPServer) {
	tool := mcp.NewTool("qdd_resolve_certify",
		mcp.WithDescription("Firma y certifica la resolución de un problema tras la validación de QA y DevOps."),
		mcp.WithString("resolution_id", mcp.Required(), mcp.Description("ID o nombre de la resolución")),
		mcp.WithString("golden_set_path", mcp.Required(), mcp.Description("Ruta al archivo Golden Set de prueba unitaria")),
		mcp.WithBoolean("qa_approved", mcp.Required(), mcp.Description("Aprobación explícita de QA")),
		mcp.WithString("qa_notes", mcp.Required(), mcp.Description("Notas de auditoría de QA (Zero-Mocks, casos de borde)")),
		mcp.WithBoolean("devops_approved", mcp.Required(), mcp.Description("Aprobación explícita de DevOps/SRE")),
		mcp.WithString("devops_notes", mcp.Required(), mcp.Description("Notas de despliegue de DevOps (CI/CD, impacto)")),
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

		if !qaApproved {
			return mcp.NewToolResultError("Certificación RECHAZADA: QA no ha dado su aprobación. Debe regresar a Fase 3 (Implementador)."), nil
		}

		if !devopsApproved {
			return mcp.NewToolResultError("Certificación RECHAZADA: DevOps no ha aprobado el plan de despliegue sin impacto."), nil
		}

		cwd, _ := os.Getwd()
		certDir := filepath.Join(cwd, ".qdd", "project", "certification")
		os.MkdirAll(certDir, 0755)

		certFile := filepath.Join(certDir, fmt.Sprintf("resolution_%s.md", resolutionID))
		content := fmt.Sprintf(`# CERTIFICADO DE RESOLUCIÓN %s

- **Fecha de Certificación:** %s
- **Golden Set Verificado:** %s
- **Estado:** APROBADO 100%% (Cero-Else, Zero-Mocks en Prod)

## Validación QA
- **Aprobado:** Sí
- **Notas de QA:** %s

## Validación DevOps / SRE
- **Aprobado:** Sí
- **Notas de DevOps:** %s
`, resolutionID, time.Now().Format(time.RFC3339), goldenSetPath, qaNotes, devopsNotes)

		if err := os.WriteFile(certFile, []byte(content), 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error guardando certificado: %v", err)), nil
		}

		msg := fmt.Sprintf("🎉 RESOLUCIÓN CERTIFICADA Y APROBADA.\nCertificado generado en: %s\nEl cambio está certificado para release a producción.", certFile)
		return mcp.NewToolResultText(msg), nil
	})
}
