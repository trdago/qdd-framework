package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var scoreCmd = &cobra.Command{
	Use:     "score",
	Aliases: []string{"SCORE"},
	Short:   "Calcula y muestra el puntaje de calidad del proyecto",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("--- 🏆 QDD QUALITY SCORE ---")

		qddDir := filepath.Join(".", ".qdd")
		
		// Certificaciones
		certDirs := []string{
			filepath.Join(qddDir, "core", "certification"),
			filepath.Join(qddDir, "project", "certification"),
		}
		
		totalCerts := 0
		certifiedCerts := 0

		for _, certDir := range certDirs {
			entries, _ := os.ReadDir(certDir)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
					totalCerts++
					content, _ := os.ReadFile(filepath.Join(certDir, entry.Name()))
					var cert Certification
					yaml.Unmarshal(content, &cert)
					if cert.Status == "certified" {
						certifiedCerts++
					}
				}
			}
		}

		// Findings
		findDir := filepath.Join(qddDir, "project", "findings")
		findEntries, _ := os.ReadDir(findDir)
		
		type Finding struct {
			Status string `yaml:"status"`
		}

		openFindings := 0
		resolvedFindings := 0
		for _, entry := range findEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				content, _ := os.ReadFile(filepath.Join(findDir, entry.Name()))
				var fnd Finding
				yaml.Unmarshal(content, &fnd)
				if fnd.Status == "open" {
					openFindings++
					continue
				}
				if fnd.Status == "resolved" {
					resolvedFindings++
				}
			}
		}

		// Calcular Puntaje
		baseScore := 100
		
		// Penalizaciones
		pendingCerts := totalCerts - certifiedCerts
		certPenalty := pendingCerts * 20
		findingPenalty := openFindings * 30
		
		finalScore := baseScore - certPenalty - findingPenalty
		if finalScore < 0 {
			finalScore = 0
		}

		grado := "F"
		if finalScore >= 95 {
			grado = "A+ (World-Class)"
		}
		if finalScore >= 80 && finalScore < 95 {
			grado = "B (Bueno)"
		}
		if finalScore >= 60 && finalScore < 80 {
			grado = "C (Requiere Atención)"
		}
		if finalScore >= 40 && finalScore < 60 {
			grado = "D (Pobre)"
		}

		fmt.Printf("\n**Puntaje Global:** `%d / 100` (Grado: %s)\n", finalScore, grado)
		fmt.Println("\n📈 **Desglose del Puntaje:**")
		fmt.Printf("* **Certificaciones Cumplidas:** %d/%d `(+%d pts)`\n", certifiedCerts, totalCerts, certifiedCerts*20)
		fmt.Printf("* **Deuda Técnica (Findings Abiertos):** %d `(-%d pts)`\n", openFindings, findingPenalty)
		fmt.Printf("* **Findings Resueltos:** %d\n", resolvedFindings)
		fmt.Printf("* **Certificaciones Pendientes (Riesgo):** %d `(-%d pts potenciales)`\n", pendingCerts, certPenalty)
		
		if finalScore >= 95 {
			fmt.Println("\n💡 **Recomendación del Framework:**\n¡Excelente trabajo! El código cumple con las métricas World-Class.")
			return
		}
		
		fmt.Println("\n💡 **Recomendación del Framework:**\nDebes resolver los findings abiertos y completar las certificaciones pendientes para subir tu grado.")
	},
}

func init() {
	rootCmd.AddCommand(scoreCmd)
}
