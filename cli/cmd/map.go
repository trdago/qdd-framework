package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qdd-framework/qdd/pkg/topology"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Genera el mapa topológico de certificación del proyecto (App -> Módulo -> Endpoint)",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		
		fmt.Println("[+] Escaneando topología del proyecto...")
		
		projTopology, err := topology.MapProject(cwd)
		if err != nil {
			fmt.Printf("[!] Error al mapear el proyecto: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("[✔] Topología generada. Score Global de Certificación: %d%%\n", projTopology.GlobalScore)
		
		// Guardar en .qdd/project/topology.json
		qddDir := filepath.Join(cwd, ".qdd", "project")
		os.MkdirAll(qddDir, 0755)
		
		outPath := filepath.Join(qddDir, "topology.json")
		data, _ := json.MarshalIndent(projTopology, "", "  ")
		
		err = os.WriteFile(outPath, data, 0644)
		if err != nil {
			fmt.Printf("[!] Error guardando topología: %v\n", err)
			return
		}
		
		fmt.Println("[+] Topología guardada en", outPath)
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}
