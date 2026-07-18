package cmd

import (
	"fmt"
	"os"

	"github.com/qdd-framework/qdd/pkg/bugreport"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/spf13/cobra"
)

var (
	bugTitle    string
	bugCommand  string
	bugExitCode int
	bugOutput   string
)

var bugCmd = &cobra.Command{
	Use:   "bug",
	Short: "Registra un error como Finding permanente (usado también por `qdd run --keep-alive`)",
	Long: `Convierte un error observado (de un proceso supervisado, un CI, o reportado a mano) en un
Finding con estado OPEN y su evidencia asociada, siguiendo el principio 'Aprendizaje Perpetuo'.
No repara nada por sí solo: deja el finding marcado con test_pending para que un humano o una IA
conectada por MCP agregue el test de regresión y aplique el fix real.`,
	Run: runBug,
}

func init() {
	bugCmd.Flags().StringVar(&bugTitle, "title", "", "Título del problema (opcional)")
	bugCmd.Flags().StringVar(&bugCommand, "command", "", "Comando que falló (requerido)")
	bugCmd.Flags().IntVar(&bugExitCode, "exit-code", 1, "Código de salida observado")
	bugCmd.Flags().StringVar(&bugOutput, "output", "", "Salida/stacktrace capturado")
	bugCmd.MarkFlagRequired("command")
	rootCmd.AddCommand(bugCmd)
}

func runBug(cmd *cobra.Command, args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}
	cwd = integration.FindProjectRoot(cwd)

	filed, err := bugreport.File(cwd, bugreport.Report{
		Title:    bugTitle,
		Command:  bugCommand,
		Args:     args,
		ExitCode: bugExitCode,
		Output:   bugOutput,
	})
	if err != nil {
		fmt.Printf("[!] Error registrando el bug: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[!] Finding %s creado (%s).\n", filed.Finding.ID, filed.FindingPath)
	fmt.Printf("[!] Evidencia: %s\n", filed.EvidencePath)
	fmt.Println("[!] Pendiente: agregar un test de regresión antes de marcarlo RESOLVED (ver metadata.test_note).")
}
