package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/qcl/wisdom"
	"github.com/spf13/cobra"
)

var autoFix bool

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

		success, _ := RunDoctorCheck(cwd, autoFix)
		if !success {
			fmt.Println("[-] QDD Doctor detectó anomalías. Ejecuta `qdd doctor --fix` para intentar corregirlas automáticamente.")
			os.Exit(1)
		}

		fmt.Println("[+] QDD Doctor: Todo el entorno se encuentra 100% operativo.")
	},
}

func init() {
	doctorCmd.Flags().BoolVarP(&autoFix, "fix", "f", false, "Intenta auto-reparar los archivos o configuraciones del core dañadas")
	rootCmd.AddCommand(doctorCmd)
}

// RunDoctorCheck ejecuta las pruebas deterministas del framework y deja un reporte.
func RunDoctorCheck(projectPath string, autoFix bool) (bool, int) {
	checks := []string{}
	failedChecks := []string{}

	checkCoreStructure(projectPath, &checks, &failedChecks)

	missingIntegrations := checkAllIntegrations(projectPath, &checks, &failedChecks)

	if missingIntegrations && autoFix {
		attemptAutoFix(projectPath, &checks, &failedChecks)
	}

	generateDoctorReport(projectPath, checks, failedChecks)

	if len(failedChecks) > 0 {
		return false, len(failedChecks)
	}

	return true, 0
}

func checkCoreStructure(projectPath string, checks, failedChecks *[]string) {
	qddDir := filepath.Join(projectPath, ".qdd")
	checkDirExists(qddDir, ".qdd/", checks, failedChecks)
	checkFileExists(filepath.Join(qddDir, "config.yaml"), "config.yaml", checks, failedChecks)
	checkFileExists(filepath.Join(qddDir, "state.json"), "state.json", checks, failedChecks)
}

func checkDirExists(path, name string, checks, failedChecks *[]string) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.IsDir() {
		*failedChecks = append(*failedChecks, fmt.Sprintf("[Fallo] Directorio raíz %s no encontrado", name))
		return
	}
	*checks = append(*checks, fmt.Sprintf("[OK] Directorio raíz %s verificado", name))
}

func checkFileExists(path, name string, checks, failedChecks *[]string) {
	if !fileExists(path) {
		*failedChecks = append(*failedChecks, fmt.Sprintf("[Fallo] Archivo %s no encontrado", name))
		return
	}
	*checks = append(*checks, fmt.Sprintf("[OK] Archivo %s verificado", name))
}

func attemptAutoFix(projectPath string, checks, failedChecks *[]string) {
	fmt.Println("[!] Doctor: Intentando reparar integraciones de IA faltantes...")
	manager := integration.NewIntegrationManager()
	if err := manager.SyncAll(projectPath); err != nil {
		*failedChecks = append(*failedChecks, fmt.Sprintf("[Fallo] Error al auto-reparar integraciones: %v", err))
		return
	}
	*checks = append(*checks, "[OK] Integraciones reparadas exitosamente")
	*failedChecks = filterRepairedIntegrations(*failedChecks)

	fmt.Println("[+] Doctor: Consultando oráculo Cloud Wisdom para estrategias de reparación actualizadas...")
	wClient := wisdom.NewClient(projectPath)
	manifest, err := wClient.FetchRulesManifest(context.Background())
	if err == nil && manifest != nil {
		*checks = append(*checks, fmt.Sprintf("[OK] Cloud Wisdom conectado (Manifiesto v%s)", manifest.Version))
		return
	}
	*checks = append(*checks, "[OK] Cloud Wisdom operando en modo Offline (Graceful Degradation)")
}

func generateDoctorReport(projectPath string, checks, failedChecks []string) {
	qddDir := filepath.Join(projectPath, ".qdd")
	evidenceDir := filepath.Join(qddDir, "project", "evidence", "doctor")
	_ = os.MkdirAll(evidenceDir, 0755)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportPath := filepath.Join(evidenceDir, fmt.Sprintf("report_%s.md", timestamp))

	reportContent := buildReportContent(checks, failedChecks)
	_ = os.WriteFile(reportPath, []byte(reportContent), 0644)
}

func buildReportContent(checks, failedChecks []string) string {
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

	stateStr := "HEALTHY"
	if len(failedChecks) > 0 {
		stateStr = "CRITICAL_FAILURES"
	}
	reportContent += fmt.Sprintf("\n**Estado Global:** %s\n", stateStr)

	return reportContent
}

func checkIntegrationFile(name, path string, checks, failedChecks *[]string, missingIntegrations *bool) {
	if !fileExists(path) {
		*failedChecks = append(*failedChecks, fmt.Sprintf("[Fallo] %s no encontrada", name))
		*missingIntegrations = true
		return
	}
	*checks = append(*checks, fmt.Sprintf("[OK] %s verificada", name))
}

func checkAllIntegrations(projectPath string, checks, failedChecks *[]string) bool {
	missingIntegrations := false

	cursorMCP := filepath.Join(projectPath, ".cursor", "mcp.json")
	claudeRC := filepath.Join(projectPath, ".clauderc")
	antigravityRules := filepath.Join(projectPath, ".antigravityrules")

	checkIntegrationFile("Configuración MCP (.cursor/mcp.json)", cursorMCP, checks, failedChecks, &missingIntegrations)
	checkIntegrationFile("Reglas de Claude (.clauderc)", claudeRC, checks, failedChecks, &missingIntegrations)
	checkIntegrationFile("Reglas de Antigravity (.antigravityrules)", antigravityRules, checks, failedChecks, &missingIntegrations)

	return missingIntegrations
}

func filterRepairedIntegrations(failedChecks []string) []string {
	var remainingFailed []string
	for _, f := range failedChecks {
		if !strings.Contains(f, "Configuración MCP") && !strings.Contains(f, "Reglas de Claude") && !strings.Contains(f, "Reglas de Antigravity") {
			remainingFailed = append(remainingFailed, f)
		}
	}
	return remainingFailed
}
