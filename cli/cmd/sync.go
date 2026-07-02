package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qdd-framework/qdd/pkg/integration"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize QDD AI integration (Adapters)",
	Long: `Detects if the current directory is a QDD project and updates the 
AI assistant configuration files (.cursorrules, .clauderc, .antigravityrules)
so that they natively support the /qdd commands.`,
	Run: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Inicializando sincronización con Adaptadores AI...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		return
	}

	manager := integration.NewIntegrationManager()
	
	err = manager.SyncAll(cwd)
	if err != nil {
		fmt.Printf("[!] Error durante la sincronización: %v\n", err)
		return
	}

	fmt.Println("[+] ¡Sincronización completada! Tu asistente de IA ahora comprende comandos nativos QDD.")
}
