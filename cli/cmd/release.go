package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:     "release [versión]",
	Aliases: []string{"RELEASE"},
	Short:   "Empaqueta una nueva versión del framework",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		version := args[0]
		fmt.Printf("🚀 Preparando Release %s...\n", version)

		qddDir := filepath.Join(".", ".qdd")
		statePath := filepath.Join(qddDir, "state.json")
		
		stateData, err := os.ReadFile(statePath)
		if err != nil {
			fmt.Printf("[!] Error leyendo state.json: %v\n", err)
			return
		}

		var state map[string]interface{}
		json.Unmarshal(stateData, &state)
		
		oldVersion := state["version"]
		state["version"] = version

		newData, _ := json.MarshalIndent(state, "", "  ")
		os.WriteFile(statePath, newData, 0644)

		fmt.Printf("✅ state.json actualizado de %v a %v\n", oldVersion, version)

		// Create git tag
		fmt.Println("📦 Creando Git Tag...")
		execCmd := exec.Command("git", "tag", version)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		if err := execCmd.Run(); err != nil {
			fmt.Printf("[!] No se pudo crear el tag en Git: %v\n", err)
			fmt.Println("    Probablemente ya exista o el directorio no es un repositorio limpio.")
			return
		}
		
		fmt.Printf("✅ Git tag %s creado exitosamente.\n", version)

		fmt.Printf("\n[🏆] Release %s finalizado. Ejecuta 'git push --tags' para publicar.\n", version)
	},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
