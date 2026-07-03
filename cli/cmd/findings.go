package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Finding struct {
	ID       string `yaml:"id"`
	Type     string `yaml:"type"`
	Title    string `yaml:"title"`
	Status   string `yaml:"status"`
	Severity string `yaml:"severity"`
}

var findingsCmd = &cobra.Command{
	Use:     "findings",
	Aliases: []string{"FINDINGS"},
	Short:   "Muestra todos los hallazgos técnicos (bugs, vulnerabilidades) del proyecto",
	Run: func(cmd *cobra.Command, args []string) {
		qddDir := filepath.Join(".", ".qdd", "project", "findings")
		
		if _, err := os.Stat(qddDir); os.IsNotExist(err) {
			fmt.Println("[-] No se encontró el directorio de findings (.qdd/project/findings).")
			return
		}

		files, err := os.ReadDir(qddDir)
		if err != nil {
			fmt.Printf("[!] Error leyendo directorio de findings: %v\n", err)
			return
		}

		fmt.Println("--- QDD FINDINGS REPORT ---")
		fmt.Println()

		total := 0
		resolved := 0
		pending := 0

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			path := filepath.Join(qddDir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var f Finding
			if err := yaml.Unmarshal(content, &f); err != nil {
				fmt.Printf("[!] Archivo %s tiene formato inválido\n", file.Name())
				continue
			}

			total++
			icon := "🔴"
			statusLower := strings.ToLower(f.Status)
			
			if statusLower == "resolved" || statusLower == "closed" {
				icon = "🟢"
				resolved++
				fmt.Printf("%s [%s] [%s] %s - %s\n", icon, f.ID, strings.ToUpper(f.Severity), strings.ToUpper(f.Status), f.Title)
				continue
			}

			pending++
			fmt.Printf("%s [%s] [%s] %s - %s\n", icon, f.ID, strings.ToUpper(f.Severity), strings.ToUpper(f.Status), f.Title)
		}

		if total == 0 {
			fmt.Println("[-] No hay hallazgos registrados en el proyecto.")
			return
		}

		fmt.Println("\n-------------------------------")
		fmt.Printf("Total: %d | Resueltos: %d | Pendientes: %d\n", total, resolved, pending)
	},
}

func init() {
	rootCmd.AddCommand(findingsCmd)
}
