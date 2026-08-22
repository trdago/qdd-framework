package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/cognitive"
	"github.com/qdd-framework/qdd/pkg/evolution"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/spf13/cobra"
)

var (
	reportFormat string
	reportOutput string
)

type ComprehensiveReport struct {
	Timestamp          string                `json:"timestamp"`
	ProjectName        string                `json:"project_name"`
	ProjectRoot        string                `json:"project_root"`
	Score              int                   `json:"score"`
	Grade              string                `json:"grade"`
	Tendency           string                `json:"tendency"`
	ViolationsCount    int                   `json:"violations_count"`
	Violations         []audit.Violation     `json:"violations"`
	TotalCertifications int                  `json:"total_certifications"`
	CertifiedCount     int                   `json:"certified_count"`
	PendingCount       int                   `json:"pending_count"`
	OpenFindingsCount  int                   `json:"open_findings_count"`
	ResolvedFindings   int                   `json:"resolved_findings_count"`
	AIGovernanceStatus string                `json:"ai_governance_status"`
	DoctorHealthy      bool                  `json:"doctor_healthy"`
	DoctorDetails      []string              `json:"doctor_details"`
	LocalLLM           cognitive.LocalLLMInfo `json:"local_llm"`
	EvolutionPriority  string                `json:"evolution_priority"`
	EvolutionAdvice    string                `json:"evolution_advice"`
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Genera el reporte integral y consolidado de calidad, auditoría y gobernanza",
	Long: `Consolida en un único reporte determinista el estado de auditoría (9 pilares),
score de calidad, certificaciones, findings, salud de doctor, gobernanza de IA y evolución.
Soporta formatos text, markdown (md) y JSON para integración con CI/CD.`,
	Run: runReport,
}

func init() {
	reportCmd.Flags().StringVar(&reportFormat, "format", "text", "Formato de salida: text, md, json")
	reportCmd.Flags().StringVar(&reportOutput, "output", "", "Ruta de archivo para guardar el reporte (opcional)")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}
	cwd = integration.FindProjectRoot(cwd)

	report := GenerateComprehensiveReport(cwd)

	outputContent := formatReportOutput(report, reportFormat)
	fmt.Print(outputContent)

	handleReportFileSaving(cwd, outputContent, reportOutput, reportFormat)
}

func GenerateComprehensiveReport(cwd string) ComprehensiveReport {
	engine := audit.NewEngine(cwd)
	violations := engine.RunAll()

	evoReport, _ := evolution.Analyze(cwd, len(violations))

	score, grade, tendency := getScoreMetadata(cwd, evoReport)

	totalCerts, certifiedCerts, pendingCerts := countProjectCertifications(cwd)
	openFindings, resolvedFindings := countProjectFindings(cwd)
	doctorHealthy, doctorDetails := runDoctorQuickCheck(cwd)
	localLLM := cognitive.DetectLocalLLM()

	aiStatus := "CERTIFIED"
	if len(filterAIViolations(violations)) > 0 {
		aiStatus = "VIOLATIONS_DETECTED"
	}

	evoPriority := "LOW"
	evoAdvice := "Proyecto estable y sin deuda técnica detectada."
	if evoReport != nil {
		evoPriority = evoReport.Priority
		evoAdvice = evoReport.Recommendation
	}

	projectName := filepath.Base(cwd)
	return ComprehensiveReport{
		Timestamp:           time.Now().Format(time.RFC3339),
		ProjectName:         projectName,
		ProjectRoot:         cwd,
		Score:               score,
		Grade:               grade,
		Tendency:            tendency,
		ViolationsCount:     len(violations),
		Violations:          violations,
		TotalCertifications: totalCerts,
		CertifiedCount:      certifiedCerts,
		PendingCount:        pendingCerts,
		OpenFindingsCount:   openFindings,
		ResolvedFindings:    resolvedFindings,
		AIGovernanceStatus:  aiStatus,
		DoctorHealthy:       doctorHealthy,
		DoctorDetails:       doctorDetails,
		LocalLLM:           localLLM,
		EvolutionPriority:   evoPriority,
		EvolutionAdvice:     evoAdvice,
	}
}

