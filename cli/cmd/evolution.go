package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/evolution"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/spf13/cobra"
)

var evolutionCmd = &cobra.Command{
	Use:   "evolution",
	Short: "Estudia el conocimiento acumulado del proyecto y recomienda la siguiente mejora",
	Long: `Analiza findings, certificaciones, violaciones de auditoría y el historial de score
para determinar cuál debería ser la siguiente mejora natural del proyecto. Es de solo lectura:
nunca crea ni modifica certificaciones por sí solo — en Modo Consultivo, propone y espera autorización.`,
	Run: runEvolution,
}

func init() {
	rootCmd.AddCommand(evolutionCmd)
}

func runEvolution(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Estudiando la evolución del proyecto...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}
	cwd = integration.FindProjectRoot(cwd)

	violations := audit.NewEngine(cwd).RunAll()

	report, err := evolution.Analyze(cwd, len(violations))
	if err != nil {
		fmt.Printf("[!] Error analizando la evolución del proyecto: %v\n", err)
		os.Exit(1)
	}

	printEvolutionReport(report)
	saveEvolutionEvidence(cwd, report)
}

func printEvolutionReport(r *evolution.Report) {
	fmt.Printf("\n[Score actual] %d/100 (Tendencia: %s)\n", r.Score, r.Tendency)
	fmt.Printf("[Violaciones de auditoría activas] %d\n", r.Violations)
	fmt.Printf("[Findings abiertos] %d\n", len(r.OpenFindings))
	for _, f := range r.OpenFindings {
		fmt.Printf("    - %s: %s\n", f.ID, f.Title)
	}
	fmt.Printf("[Certificaciones de proyecto pendientes] %d\n", len(r.PendingCerts))
	for _, c := range r.PendingCerts {
		fmt.Printf("    - %s: %s\n", c.ID, c.Title)
	}
	fmt.Printf("[Certificaciones core disponibles para adopción] %d\n", len(r.AvailableCoreCerts))

	fmt.Printf("\n[Prioridad: %s] %s\n", r.Priority, r.Recommendation)
}

func saveEvolutionEvidence(cwd string, r *evolution.Report) {
	evidenceDir := filepath.Join(cwd, ".qdd", "project", "evidence", "evolution")
	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportPath := filepath.Join(evidenceDir, fmt.Sprintf("report_%s.md", timestamp))

	content := "# QDD Evolution Report\n\n"
	content += fmt.Sprintf("Date: %s\n\n", time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("- Score: %d/100 (Tendencia: %s)\n", r.Score, r.Tendency)
	content += fmt.Sprintf("- Violaciones activas: %d\n", r.Violations)
	content += fmt.Sprintf("- Findings abiertos: %d\n", len(r.OpenFindings))
	content += fmt.Sprintf("- Certificaciones de proyecto pendientes: %d\n", len(r.PendingCerts))
	content += fmt.Sprintf("- Certificaciones core disponibles: %d\n\n", len(r.AvailableCoreCerts))
	content += fmt.Sprintf("## Prioridad: %s\n\n%s\n", r.Priority, r.Recommendation)

	_ = os.WriteFile(reportPath, []byte(content), 0644)
}
