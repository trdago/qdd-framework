package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/qdd-framework/qdd/pkg/integration"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Aliases: []string{"sync-ai"},
	Short:   "Sincroniza las reglas nativas (QDD Protocol) con los asistentes de IA",
	Long: `Detecta si el directorio actual es un proyecto QDD y actualiza los 
archivos de configuración de tu IDE o asistente de IA (Cursor, Claude Code, Antigravity)
para que soporten nativamente los slash commands del framework (/qdd).`,
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
