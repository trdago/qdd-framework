package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/topology"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Certification struct {
	ID     string `yaml:"id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`
}

var certifyCmd = &cobra.Command{
	Use:     "certify",
	Aliases: []string{"CERTIFY"},
	Short:   "Ejecuta el proceso de certificación (Fast Path)",
	Long:    `Ejecuta la certificación de manera determinista validando los archivos YAML.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[⚙️ EXECUTOR] Ejecutando comando determinista: CERTIFY")
		fmt.Println("  -> Leyendo reglas en .qdd/core/certification/ y .qdd/project/certification/")

		certDirs := []string{
			filepath.Join(".", ".qdd", "core", "certification"),
			filepath.Join(".", ".qdd", "project", "certification"),
		}

		total := 0
		certified := 0

		for _, certDir := range certDirs {
			entries, err := os.ReadDir(certDir)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
					continue
				}

				content, err := os.ReadFile(filepath.Join(certDir, entry.Name()))
				if err != nil {
					continue
				}

				var cert Certification
				if err := yaml.Unmarshal(content, &cert); err != nil {
					continue
				}

				total++
				icon := "[ ]"
				if cert.Status == "certified" {
					icon = "[✔]"
					certified++
				}

				fmt.Printf("%s %s (%s) - %s\n", icon, cert.ID, cert.Status, cert.Title)
			}
		}

		fmt.Println("  -> Ejecutando validaciones locales (Deuda Técnica y Topología)...")
		
		cwd, _ := os.Getwd()
		fndDir := filepath.Join(cwd, ".qdd", "project", "findings")
		fnds, _ := os.ReadDir(fndDir)
		openFindings := 0
		for _, f := range fnds {
			if !f.IsDir() {
				fndData, _ := os.ReadFile(filepath.Join(fndDir, f.Name()))
				var rawData map[string]interface{}
				yaml.Unmarshal(fndData, &rawData)
				status := fmt.Sprintf("%v", rawData["status"])
				if status != "resolved" && status != "RESOLVED" {
					openFindings++
				}
			}
		}

		top, err := topology.MapProject(cwd)
		topScore := 100
		if err == nil && top != nil {
			topScore = top.GlobalScore
		}

		if total == 0 {
			fmt.Println("[!] No se encontraron certificaciones.")
			return
		}

		if certified == total && openFindings == 0 && topScore == 100 {
			fmt.Printf("[🏆] Proyecto 100%% Certificado (%d/%d reglas cumplidas).\n", certified, total)
			return
		}
		
		fmt.Printf("[⚠️] Proyecto No Certificado (%d/%d reglas cumplidas).\n", certified, total)
		if openFindings > 0 {
			fmt.Printf("  - [!] Deuda Técnica Abierta: %d hallazgos sin resolver.\n", openFindings)
		}
		if topScore < 100 {
			fmt.Printf("  - [!] Deuda de Certificación: Score Topológico al %d%%.\n", topScore)
		}
		os.Exit(1)
	},
}

func init() {
	rootCmd.AddCommand(certifyCmd)
}
