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
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/qdd-framework/qdd/pkg/qcl"
	"github.com/qdd-framework/qdd/pkg/qcl/adapters"
	"github.com/qdd-framework/qdd/pkg/qcl/nodes"
	"github.com/qdd-framework/qdd/pkg/topology"
	"github.com/qdd-framework/qdd/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var startTime = time.Now()

type StateBroker struct {
	sync.RWMutex
	clients map[chan []byte]bool
	cache   []byte
}

var broker = &StateBroker{
	clients: make(map[chan []byte]bool),
}

func (b *StateBroker) AddClient() chan []byte {
	b.Lock()
	defer b.Unlock()
	ch := make(chan []byte, 1)
	if len(b.cache) > 0 {
		ch <- b.cache
	}
	b.clients[ch] = true
	return ch
}

func (b *StateBroker) RemoveClient(ch chan []byte) {
	b.Lock()
	defer b.Unlock()
	delete(b.clients, ch)
	close(ch)
}

func (b *StateBroker) Broadcast(data []byte) {
	b.Lock()
	defer b.Unlock()
	b.cache = data
	for ch := range b.clients {
		select {
		case ch <- data:
		default:
		}
	}
}

func runWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creando fsnotify:", err)
		return
	}
	defer watcher.Close()

	cwd, _ := os.Getwd()
	qddDir := filepath.Join(cwd, ".qdd")

	filepath.Walk(qddDir, func(path string, info fs.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	res := buildState()
	data, _ := json.Marshal(res)
	broker.Broadcast(data)

	var timer *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(500*time.Millisecond, func() {
					res := buildState()
					data, _ := json.Marshal(res)
					broker.Broadcast(data)
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Watcher error:", err)
		}
	}
}

// Definición estricta del Contrato OpenAPI (Type-Safety)
type DashboardFinding struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Desc   string                 `json:"desc"`
	Raw    map[string]interface{} `json:"raw"`
}

type DashboardCertification struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Raw    map[string]interface{} `json:"raw"`
}

type DashboardSprint struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Raw    map[string]interface{} `json:"raw"`
}

type DashboardKnowledgeDoc struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type DashboardTelemetry struct {
	Uptime       string `json:"uptime"`
	MemoryAlloc  string `json:"memory_alloc"`
	MemorySys    string `json:"memory_sys"`
	Goroutines   int    `json:"goroutines"`
}

type DashboardUnderstanding struct {
	Summary     string   `json:"summary"`
	Components  []string `json:"components"`
	Objectives  []string `json:"objectives"`
	Guidelines  []string `json:"guidelines"`
	NextSteps   string   `json:"next_steps"`
}

type QDDState struct {
	Score          int                      `json:"score"`
	Grade          string                   `json:"grade"`
	Version        string                   `json:"version"`
	AuditStatus    string                   `json:"audit_status"`
	Findings       []DashboardFinding       `json:"findings"`
	Certifications []DashboardCertification `json:"certifications"`
	Sprints        []DashboardSprint        `json:"sprints"`
	Knowledge      []DashboardKnowledgeDoc  `json:"knowledge"`
	Understanding  *DashboardUnderstanding  `json:"understanding"`
	Topology       *topology.ProjectTopology `json:"topology"`
	Config         map[string]interface{}   `json:"config"`
	Telemetry      DashboardTelemetry       `json:"telemetry"`
	WorkingOn      string                   `json:"working_on"`
}

