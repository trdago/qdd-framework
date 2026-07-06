package cmd

import (
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/qcl/benchmark"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Evalúa la calidad de respuesta (Cognitive Quality) del framework",
	Long: `Evalúa el desempeño de la capa cognitiva (QCL) en 4 dimensiones:
- Identificación de Contexto
- Entendimiento del Sistema
- Registro de Mejoras
- Cumplimiento de Promesas (Gobernanza)`,
	Run: runBenchmark,
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
}

func runBenchmark(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Iniciando Benchmark Cognitivo QCL...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		return
	}

	scores := benchmark.RunBenchmark(cwd)
	
	fmt.Println("[+] Evaluando modelo y generando Certificado Cognitivo...")
	cert, err := benchmark.GenerateCognitiveCertificate(cwd, scores)
	if err != nil {
		fmt.Printf("[!] Error generando certificado cognitivo: %v\n", err)
		return
	}

	fmt.Printf("\n=========================================\n")
	fmt.Printf(" CERTIFICADO DE CALIDAD COGNITIVA (QCL)\n")
	fmt.Printf("=========================================\n")
	fmt.Printf(" Fecha:           %s\n\n", cert.Timestamp)
	
	fmt.Printf(" 🧠 Contexto (Identificación): %d/100\n", cert.Scores.ContextScore)
	fmt.Printf(" 💡 Entendimiento (Intención): %d/100\n", cert.Scores.UnderstandingScore)
	fmt.Printf(" 📈 Registro de Mejoras:       %d/100\n", cert.Scores.ImprovementScore)
	fmt.Printf(" ⚖️  Cumplimiento (Promesas):  %d/100\n\n", cert.Scores.ComplianceScore)

	fmt.Printf(" Score Total:     %d/100\n", cert.Total)
	printTendencyCognitive(cert.Tendency)
	fmt.Printf("=========================================\n\n")
}

func printTendencyCognitive(tendency audit.Tendency) {
	if tendency == audit.TendencyImproving {
		fmt.Println(" Tendencia:       📈 MEJORANDO")
		return
	}
	if tendency == audit.TendencyWorsening {
		fmt.Println(" Tendencia:       📉 EMPEORANDO")
		return
	}
	fmt.Println(" Tendencia:       ➡️ ESTABLE")
}