func getScoreMetadata(cwd string, evoReport *evolution.Report) (int, string, string) {
	if evoReport != nil {
		score := evoReport.Score
		grade := determineGrade(score)
		return score, grade, evoReport.Tendency
	}
	return 100, "A+ (Excelente)", "Estable"
}

func filterAIViolations(violations []audit.Violation) []audit.Violation {
	var aiViolations []audit.Violation
	for _, v := range violations {
		if v.Category == "ISO/IEC 42001 (AIMS)" ||
			v.Category == "ISO/IEC 29119-11 (AI Testing)" ||
			v.Category == "SOC 2 (AI Processing Integrity)" ||
			v.Category == "ISO/IEC 23894 (AI Risk Management)" {
			aiViolations = append(aiViolations, v)
		}
	}
	return aiViolations
}

func countProjectCertifications(cwd string) (int, int, int) {
	certDir := filepath.Join(cwd, ".qdd", "project", "certification")
	entries, err := os.ReadDir(certDir)
	if err != nil {
		return 0, 0, 0
	}

	total, certified, pending := 0, 0, 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		total++
		content, err := os.ReadFile(filepath.Join(certDir, entry.Name()))
		if err != nil {
			continue
		}
		if containsStringFold(string(content), "status: certified") {
			certified++
		}
		if containsStringFold(string(content), "status: pending") {
			pending++
		}
	}
	return total, certified, pending
}

func countProjectFindings(cwd string) (int, int) {
	findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
	entries, err := os.ReadDir(findingsDir)
	if err != nil {
		return 0, 0
	}

	open, resolved := 0, 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(findingsDir, entry.Name()))
		if err != nil {
			continue
		}
		text := string(content)
		if containsStringFold(text, "status: resolved") || containsStringFold(text, "status: closed") {
			resolved++
		}
		if !containsStringFold(text, "status: resolved") && !containsStringFold(text, "status: closed") {
			open++
		}
	}
	return open, resolved
}

func runDoctorQuickCheck(cwd string) (bool, []string) {
	var details []string
	healthy := true

	qddDir := filepath.Join(cwd, ".qdd")
	if _, err := os.Stat(qddDir); err != nil {
		details = append(details, "❌ Carpeta .qdd no encontrada")
		return false, details
	}
	details = append(details, "✅ Entorno de gobernanza .qdd inicializado")

	configPath := filepath.Join(qddDir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		details = append(details, "❌ Archivo .qdd/config.yaml faltante")
		healthy = false
	}
	if healthy {
		details = append(details, "✅ Configuración y políticas activas")
	}

	return healthy, details
}

func formatReportOutput(r ComprehensiveReport, format string) string {
	if format == "json" {
		data, _ := json.MarshalIndent(r, "", "  ")
		return string(data) + "\n"
	}

	if format == "md" {
		return formatMarkdownReport(r)
	}

	return formatTextReport(r)
}

func formatTextReport(r ComprehensiveReport) string {
	out := "================================================================================\n"
	out += fmt.Sprintf("                      REPORTE INTEGRAL QDD FRAMEWORK\n")
	out += fmt.Sprintf("                 \"Lo que no se mide no se mejora\"\n")
	out += "================================================================================\n"
	out += fmt.Sprintf("Proyecto:       %s\n", r.ProjectName)
	out += fmt.Sprintf("Fecha:          %s\n", r.Timestamp)
	out += fmt.Sprintf("Score Global:   %d / 100 (%s) | Tendencia: %s\n", r.Score, r.Grade, r.Tendency)
	out += "--------------------------------------------------------------------------------\n"
	out += fmt.Sprintf("Auditoría:      %d violaciones activas\n", r.ViolationsCount)
	out += fmt.Sprintf("Certificaciones: %d adoptadas (%d certificadas, %d pendientes)\n", r.TotalCertifications, r.CertifiedCount, r.PendingCount)
	out += fmt.Sprintf("Findings:       %d abiertos / %d resueltos\n", r.OpenFindingsCount, r.ResolvedFindings)
	out += fmt.Sprintf("Gobernanza IA:  %s (ISO 42001, ISO 29119-11, SOC 2, ISO 23894)\n", r.AIGovernanceStatus)
	out += fmt.Sprintf("Doctor:         %s\n", formatDoctorBool(r.DoctorHealthy))
	out += fmt.Sprintf("Motor LLM Local: [%s] %s (%s)\n", r.LocalLLM.Status, r.LocalLLM.Provider, r.LocalLLM.Endpoint)
	out += "--------------------------------------------------------------------------------\n"
	out += fmt.Sprintf("Evolución [%s]: %s\n", r.EvolutionPriority, r.EvolutionAdvice)
	out += "================================================================================\n"
	return out
}

