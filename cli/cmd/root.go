package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/qdd-framework/qdd/pkg/findings"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/qcl"
	"gopkg.in/yaml.v3"
)

//go:embed version.txt
var embeddedVersion string

var Version = "dev" // Se inyecta dinámicamente en build-time con -ldflags o embed

func init() {
	v := strings.TrimSpace(embeddedVersion)
	if v != "" {
		Version = v
		rootCmd.Version = v
	}
	rootCmd.AddCommand(undoCmd)
}

var undoCmd = &cobra.Command{
	Use:   "undo <comando>",
	Short: "Revierte el efecto de un comando (p. ej. 'qdd undo init'); lo usa el rollback de 'qdd run'",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		target := args[0]
		if target == "init" {
			cwd, _ := os.Getwd()
			qddPath := filepath.Join(cwd, ".qdd")
			if _, err := os.Stat(qddPath); err == nil {
				os.RemoveAll(qddPath)
				fmt.Printf("[UNDO] Carpeta .qdd eliminada exitosamente para revertir 'init'.\n")
			}
			return
		}
		fmt.Printf("[UNDO] Revirtiendo estado del comando '%s' (No-op en comandos de solo lectura)\n", target)
	},
}

var rootCmd = &cobra.Command{
	Use:     "qdd",
	Short:   "QDD (Quality Driven Development) Framework CLI",
	Version: Version,
	Long: `QDD es un CLI para gobernar, generar y evaluar arquitecturas de software aplicando
certificaciones de calidad, con evidencia y calidad desde el día uno.

Para empezar, en la carpeta de tu proyecto:
  qdd init     configura QDD y conecta tu IA (Claude Code, Cursor, Antigravity, OpenCode)
  qdd audit    revisa el proyecto (funciona desde cualquier subcarpeta)

Para encadenar comandos con rollback automático si uno falla:
  qdd run "init" "audit"`,
	Args: cobra.ArbitraryArgs,
	// Disable default error printing so we can intercept it gracefully
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		enterProjectRoot(cmd.Name())
		migrateBlueprints(cmd.Name())

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

		// No exigimos contexto si apenas estamos inicializando, aprendiendo, reparando,
		// o reportando un error (bug): un proceso supervisado puede fallar incluso en un
		// proyecto que todavía no completó `qdd init`, y ese fallo igual debe poder registrarse.
		if name == "init" || name == "learn" || name == "doctor" || name == "bug" {
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

// migrateBlueprints migra, la primera vez que corre esta versión en un
// proyecto del esquema anterior, sus blueprints a la carpeta central (RFC
// 0004). Escribe en stderr para no ensuciar salidas como --json. El servidor
// MCP no migra al arrancar: lo hace qdd_project_context, para que la IA se
// entere y lo termine.
func migrateBlueprints(name string) {
	if name == "init" || name == "mcp-server" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	if notice := migrationNotice(cwd); notice != "" {
		fmt.Fprint(os.Stderr, notice)
	}
}

// enterProjectRoot hace que cada comando opere sobre la raíz del proyecto
// aunque se invoque desde una subcarpeta (FND-044). init decide su propia raíz,
// y si no hay un proyecto QDD hacia arriba el directorio no cambia.
func enterProjectRoot(name string) {
	if name == "init" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	root := integration.FindProjectRoot(cwd)
	if root == cwd || !integration.IsQDDProject(root) {
		return
	}
	_ = os.Chdir(root)
}

func runQCL(input string) {
	fmt.Printf("[QCL] Intención recibida: '%s'\n\n", input)
	fmt.Println("⚠️  ATENCIÓN: El motor cognitivo interno basado en API ha sido deprecado.")
	fmt.Println("El QDD Framework ahora opera exclusivamente como un entorno Agentic Harness (Servidor MCP).")
	fmt.Println("Para procesar esta intención, por favor comunícate directamente con tu IA externa (Antigravity, Claude Code, Cursor) y pídele que utilice las herramientas de QDD para cumplir tu petición.")
	os.Exit(0)
}

func Execute() {
	defer func() {
		if r := recover(); r != nil {
			handlePanicRecovery(r)
		}
	}()

	if isSupervisorMode() {
		executeSupervisor(os.Args[3:])
		return
	}

	if isPipelineMode() {
		executePipeline()
		return
	}

	if err := rootCmd.Execute(); err != nil {
		handleRootCmdError(err)
	}
}

func handlePanicRecovery(r interface{}) {
	fmt.Printf("\n[🛑 RECOVERY - ZERO PANIC] Se interceptó una excepción no controlada: %v\n", r)
	fmt.Printf("El QDD Framework previno la caída abrupta del proceso y documentó el incidente.\n")

	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
	if mkErr := os.MkdirAll(findingsDir, 0755); mkErr != nil {
		os.Exit(1)
	}

	fndID := fmt.Sprintf("FND-PANIC-%d", time.Now().Unix())
	fndPath := filepath.Join(findingsDir, fndID+".yaml")

	content, err := findingDePanico(fndID, r, time.Now())
	if err == nil {
		_ = os.WriteFile(fndPath, content, 0644)
	}

	os.Exit(1)
}

// findingDePanico registra el pánico como un finding abierto: atraparlo evita
// la caída, pero el defecto que lo causó sigue sin corregir. Se serializa con
// yaml para que cualquier mensaje quede legible (FND-062).
func findingDePanico(id string, r interface{}, ahora time.Time) ([]byte, error) {
	return yaml.Marshal(struct {
		ID           string `yaml:"id"`
		Type         string `yaml:"type"`
		Title        string `yaml:"title"`
		Status       string `yaml:"status"`
		Severity     string `yaml:"severity"`
		Impact       string `yaml:"impact"`
		DetectadoPor string `yaml:"detectado_por"`
		CreatedAt    string `yaml:"created_at"`
	}{
		ID:           id,
		Type:         "PANIC_RECOVERED",
		Title:        "Pánico interceptado por Zero-Panic Guard",
		Status:       string(findings.Abierto),
		Severity:     "CRITICAL",
		Impact:       fmt.Sprintf("CRITICAL - Excepción no controlada interceptada en tiempo de ejecución: %v", r),
		DetectadoPor: "qdd",
		CreatedAt:    ahora.Format(time.RFC3339),
	})
}

func isPipelineMode() bool {
	return len(os.Args) > 1 && os.Args[1] == "run"
}

func isSupervisorMode() bool {
	return isPipelineMode() && len(os.Args) > 2 && os.Args[2] == "--keep-alive"
}

func executePipeline() {
	var completed []string
	for _, cmdStr := range os.Args[2:] {
		fmt.Printf("\n🚀 [QDD PIPELINE] Ejecutando: qdd %s\n", cmdStr)
		
		if err := executePipelineCommand(cmdStr); err != nil {
			fmt.Printf("\n[🛑 ERROR PIPELINE] El comando '%s' falló. Abortando pipeline secuencial y ejecutando Rollback automático.\n", cmdStr)
			executeRollback(completed)
			os.Exit(1)
		}
		completed = append(completed, cmdStr)
	}
	fmt.Printf("\n✅ [QDD PIPELINE] Todos los comandos ejecutados exitosamente.\n")
}

func executeRollback(completed []string) {
	for i := len(completed) - 1; i >= 0; i-- {
		cmdStr := completed[i]
		fmt.Printf("🔄 [QDD ROLLBACK] Deshaciendo cambios de: qdd %s\n", cmdStr)
		// Ejecuta el rollback específico para el comando
		executePipelineCommand("undo " + cmdStr)
	}
}

// selfExecutable devuelve la ruta absoluta del binario en ejecución:
// os.StartProcess no busca en el PATH, así que con os.Args[0] = "qdd" (la forma
// normal de invocarlo) `qdd run` nunca lograba iniciar un comando (FND-046).
func selfExecutable() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	if p, err := exec.LookPath(os.Args[0]); err == nil {
		return p
	}
	return os.Args[0]
}

func executePipelineCommand(cmdStr string) error {
	execCmd := selfExecutable()
	args := strings.Fields(cmdStr)
	processArgs := append([]string{os.Args[0]}, args...)
	
	process, err := os.StartProcess(execCmd, processArgs, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		fmt.Printf("[🛑 ERROR PIPELINE] No se pudo iniciar el comando %s: %v\n", cmdStr, err)
		return err
	}
	
	state, err := process.Wait()
	if err != nil || !state.Success() {
		return fmt.Errorf("command execution failed")
	}
	return nil
}

func handleRootCmdError(err error) {
	if strings.HasPrefix(err.Error(), "unknown command") {
		input := strings.Join(os.Args[1:], " ")
		runQCL(input)
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func validateProjectVersion(cliVersion string) error {
	state, err := loadProjectState()
	if hasInvalidState(state, err) {
		return nil
	}

	projVersion := extractProjectVersion(state)
	if projVersion == "" {
		return nil
	}

	return checkVersionAge(cliVersion, projVersion)
}

func hasInvalidState(state map[string]interface{}, err error) bool {
	return err != nil || state == nil
}

func extractProjectVersion(state map[string]interface{}) string {
	if projVersion, ok := state["version"].(string); ok {
		return projVersion
	}
	return ""
}

func checkVersionAge(cliVersion, projVersion string) error {
	if cliVersion == "dev" {
		return nil
	}
	if isOlder(cliVersion, projVersion) {
		return fmt.Errorf("El proyecto requiere QDD %s, pero estás ejecutando %s. Revisa si un gestor como 'mise' o 'asdf' está interceptando el binario.", projVersion, cliVersion)
	}
	return nil
}

func loadProjectState() (map[string]interface{}, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	
	statePath := filepath.Join(cwd, ".qdd", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}
	
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	
	return state, nil
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
