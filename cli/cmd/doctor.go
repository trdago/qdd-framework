package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verifica que el entorno QDD esté completamente funcional",
	Long:  `Ejecuta pruebas deterministas para asegurar que la estructura y las integraciones del framework QDD están operando correctamente, dejando evidencia auditable.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
			os.Exit(1)
		}

		success := RunDoctorCheck(cwd)
		if !success {
			fmt.Println("[-] QDD Doctor detectó anomalías. Ejecuta `qdd init` para intentar corregirlas automáticamente.")
			os.Exit(1)
		}

		fmt.Println("[+] QDD Doctor: Todo el entorno se encuentra 100% operativo.")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// RunDoctorCheck ejecuta las pruebas deterministas del framework y deja un reporte.
func RunDoctorCheck(projectPath string) bool {
	checks := []string{}
	failedChecks := []string{}

	// 1. Integridad Estructural Básica
	qddDir := filepath.Join(projectPath, ".qdd")
	if info, err := os.Stat(qddDir); os.IsNotExist(err) || !info.IsDir() {
		failedChecks = append(failedChecks, "[Fallo] Directorio raíz .qdd/ no encontrado")
	}
	if info, err := os.Stat(qddDir); err == nil && info.IsDir() {
		checks = append(checks, "[OK] Directorio raíz .qdd/ verificado")
	}

	configPath := filepath.Join(qddDir, "config.yaml")
	if !fileExists(configPath) {
		failedChecks = append(failedChecks, "[Fallo] Archivo config.yaml no encontrado")
	}
	if fileExists(configPath) {
		checks = append(checks, "[OK] Archivo config.yaml verificado")
	}

	statePath := filepath.Join(qddDir, "state.json")
	if !fileExists(statePath) {
		failedChecks = append(failedChecks, "[Fallo] Archivo state.json no encontrado")
	}
	if fileExists(statePath) {
		checks = append(checks, "[OK] Archivo state.json verificado")
	}

	// 2. Integraciones de IA
	cursorMCP := filepath.Join(projectPath, ".cursor", "mcp.json")
	if !fileExists(cursorMCP) {
		failedChecks = append(failedChecks, "[Fallo] Configuración MCP (.cursor/mcp.json) no encontrada")
	}
	if fileExists(cursorMCP) {
		checks = append(checks, "[OK] Configuración MCP verificada")
	}

	claudeRC := filepath.Join(projectPath, ".clauderc")
	if !fileExists(claudeRC) {
		failedChecks = append(failedChecks, "[Fallo] Reglas de Claude (.clauderc) no encontradas")
	}
	if fileExists(claudeRC) {
		checks = append(checks, "[OK] Reglas de Claude verificadas")
	}

	antigravityRules := filepath.Join(projectPath, ".antigravityrules")
	if !fileExists(antigravityRules) {
		failedChecks = append(failedChecks, "[Fallo] Reglas de Antigravity (.antigravityrules) no encontradas")
	}
	if fileExists(antigravityRules) {
		checks = append(checks, "[OK] Reglas de Antigravity verificadas")
	}

	// 3. Generar Evidencia
	evidenceDir := filepath.Join(qddDir, "project", "evidence", "doctor")
	_ = os.MkdirAll(evidenceDir, 0755) // Ignoramos error, si falla no creará archivo

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportPath := filepath.Join(evidenceDir, fmt.Sprintf("report_%s.md", timestamp))
	
	reportContent := "# QDD Doctor Report\n\n"
	reportContent += fmt.Sprintf("Date: %s\n\n", time.Now().Format(time.RFC3339))
	
	reportContent += "## Checks Completados\n"
	for _, c := range checks {
		reportContent += fmt.Sprintf("- %s\n", c)
	}

	if len(failedChecks) > 0 {
		reportContent += "\n## Anomalías Detectadas\n"
		for _, f := range failedChecks {
			reportContent += fmt.Sprintf("- %s\n", f)
		}
	}

	reportContent += fmt.Sprintf("\n**Estado Global:** %s\n", func() string {
		if len(failedChecks) > 0 {
			return "CRITICAL_FAILURES"
		}
		return "HEALTHY"
	}())

	_ = os.WriteFile(reportPath, []byte(reportContent), 0644)

	if len(failedChecks) > 0 {
		return false
	}

	return true
}
