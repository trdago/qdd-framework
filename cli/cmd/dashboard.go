package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/qdd-framework/qdd/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"ui"},
	Short:   "Inicia el Centro de Comando Web (Frontend embebido)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Iniciando QDD Dashboard en http://localhost:8080...")

		// Servir archivos estáticos embebidos
		distFs, err := fs.Sub(ui.StaticFiles, "dist")
		if err != nil {
			fmt.Println("Error accediendo a los archivos estáticos de UI:", err)
			return
		}
		http.Handle("/", http.FileServer(http.FS(distFs)))

		// Endpoint de API REST para la UI
		http.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
			cwd, _ := os.Getwd()
			qddDir := filepath.Join(cwd, ".qdd")

			response := map[string]interface{}{
				"score":          100,
				"grade":          "World-Class",
				"version":        "v0.1.1",
				"findings":       []map[string]interface{}{},
				"certifications": []map[string]interface{}{},
				"sprints":        []map[string]interface{}{},
				"config":         map[string]interface{}{},
				"audit_status":   "",
			}

			// Read state.json
			stateData, err := os.ReadFile(filepath.Join(qddDir, "state.json"))
			if err == nil {
				var state map[string]interface{}
				json.Unmarshal(stateData, &state)
				if ver, ok := state["version"]; ok {
					response["version"] = ver
				}
			}

			// Read config.yaml
			configData, err := os.ReadFile(filepath.Join(qddDir, "config.yaml"))
			if err == nil {
				var config map[string]interface{}
				yaml.Unmarshal(configData, &config)
				response["config"] = config
			}

			// Read Certifications
			certDir := filepath.Join(qddDir, "certification")
			certs, _ := os.ReadDir(certDir)
			for _, c := range certs {
				if !c.IsDir() {
					certPath := filepath.Join(certDir, c.Name())
					certData, _ := os.ReadFile(certPath)
					var rawData map[string]interface{}
					yaml.Unmarshal(certData, &rawData)
					
					status := "PASS"
					if rawData["status"] != nil {
						status = fmt.Sprintf("%v", rawData["status"])
					}

					response["certifications"] = append(response["certifications"].([]map[string]interface{}), map[string]interface{}{
						"id":     c.Name(),
						"status": status,
						"name":   "Cumplimiento verificado",
						"raw":    rawData,
					})
				}
			}

			// Read Findings
			fndDir := filepath.Join(qddDir, "findings")
			fnds, _ := os.ReadDir(fndDir)
			openFindings := 0
			for _, f := range fnds {
				if !f.IsDir() {
					fndPath := filepath.Join(fndDir, f.Name())
					fndData, _ := os.ReadFile(fndPath)
					var rawData map[string]interface{}
					yaml.Unmarshal(fndData, &rawData)

					status := "OPEN"
					if rawData["status"] != nil {
						status = fmt.Sprintf("%v", rawData["status"])
					}
					
					if status != "RESOLVED" && status != "resolved" {
						openFindings++
					}

					response["findings"] = append(response["findings"].([]map[string]interface{}), map[string]interface{}{
						"id":     f.Name(),
						"status": status,
						"desc":   "Deuda técnica documentada.",
						"raw":    rawData,
					})
				}
			}

			// Compute dynamic score
			baseScore := 100
			findingPenalty := openFindings * 30
			finalScore := baseScore - findingPenalty
			if finalScore < 0 {
				finalScore = 0
			}
			response["score"] = finalScore

			// Read Sprints (Avances)
			sprintDir := filepath.Join(qddDir, "sprints")
			sprintsData, _ := os.ReadDir(sprintDir)
			for _, s := range sprintsData {
				if !s.IsDir() {
					sprintPath := filepath.Join(sprintDir, s.Name())
					sprintData, _ := os.ReadFile(sprintPath)
					var rawData map[string]interface{}
					yaml.Unmarshal(sprintData, &rawData)
					
					status := "IN-PROGRESS"
					if rawData["status"] != nil {
						status = fmt.Sprintf("%v", rawData["status"])
					}

					response["sprints"] = append(response["sprints"].([]map[string]interface{}), map[string]interface{}{
						"id":     s.Name(),
						"status": status,
						"raw":    rawData,
					})
				}
			}

			grade := "World-Class"
			if finalScore < 90 { grade = "A" }
			if finalScore < 80 { grade = "B" }
			if finalScore < 70 { grade = "C" }
			if finalScore < 50 { grade = "D (CRITICAL)" }
			response["grade"] = grade

			// Set Audit Status
			auditStatus := "PASS"
			if openFindings > 0 {
				auditStatus = "FAIL (Deuda Técnica Detectada)"
			}
			response["audit_status"] = auditStatus

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})

		port := 8080
		var listener net.Listener
		for {
			addr := fmt.Sprintf(":%d", port)
			listener, err = net.Listen("tcp", addr)
			if err == nil {
				break
			}
			port++
		}

		dashboardURL := fmt.Sprintf("http://localhost:%d", port)
		fmt.Printf("✅ Servidor listo y escuchando en %s\n", dashboardURL)

		go func() {
			http.Serve(listener, nil)
		}()

		if err := openBrowser(dashboardURL); err != nil {
			fmt.Printf("Por favor abre manualmente: %s\n", dashboardURL)
		}

		select {}
	},
}

func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
