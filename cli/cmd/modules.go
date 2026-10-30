package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/modules"
	"github.com/spf13/cobra"
)

var modulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Fichas de módulo: el corazón operativo y tecnológico de cada módulo",
	Long: `Lista las fichas de .qdd/project/modules con su estado y desfase, y las zonas
de código que aún no tienen ficha. Las IAs proponen fichas (MCP qdd_module_save,
siempre en borrador); un humano las aprueba con 'qdd modules approve <id>'.`,
	Run: runModulesList,
}

var modulesApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Aprueba una ficha y la marca como revisada en el commit actual",
	Args:  cobra.ExactArgs(1),
	Run:   runModulesApprove,
}

var modulesContextCmd = &cobra.Command{
	Use:   "context [rutas...]",
	Short: "Imprime el contexto QDD que recibe una IA (igual que la tool MCP qdd_project_context)",
	Run:   runModulesContext,
}

func init() {
	modulesCmd.AddCommand(modulesApproveCmd)
	modulesCmd.AddCommand(modulesContextCmd)
	rootCmd.AddCommand(modulesCmd)
}

func modulesRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error obteniendo directorio actual: %v\n", err)
		os.Exit(1)
	}
	return integration.FindProjectRoot(cwd)
}

func runModulesList(cmd *cobra.Command, args []string) {
	root := modulesRoot()
	mods, errs := modules.Load(root)
	for _, err := range errs {
		fmt.Printf("[!] Ficha ilegible: %v\n", err)
	}
	if len(mods) == 0 {
		fmt.Printf("Sin fichas en %s.\n", modules.Dir)
	}
	for _, m := range mods {
		fmt.Printf("- %-24s %-9s %s\n", m.ID, m.Estado, moduleAlert(root, m))
	}
	zones := modules.UncoveredZones(root, mods)
	if len(zones) == 0 {
		return
	}
	fmt.Printf("\nZonas de código sin ficha (%d):\n", len(zones))
	for _, z := range zones {
		fmt.Printf("  %s\n", z)
	}
}

func moduleAlert(root string, m modules.Module) string {
	if problems := m.Validate(root); len(problems) > 0 {
		return "INVÁLIDA: " + strings.Join(problems, "; ")
	}
	if m.Estado != modules.EstadoAprobado {
		return "pendiente de aprobación humana"
	}
	drift := modules.Drift(root, m)
	if drift > 0 {
		return fmt.Sprintf("desfasada: %d commit(s) desde la revisión %s", drift, m.UltimaRevision)
	}
	return "vigente (revisada " + m.UltimaRevision + ")"
}

func runModulesApprove(cmd *cobra.Command, args []string) {
	m, problems := modules.Approve(modulesRoot(), args[0])
	if len(problems) > 0 {
		fmt.Printf("[!] No se pudo aprobar '%s':\n  - %s\n", args[0], strings.Join(problems, "\n  - "))
		os.Exit(1)
	}
	fmt.Println(modules.TextoAprobada(m))
}

func runModulesContext(cmd *cobra.Command, args []string) {
	root := modulesRoot()
	fmt.Print(modules.BuildContext(modules.ContextInput{
		Root:         root,
		Paths:        args,
		TestCommands: audit.TestCommandLabels(root),
		Estandares:   blueprints.Estandares(root),
	}))
}
