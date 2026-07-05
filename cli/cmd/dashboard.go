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
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
	"github.com/qdd-framework/qdd/pkg/qcl/nodes"
	"github.com/qdd-framework/qdd/pkg/topology"
	"github.com/qdd-framework/qdd/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var startTime = time.Now()

const qddLifecycleMermaid = "```mermaid\ngraph TD\n" +
`    classDef default fill:#1e1e1e,stroke:#3b82f6,stroke-width:2px,color:#fff;
    classDef init fill:#6366f1,stroke:#4338ca,stroke-width:2px,color:#fff;
    classDef agent fill:#ec4899,stroke:#be185d,stroke-width:2px,color:#fff;
    classDef gatekeeper fill:#14b8a6,stroke:#0f766e,stroke-width:2px,color:#fff;
    classDef success fill:#22c55e,stroke:#15803d,stroke-width:2px,color:#fff;
    classDef warning fill:#f59e0b,stroke:#b45309,stroke-width:2px,color:#fff;

    A[qdd init<br/>Crea Entorno y Wisdom Registry]:::init
    B[qdd sprint<br/>Define Requerimientos]:::default
    C[qdd 'prompt'<br/>Delegación a IA]:::agent
    D{Gatekeeper<br/>Pre-Flight Check}:::gatekeeper
    E[qdd learn<br/>Absorber Arquitectura e Intelligence Report]:::default
    F[Modo Consultivo<br/>Propuesta de Estándares]:::agent
    G[qdd audit<br/>Inspección Técnica]:::warning
    H[qdd certify<br/>Sello de Gobernanza]:::success
    I[qdd release<br/>Git Tag / Deploy]:::success

    A --> B
    B --> C
    C --> D
    D -- Contexto Incompleto --> E
    E --> C
    D -- Autorizado --> F
    F --> G
    G -- Fallo Técnico --> C
    G -- Reglas Cumplidas --> H
    H --> I
    I --> B
` + "```\n"

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
	ID      string                 `json:"id"`
	Version string                 `json:"version"`
	Status  string                 `json:"status"`
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Raw     map[string]interface{} `json:"raw"`
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
	ProjectName    string                   `json:"project_name"`
}

func buildState() QDDState {
	cwd, _ := os.Getwd()
	qddDir := filepath.Join(cwd, ".qdd")

	response := QDDState{
		Score:          100,
		Grade:          "World-Class",
		Version:        "v0.1.1",
		AuditStatus:    "PASS",
		ProjectName:    filepath.Base(cwd),
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

	hasLifecycle := false
	for _, k := range response.Knowledge {
		if k.Path == "docs/command-reference.md" {
			hasLifecycle = true
			break
		}
	}
	if !hasLifecycle {
		response.Knowledge = append(response.Knowledge, DashboardKnowledgeDoc{
			ID:      "command-reference.md",
			Path:    "docs/command-reference.md",
			Content: qddLifecycleMermaid,
		})
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

	openFindings := 0
	finalScore := 100

	dbGraph, errDB := graph.InitDB()
	if errDB == nil {
		defer dbGraph.Close()
		dbGraph.SyncToGraph(cwd)
		db := dbGraph.GetDB()

		// Read Certifications (rules)
		rows, err := db.Query("SELECT id, name, metadata FROM nodes WHERE type = 'rule'")
		if err == nil {
			for rows.Next() {
				var id, name, metaStr string
				rows.Scan(&id, &name, &metaStr)
				var rawData map[string]interface{}
				json.Unmarshal([]byte(metaStr), &rawData)

				status := "PASS"
				if rawData != nil && rawData["status"] != nil {
					status = fmt.Sprintf("%v", rawData["status"])
				}

				version := "unknown"
				if rawData != nil && rawData["version"] != nil {
					version = fmt.Sprintf("%v", rawData["version"])
				}

				response.Certifications = append(response.Certifications, DashboardCertification{
					ID:      name,
					Version: version,
					Status:  status,
					Name:    "Cumplimiento verificado",
					Type:    "Proyecto",
					Raw:     rawData,
				})
			}
			rows.Close()
		}

		// Read Findings
		rows, err = db.Query("SELECT id, name, metadata FROM nodes WHERE type = 'finding'")
		if err == nil {
			for rows.Next() {
				var id, name, metaStr string
				rows.Scan(&id, &name, &metaStr)
				var rawData map[string]interface{}
				json.Unmarshal([]byte(metaStr), &rawData)

				status := "OPEN"
				if rawData != nil && rawData["status"] != nil {
					status = fmt.Sprintf("%v", rawData["status"])
				}

				if status != "RESOLVED" && status != "resolved" {
					openFindings++
				}

				response.Findings = append(response.Findings, DashboardFinding{
					ID:     name,
					Status: status,
					Desc:   "Deuda técnica documentada.",
					Raw:    rawData,
				})
			}
			rows.Close()
		}

		// Compute dynamic score
		baseScore := 100
		findingPenalty := openFindings * 30
		finalScore = baseScore - findingPenalty
		if finalScore < 0 {
			finalScore = 0
		}
		response.Score = finalScore

		// Read Sprints (tasks)
		rows, err = db.Query("SELECT id, name, content, metadata FROM nodes WHERE type = 'task'")
		if err == nil {
			for rows.Next() {
				var id, name, contentStr, metaStr string
				rows.Scan(&id, &name, &contentStr, &metaStr)
				var rawData map[string]interface{}
				json.Unmarshal([]byte(metaStr), &rawData)

				status := "IN-PROGRESS"
				if rawData != nil && rawData["status"] != nil {
					status = fmt.Sprintf("%v", rawData["status"])
				}
				
				if rawData == nil || rawData["status"] == nil {
					if strings.Contains(contentStr, "- [x]") && !strings.Contains(contentStr, "- [ ]") {
						status = "COMPLETED"
					}
					if !strings.Contains(contentStr, "- [x]") && !strings.Contains(contentStr, "- [ ]") {
						status = "BACKLOG"
					}
				}

				response.Sprints = append(response.Sprints, DashboardSprint{
					ID:     name,
					Status: status,
					Raw:    rawData,
				})
			}
			rows.Close()
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
				nodes.NewContextAnalyzer(),
				nodes.NewRiskAnalyzer(),
				nodes.NewConsultativeNode(),
				nodes.NewStrategyPlanner(engine),
				nodes.NewPlanBuilder(engine),
				nodes.NewApprovalManager(),
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
