package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	
	"github.com/qdd-framework/qdd/pkg/qcl"
	"github.com/qdd-framework/qdd/pkg/qcl/nodes"
)

var rootCmd = &cobra.Command{
	Use:   "qdd [intención]",
	Short: "QDD es un Framework de Ingeniería de Software Nativo-IA",
	Long: `QDD (Quality-Driven Development) orquesta el ciclo de vida del 
desarrollo de software asistido por IA, garantizando certificaciones,
evidencia y calidad desde el día uno.

Puedes ejecutar comandos específicos (como 'qdd init') o expresar una intención en lenguaje natural.
Ejemplo: qdd "Necesito agregar autenticación"`,
	Args: cobra.ArbitraryArgs,
	// Disable default error printing so we can intercept it gracefully
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// No exigimos contexto si apenas estamos inicializando o aprendiendo
		if cmd.Name() == "init" || cmd.Name() == "learn" {
			return nil
		}
		// Para todo lo demás, el Gatekeeper bloquea si falta contexto crítico
		if err := qcl.CheckMinimumAlignment(); err != nil {
			fmt.Printf("[🛑 GATEKEEPER] %v\n", err)
			os.Exit(1)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}

		input := strings.Join(args, " ")
		runQCL(input)
	},
}

func runQCL(input string) {
	// Revisión manual de Gatekeeper por si entra vía interceptor de Execute()
	if err := qcl.CheckMinimumAlignment(); err != nil {
		fmt.Printf("[🛑 GATEKEEPER] %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[QCL] Iniciando proceso cognitivo para: '%s'\n", input)

	pipeline := qcl.NewPipeline(
		nodes.NewContextAnalyzer(),
		nodes.NewIntentAnalyzer(),
		nodes.NewRiskAnalyzer(),
		nodes.NewStrategyPlanner(),
		nodes.NewPlanBuilder(),
	)

	session, err := pipeline.Execute(input)
	if err != nil {
		fmt.Printf("[!] Error en la capa cognitiva: %v\n", err)
		os.Exit(1)
	}

	if session.ClarificationRequest != nil {
		fmt.Printf("\n[🤔] %s\n", session.ClarificationRequest.Message)
		for i, opt := range session.ClarificationRequest.Options {
			fmt.Printf("  %d. %s\n", i+1, opt)
		}
		fmt.Print("Selecciona una opción (número): ")
		
		var choice int
		_, err := fmt.Scanf("%d", &choice)
		if err != nil || choice < 1 || choice > len(session.ClarificationRequest.Options) {
			fmt.Println("[!] Selección inválida. Abortando.")
			os.Exit(1)
		}
		
		newInput := session.ClarificationRequest.Options[choice-1]
		fmt.Printf("\n[+] Re-evaluando con intención aclarada: '%s'\n\n", newInput)
		runQCL(newInput)
		return
	}

	if session.ApprovalRequest != nil {
		fmt.Printf("\n[?] Aprobación Requerida: %s\n", session.ApprovalRequest.Reason)
		return
	}

	fmt.Println("\n[✔] Plan cognitivo completado.")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Interceptamos "unknown command" para pasarlo al Cognitive Layer
		if strings.HasPrefix(err.Error(), "unknown command") {
			input := strings.Join(os.Args[1:], " ")
			runQCL(input)
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
