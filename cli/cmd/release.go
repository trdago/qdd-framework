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
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		fmt.Printf("🚀 Preparando Release %s...\n", version)

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		fmt.Println("🔨 Ejecutando pipeline de Build (transpilación)...")
		if err := runBuildStep(cwd); err != nil {
			return err
		}

		qddDir := filepath.Join(".", ".qdd")
		statePath := filepath.Join(qddDir, "state.json")
		
		stateData, err := os.ReadFile(statePath)
		if err != nil {
			fmt.Printf("[!] Error leyendo state.json: %v\n", err)
			return nil
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
			return nil
		}
		
		fmt.Printf("✅ Git tag %s creado exitosamente.\n", version)

		fmt.Printf("\n[🏆] Release %s finalizado. Ejecuta 'git push --tags' para publicar.\n", version)
		return nil
	},
}

func runBuildStep(cwd string) error {
	if fileExists(filepath.Join(cwd, "package.json")) {
		fmt.Println("  -> Detectado Node.js, ejecutando 'npm run build'...")
		buildCmd := exec.Command("npm", "run", "build")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		return buildCmd.Run()
	}

	if fileExists(filepath.Join(cwd, "Makefile")) {
		fmt.Println("  -> Detectado Makefile, ejecutando 'make build'...")
		buildCmd := exec.Command("make", "build")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		return buildCmd.Run()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
