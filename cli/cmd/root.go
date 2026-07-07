package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	
	"github.com/qdd-framework/qdd/pkg/qcl"
)

var rootCmd = &cobra.Command{
	Use:     "qdd",
	Short:   "QDD (Quality Driven Development) Framework CLI",
	Version: "v1.7.0",
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
	fmt.Printf("[QCL] Intención recibida: '%s'\n\n", input)
	fmt.Println("⚠️  ATENCIÓN: El motor cognitivo interno basado en API ha sido deprecado.")
	fmt.Println("El QDD Framework ahora opera exclusivamente como un entorno Agentic Harness (Servidor MCP).")
	fmt.Println("Para procesar esta intención, por favor comunícate directamente con tu IA externa (Antigravity, Claude Code, Cursor) y pídele que utilice las herramientas de QDD para cumplir tu petición.")
	os.Exit(0)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		// Native Pipelining: Execute each subsequent argument sequentially
		for _, cmdStr := range os.Args[2:] {
			fmt.Printf("\n🚀 [QDD PIPELINE] Ejecutando: qdd %s\n", cmdStr)
			
			// Creamos un nuevo rootCmd por cada iteración para limpiar estado
			// o podemos usar os/exec para aislar la ejecución
			
			// Para evitar problemas de estado de Cobra, delegamos a un subproceso
			execCmd := os.Args[0]
			process, err := os.StartProcess(execCmd, []string{execCmd, cmdStr}, &os.ProcAttr{
				Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			})
			if err != nil {
				fmt.Printf("[🛑 ERROR PIPELINE] No se pudo iniciar el comando %s: %v\n", cmdStr, err)
				os.Exit(1)
			}
			
			state, err := process.Wait()
			if err != nil || !state.Success() {
				fmt.Printf("\n[🛑 ERROR PIPELINE] El comando '%s' falló. Abortando pipeline secuencial.\n", cmdStr)
				os.Exit(1)
			}
		}
		fmt.Printf("\n✅ [QDD PIPELINE] Todos los comandos ejecutados exitosamente.\n")
		return
	}

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
