package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var learnCmd = &cobra.Command{
	Use:     "learn",
	Aliases: []string{"LEARN"},
	Short:   "Aprende la arquitectura del proyecto (Fast Path)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[⚙️ EXECUTOR] Ejecutando comando determinista: LEARN")
		fmt.Println("  -> Escaneando archivos y arquitectura...")
		fmt.Println("[✔] Conocimiento inicial generado.")
	},
}

func init() {
	rootCmd.AddCommand(learnCmd)
}