func buildState() QDDState {
	cwd, _ := os.Getwd()
	qddDir := filepath.Join(cwd, ".qdd")

	response := QDDState{
		Score:          100,
		Grade:          "World-Class",
		Version:        "v0.1.1",
		AuditStatus:    "PASS",
		Findings:       []DashboardFinding{},
		Certifications: []DashboardCertification{},
		Sprints:        []DashboardSprint{},
		Knowledge:      []DashboardKnowledgeDoc{},
		Config:         make(map[string]interface{}),
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	response.Telemetry = DashboardTelemetry{
		Uptime:      time.Since(startTime).Round(time.Second).String(),
		MemoryAlloc: fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024),
		MemorySys:   fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024),
		Goroutines:  runtime.NumGoroutine(),
	}

	// Read state.json
	stateData, err := os.ReadFile(filepath.Join(qddDir, "state.json"))
	if err == nil {
		var state map[string]interface{}
		json.Unmarshal(stateData, &state)
		if ver, ok := state["version"]; ok {
			response.Version = fmt.Sprintf("%v", ver)
		}
	}

	workData, err := os.ReadFile(filepath.Join(qddDir, "working"))
	if err == nil {
		response.WorkingOn = strings.TrimSpace(string(workData))
	}

	// Read config.yaml
	configData, err := os.ReadFile(filepath.Join(qddDir, "config.yaml"))
	if err == nil {
		var config map[string]interface{}
		yaml.Unmarshal(configData, &config)
		response.Config = config

		// Parse documentation_index for knowledge base
		if docs, ok := config["documentation_index"].([]interface{}); ok {
			for _, docPath := range docs {
				if p, ok := docPath.(string); ok {
					content, err := os.ReadFile(filepath.Join(cwd, p))
					if err == nil {
						response.Knowledge = append(response.Knowledge, DashboardKnowledgeDoc{
							ID:      filepath.Base(p),
							Path:    p,
							Content: string(content),
						})
					}
				}
			}
		}
	}

	// Read Understanding
	undData, err := os.ReadFile(filepath.Join(qddDir, "understanding.json"))
	if err == nil {
		var und DashboardUnderstanding
		if err := json.Unmarshal(undData, &und); err == nil {
			response.Understanding = &und
		}
	}

	// Load Topology
	// Load Topology
	topData, errTop := os.ReadFile(filepath.Join(qddDir, "project", "topology.json"))
	if errTop != nil {
		// Auto-map if not exists
		if top, mapErr := topology.MapProject(cwd); mapErr == nil {
			response.Topology = top
		}
	}
	if errTop == nil {
		var top topology.ProjectTopology
		if err := json.Unmarshal(topData, &top); err == nil {
			response.Topology = &top
		}
	}

	// Read Certifications
	certDirs := []string{
		filepath.Join(qddDir, "core", "certification"),
		filepath.Join(qddDir, "project", "certification"),
	}

	for _, certDir := range certDirs {
		certs, _ := os.ReadDir(certDir)
		for _, c := range certs {
			if !c.IsDir() && strings.HasSuffix(c.Name(), ".yaml") {
				certPath := filepath.Join(certDir, c.Name())
				certData, _ := os.ReadFile(certPath)
				var rawData map[string]interface{}
				yaml.Unmarshal(certData, &rawData)
				
				status := "PASS"
				if rawData != nil && rawData["status"] != nil {
					status = fmt.Sprintf("%v", rawData["status"])
				}

				certType := "Core"
				if strings.Contains(certDir, "project") {
					certType = "Proyecto"
				}

				response.Certifications = append(response.Certifications, DashboardCertification{
					ID:     c.Name(),
					Status: status,
					Name:   "Cumplimiento verificado",
					Type:   certType,
					Raw:    rawData,
				})
			}
		}
	}

	// Read Findings
	fndDir := filepath.Join(qddDir, "project", "findings")
	fnds, _ := os.ReadDir(fndDir)
	openFindings := 0
	for _, f := range fnds {
		if !f.IsDir() {
			fndPath := filepath.Join(fndDir, f.Name())
			fndData, _ := os.ReadFile(fndPath)
			var rawData map[string]interface{}
			yaml.Unmarshal(fndData, &rawData)

			status := "OPEN"
			if rawData != nil && rawData["status"] != nil {
				status = fmt.Sprintf("%v", rawData["status"])
			}
			
			if status != "RESOLVED" && status != "resolved" {
				openFindings++
			}

			response.Findings = append(response.Findings, DashboardFinding{
				ID:     f.Name(),
				Status: status,
				Desc:   "Deuda técnica documentada.",
				Raw:    rawData,
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
	response.Score = finalScore

	// Read Sprints (Avances)
	sprintDir := filepath.Join(qddDir, "project", "sprints")
	sprintsData, _ := os.ReadDir(sprintDir)
	for _, s := range sprintsData {
		if !s.IsDir() {
			sprintPath := filepath.Join(sprintDir, s.Name())
			sprintData, _ := os.ReadFile(sprintPath)
			var rawData map[string]interface{}
			yaml.Unmarshal(sprintData, &rawData)
			
			status := "IN-PROGRESS"
			if rawData != nil && rawData["status"] != nil {
				status = fmt.Sprintf("%v", rawData["status"])
				response.Sprints = append(response.Sprints, DashboardSprint{
					ID:     s.Name(),
					Status: status,
					Raw:    rawData,
				})
				continue
			}

			// Infer from markdown checkboxes
			contentStr := string(sprintData)
			if strings.Contains(contentStr, "- [x]") && !strings.Contains(contentStr, "- [ ]") {
				status = "COMPLETED"
			}
			if !strings.Contains(contentStr, "- [x]") && !strings.Contains(contentStr, "- [ ]") {
				status = "BACKLOG"
			}

			response.Sprints = append(response.Sprints, DashboardSprint{
				ID:     s.Name(),
				Status: status,
				Raw:    rawData,
			})
		}
	}

	grade := "World-Class"
	if finalScore < 90 { grade = "A" }
	if finalScore < 80 { grade = "B" }
	if finalScore < 70 { grade = "C" }
	if finalScore < 50 { grade = "D (CRITICAL)" }
	response.Grade = grade

	// Set Audit Status
	auditStatus := "PASS"
	if openFindings > 0 {
		auditStatus = "FAIL (Deuda Técnica Detectada)"
	}
	response.AuditStatus = auditStatus

	return response
}

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"ui"},
	Short:   "Inicia el Centro de Comando Web (Frontend embebido)",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Iniciando QDD Dashboard en http://localhost:8080...")

		go runWatcher()

		// Servir archivos estáticos embebidos
		distFs, err := fs.Sub(ui.StaticFiles, "dist")
		if err != nil {
			fmt.Println("Error accediendo a los archivos estáticos de UI:", err)
			return
		}
		http.Handle("/", http.FileServer(http.FS(distFs)))

		// Endpoint de API REST (Legacy/Fallback)
		http.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
			response := buildState()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})

		// Endpoint SSE (Server-Sent Events) para Real-Time con Contrato Estricto
		http.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "SSE not supported", http.StatusInternalServerError)
				return
			}

			ch := broker.AddClient()
			defer broker.RemoveClient(ch)

			for {
				select {
				case <-r.Context().Done():
					return
				case data := <-ch:
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
				}
			}
		})

		type IntentRequest struct {
			Input string `json:"input"`
		}

		http.HandleFunc("/api/intent", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req IntentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			engine := adapters.NewGeminiEngine()
			
			pipeline := qcl.NewPipeline(
				nodes.NewIntentAnalyzer(engine),
				&nodes.ContextAnalyzer{},
				nodes.NewStrategyPlanner(engine),
				nodes.NewPlanBuilder(engine),
			)

			session, err := pipeline.Execute(req.Input)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(session)
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
