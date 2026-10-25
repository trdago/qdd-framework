package cmd

import (
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Revisa el proyecto contra las reglas de QDD (pruebas, OWASP, código limpio...)",
	Long: `Evalúa el código contra las reglas de gobernanza del QDD Framework.
Si algún problema bloquea el gate, termina con código distinto de cero (falla en CI).
En adopción gradual (lo que deja 'qdd init' en un proyecto nuevo) las prácticas de
madurez —E2E, ADR, Conventional Commits— se informan como recomendaciones sin bloquear.
No genera ni guarda el Certificado Histórico (eso es tarea de 'qdd certify').`,
	Run: runAudit,
}

func init() {
	rootCmd.AddCommand(auditCmd)
}

func runAudit(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Iniciando Auditoría Técnica de QDD...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[✗] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}

	engine := audit.NewEngine(cwd)
	fmt.Println("[+] Ejecutando motor de auditoría...")
	violations := engine.RunAll()

	fmt.Print(formatAuditResult(violations))
	if len(audit.Bloqueantes(violations)) > 0 {
		os.Exit(1)
	}
}

// formatAuditResult separa lo que bloquea el gate de las recomendaciones, para
// que un proyecto sano no parezca roto (FND-040). Lo usan audit, certify y MCP.
func formatAuditResult(violations []audit.Violation) string {
	bloqueantes := audit.Bloqueantes(violations)
	out := ""
	if len(bloqueantes) > 0 {
		out += fmt.Sprintf("[✗] %d problema(s) bloquean el gate:\n", len(bloqueantes))
		out += listarViolaciones(bloqueantes)
	}
	if len(bloqueantes) == 0 {
		out += "[✓] Auditoría aprobada: ningún problema bloquea el gate.\n"
	}
	out += formatRecomendaciones(audit.Recomendaciones(violations))
	out += fmt.Sprintf("[i] Puntaje QDD: %d/100\n", audit.Score(violations))
	if len(bloqueantes) > 0 {
		out += "\nRepara los problemas que bloquean para poder certificar.\n"
	}
	return out
}

func formatRecomendaciones(recomendaciones []audit.Violation) string {
	if len(recomendaciones) == 0 {
		return ""
	}
	out := fmt.Sprintf("[i] %d recomendación(es) para madurar el proyecto (no bloquean; para exigirlas: adopcion: estricta en .qdd/policies.yaml):\n", len(recomendaciones))
	return out + listarViolaciones(recomendaciones)
}

func listarViolaciones(violations []audit.Violation) string {
	out := ""
	for _, v := range violations {
		out += fmt.Sprintf("    - [%s] %s\n", v.Category, v.Format())
	}
	return out
}
