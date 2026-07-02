package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"STATUS"},
	Short:   "Muestra el estado de gobernanza del proyecto",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("--- QDD PROJECT STATUS ---")

		qddDir := filepath.Join(".", ".qdd")
		
		// 1. Leer State
		statePath := filepath.Join(qddDir, "state.json")
		stateData, err := os.ReadFile(statePath)
		if err == nil {
			var state map[string]interface{}
			json.Unmarshal(stateData, &state)
			fmt.Printf("**Versión:** %v\n", state["version"])
			fmt.Printf("**Estado de Inicialización:** %v\n", state["status"])
		}

		fmt.Println("**Entorno de Gobernanza:** Activo (`.qdd/`)")
		fmt.Println("📊 **Métricas de Calidad y Gobernanza:**")

		// 2. Contar Certificaciones
		certDir := filepath.Join(qddDir, "certification")
		entries, _ := os.ReadDir(certDir)
		totalCerts := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				totalCerts++
			}
		}
		fmt.Printf("✅ **Certificaciones Activas (%d):**\n", totalCerts)
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				content, _ := os.ReadFile(filepath.Join(certDir, entry.Name()))
				var cert Certification // defined in certify.go
				yaml.Unmarshal(content, &cert)
				
				icon := "[ ]"
				if cert.Status == "certified" {
					icon = "[x]"
				}
				fmt.Printf("- %s **%s:** (%s)\n", icon, cert.ID, cert.Status)
			}
		}

		// 3. Contar Findings
		fmt.Println("\n🐞 **Hallazgos (Findings):**")
		findDir := filepath.Join(qddDir, "findings")
		findEntries, _ := os.ReadDir(findDir)
		
		type Finding struct {
			ID     string `yaml:"id"`
			Title  string `yaml:"title"`
			Status string `yaml:"status"`
		}

		totalFindings := 0
		for _, entry := range findEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				totalFindings++
				content, _ := os.ReadFile(filepath.Join(findDir, entry.Name()))
				var fnd Finding
				yaml.Unmarshal(content, &fnd)
				
				icon := "[ ]"
				if fnd.Status == "resolved" {
					icon = "[x]"
				}
				fmt.Printf("- %s **%s:** %s (%s)\n", icon, fnd.ID, fnd.Title, fnd.Status)
			}
		}

		if totalFindings == 0 {
			fmt.Println("- No hay findings registrados.")
		}

		fmt.Println("\n[✔] Reporte de estado generado exitosamente.")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
