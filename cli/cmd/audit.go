package cmd

import (
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Ejecuta la Inspección Técnica (Fase G)",
	Long: `Evalúa el código contra las reglas de gobernanza del QDD Framework.
Si encuentra violaciones, retornará un código de salida distinto de cero (falla en CI).
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
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}

	engine := audit.NewEngine(cwd)
	fmt.Println("[+] Ejecutando motor de auditoría...")
	violations := engine.RunAll()

	if len(violations) == 0 {
		fmt.Println("[+] Se encontraron 0 violaciones. Código limpio y alineado.")
		os.Exit(0)
	}

	fmt.Printf("[!] Se encontraron %d violaciones de deuda técnica o no-cumplimiento:\n", len(violations))
	for _, v := range violations {
		fmt.Printf("    - [%s] %s\n", v.Category, v.Format())
	}

	fmt.Println("\n[!] Fallo de auditoría. Repare las violaciones para poder certificar.")
	os.Exit(1)
}
