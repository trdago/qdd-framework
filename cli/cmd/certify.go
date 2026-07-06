package cmd

import (
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/spf13/cobra"
)

var certifyCmd = &cobra.Command{
	Use:   "certify",
	Short: "Evalúa la calidad del framework y emite un certificado histórico",
	Long: `Ejecuta el motor de auditoría completo y calcula un Score de Calidad.
Compara el score actual con el historial para determinar si la calidad está mejorando o empeorando.`,
	Run: runCertify,
}

func init() {
	rootCmd.AddCommand(certifyCmd)
}

func runCertify(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Iniciando evaluación de calidad (Certificado QDD)...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		return
	}

	engine := audit.NewEngine(cwd)
	fmt.Println("[+] Ejecutando motor de auditoría...")
	violations := engine.RunAll()

	fmt.Printf("[+] Se encontraron %d violaciones.\n", len(violations))
	
	// Print a summary of the violations
	for _, v := range violations {
		fmt.Printf("    - [%s] %s\n", v.Category, v.Format())
	}

	fmt.Println("[+] Generando certificado histórico...")
	cert, err := audit.GenerateCertificate(cwd, violations)
	if err != nil {
		fmt.Printf("[!] Error generando certificado: %v\n", err)
		return
	}

	fmt.Printf("\n=========================================\n")
	fmt.Printf(" CERTIFICADO DE CALIDAD QDD\n")
	fmt.Printf("=========================================\n")
	fmt.Printf(" Fecha:        %s\n", cert.Timestamp)
	fmt.Printf(" Violaciones:  %d\n", cert.TotalViolations)
	fmt.Printf(" Score:        %d/100\n", cert.Score)
	
	printTendency(cert.Tendency)
	
	fmt.Printf("=========================================\n\n")

	// Suggestion: if worsening, exit with a non-zero code to block CI?
	// The user didn't specify, but let's just exit 0 as we want it to be a reporting tool.
}

func printTendency(tendency audit.Tendency) {
	if tendency == audit.TendencyImproving {
		fmt.Println(" Tendencia:    📈 MEJORANDO")
		return
	}
	if tendency == audit.TendencyWorsening {
		fmt.Println(" Tendencia:    📉 EMPEORANDO")
		return
	}
	fmt.Println(" Tendencia:    ➡️ ESTABLE")
}
