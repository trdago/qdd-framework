package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		fmt.Println("  -> Ejecutando validaciones locales...")
		
		if total == 0 {
			fmt.Println("[!] No se encontraron certificaciones.")
			return
		}

		if certified == total {
			fmt.Printf("[🏆] Proyecto 100%% Certificado (%d/%d reglas cumplidas).\n", certified, total)
			return
		}
		
		fmt.Printf("[⚠️] Proyecto No Certificado (%d/%d reglas cumplidas).\n", certified, total)
	},
}

func init() {
	rootCmd.AddCommand(certifyCmd)
}
