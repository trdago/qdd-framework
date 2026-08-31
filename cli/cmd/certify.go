package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	certifyCmd.AddCommand(humanCmd)
}

var humanCmd = &cobra.Command{
	Use:   "human [target]",
	Short: "Certifica el comportamiento humano, accesibilidad (WCAG 2.2) y usabilidad (DAG Engine)",
	Long: `Ejecuta el motor de certificación de comportamiento humano y usabilidad (DAG Engine).
Resuelve dependencias de flujos y valida la interfaz en viewports Desktop y Mobile.`,
	Run: runHumanCertify,
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

func runHumanCertify(cmd *cobra.Command, args []string) {
	target := resolveHumanTarget(args)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}

	runnerPath, err := findHumanCertifierRunner(cwd)
	if err != nil {
		fmt.Printf("[!] Error localizando motor de certificación humana: %v\n", err)
		os.Exit(1)
	}

	if err := executeHumanCertifier(runnerPath, target); err != nil {
		fmt.Printf("[!] Fallo en la certificación de comportamiento humano: %v\n", err)
		os.Exit(1)
	}
}

func resolveHumanTarget(args []string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return "all"
}

func findHumanCertifierRunner(cwd string) (string, error) {
	candidates := []string{
		filepath.Join(cwd, ".qdd", "core", "plugins", "qdd-human-certifier", "runtime", "index.mjs"),
		filepath.Join(cwd, "plugins", "qdd-human-certifier", "runtime", "index.mjs"),
		filepath.Join(cwd, "scripts", "human_qa", "index.mjs"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no se encontró runtime/index.mjs de qdd-human-certifier")
}

func executeHumanCertifier(runnerPath, target string) error {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("Node.js ('node') no está instalado o no se encuentra en el PATH")
	}

	cmd := exec.Command(nodePath, runnerPath, target)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

