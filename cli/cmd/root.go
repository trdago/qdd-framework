package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	
	"github.com/qdd-framework/qdd/pkg/qcl"
	"github.com/qdd-framework/qdd/pkg/qcl/adapters"
	"github.com/qdd-framework/qdd/pkg/qcl/nodes"
)

var rootCmd = &cobra.Command{
	Use:     "qdd",
	Short:   "QDD (Quality Driven Development) Framework CLI",
	Version: "v1.5.0",
	Long: `QDD es un CLI para gobernar, generar y evaluar arquitecturas de software aplicando certificaciones de calidad obligatorias.
garantizando certificaciones, evidencia y calidad desde el día uno.

Puedes ejecutar comandos específicos (como 'qdd init') o expresar una intención en lenguaje natural.
Ejemplo: qdd "Necesito agregar autenticación"`,
	Args: cobra.ArbitraryArgs,
	// Disable default error printing so we can intercept it gracefully
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Detectar version mismatch
		if err := validateProjectVersion(cmd.Root().Version); err != nil {
			fmt.Printf("[🛑 ERROR DE ENTORNO] %v\n", err)
			os.Exit(1)
		}

		name := cmd.Name()
		if name == "dashboard" || name == "ui" || name == "mcp-server" {
			return nil
		}

		cwd, _ := os.Getwd()
		workingPath := filepath.Join(cwd, ".qdd", "working")
		os.WriteFile(workingPath, []byte(name), 0644)

		// No exigimos contexto si apenas estamos inicializando o aprendiendo
		if name == "init" || name == "learn" {
			return nil
		}
		
		// Para todo lo demás, el Gatekeeper bloquea si falta contexto crítico
		if err := qcl.CheckMinimumAlignment(); err != nil {
			fmt.Printf("[🛑 GATEKEEPER] %v\n", err)
			os.Remove(workingPath)
			os.Exit(1)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		name := cmd.Name()
		if name == "dashboard" || name == "ui" || name == "mcp-server" {
			return
		}
		cwd, _ := os.Getwd()
		workingPath := filepath.Join(cwd, ".qdd", "working")
		os.Remove(workingPath)
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

	engine := adapters.NewGeminiEngine()
	
	pipeline := qcl.NewPipeline(
		nodes.NewContextAnalyzer(),
		nodes.NewIntentAnalyzer(engine),
		nodes.NewRiskAnalyzer(),
		nodes.NewConsultativeNode(),
		nodes.NewStrategyPlanner(engine),
		nodes.NewPlanBuilder(engine),
		nodes.NewApprovalManager(),
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

func validateProjectVersion(cliVersion string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	statePath := filepath.Join(cwd, ".qdd", "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	projVersion, ok := state["version"].(string)
	if !ok || projVersion == "" {
		return nil
	}

	if isOlder(cliVersion, projVersion) {
		return fmt.Errorf("El proyecto requiere QDD %s, pero estás ejecutando %s. Revisa si un gestor como 'mise' o 'asdf' está interceptando el binario.", projVersion, cliVersion)
	}

	return nil
}

func parseVersion(v string) (int, int, int) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(strings.Split(v, "-")[0], ".")
	var major, minor, patch int
	if len(parts) > 0 { fmt.Sscanf(parts[0], "%d", &major) }
	if len(parts) > 1 { fmt.Sscanf(parts[1], "%d", &minor) }
	if len(parts) > 2 { fmt.Sscanf(parts[2], "%d", &patch) }
	return major, minor, patch
}

func isOlder(v1, v2 string) bool {
	m1, min1, p1 := parseVersion(v1)
	m2, min2, p2 := parseVersion(v2)
	if m1 != m2 { return m1 < m2 }
	if min1 != min2 { return min1 < min2 }
	return p1 < p2
}