func formatMarkdownReport(r ComprehensiveReport) string {
	out := "# Reporte Integral de Calidad y Gobernanza QDD\n\n"
	out += "> *\"Lo que no se mide no se mejora.\"*\n\n"
	out += fmt.Sprintf("**Proyecto:** `%s`  \n", r.ProjectName)
	out += fmt.Sprintf("**Fecha de Auditoría:** `%s`  \n", r.Timestamp)
	out += fmt.Sprintf("**Score de Calidad:** `%d / 100` (%s)  \n", r.Score, r.Grade)
	out += fmt.Sprintf("**Tendencia:** `%s`\n\n", r.Tendency)

	out += "## Resumen Ejecutivo de Métricas\n\n"
	out += "| Subsistema | Estado / Métrica | Observación |\n"
	out += "| :--- | :--- | :--- |\n"
	out += fmt.Sprintf("| **Auditoría Técnica** | `%d violaciones` | 9 Pilares de Calidad QDD |\n", r.ViolationsCount)
	out += fmt.Sprintf("| **Certificaciones de Proyecto** | `%d / %d certificadas` | %d pendientes |\n", r.CertifiedCount, r.TotalCertifications, r.PendingCount)
	out += fmt.Sprintf("| **Hallazgos (Findings)** | `%d abiertos / %d resueltos` | Ciclo de Aprendizaje Perpetuo |\n", r.OpenFindingsCount, r.ResolvedFindings)
	out += fmt.Sprintf("| **Gobernanza de IA** | `%s` | ISO 42001, ISO 29119-11, SOC 2, ISO 23894 |\n", r.AIGovernanceStatus)
	out += fmt.Sprintf("| **Salud del Framework (Doctor)** | `%s` | Auto-reparación y estructura |\n", formatDoctorBool(r.DoctorHealthy))
	out += fmt.Sprintf("| **Poder Cognitivo Local (LLM)** | `[%s] %s` | Endpoint: `%s` |\n\n", r.LocalLLM.Status, r.LocalLLM.Provider, r.LocalLLM.Endpoint)

	out += "## Recomendación de Evolución\n\n"
	out += fmt.Sprintf("> [!TIP]\n> **Prioridad: %s**  \n> %s\n", r.EvolutionPriority, r.EvolutionAdvice)
	return out
}

func formatDoctorBool(healthy bool) string {
	if healthy {
		return "HEALTHY (Saludable)"
	}
	return "DEGRADED (Requiere Atención)"
}

func handleReportFileSaving(cwd, content, outputPath, format string) {
	if outputPath != "" {
		_ = os.WriteFile(outputPath, []byte(content), 0644)
		return
	}

	reportsDir := filepath.Join(cwd, ".qdd", "project", "evidence", "reports")
	_ = os.MkdirAll(reportsDir, 0755)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	ext := "txt"
	if format == "json" {
		ext = "json"
	}
	if format == "md" {
		ext = "md"
	}
	savePath := filepath.Join(reportsDir, fmt.Sprintf("report_%s.%s", timestamp, ext))
	_ = os.WriteFile(savePath, []byte(content), 0644)
}

func containsStringFold(s, substr string) bool {
	return containsFoldHelper(s, substr)
}

func containsFoldHelper(s, substr string) bool {
	sLower := []rune(s)
	subLower := []rune(substr)
	if len(subLower) == 0 {
		return true
	}
	if len(sLower) < len(subLower) {
		return false
	}
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		match := true
		for j := 0; j < len(subLower); j++ {
			c1 := sLower[i+j]
			c2 := subLower[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
