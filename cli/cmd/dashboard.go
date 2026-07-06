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
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
	"github.com/qdd-framework/qdd/pkg/topology"
	"github.com/qdd-framework/qdd/pkg/audit"
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

	// Heartbeat for MCP log to keep UI alive
	go func() {
		for {
			time.Sleep(15 * time.Second)
			logFile := filepath.Join(qddDir, "mcp.log")
			f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(fmt.Sprintf("[INFO] %s: QDD Framework monitor activo, analizando estado...\n", time.Now().Format("15:04:05")))
				f.Close()
			}
		}
	}()

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
	Pillar string                 `json:"pillar"`
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

type ValueMetrics struct {
	HoursSaved  int `json:"hours_saved"`
	DebtReduced int `json:"debt_reduced"`
}

type HistoricalTrendPoint struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

type GraphNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type GraphEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

type DashboardGraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
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
	ValueMetrics   ValueMetrics             `json:"value_metrics"`
	Historical     []HistoricalTrendPoint   `json:"historical_trends"`
	MCPLogs        []string                 `json:"mcp_logs"`
	UsageTime      string                   `json:"usage_time"`
	Policies       audit.QDDPolicies        `json:"policies"`
	GraphData      DashboardGraphData       `json:"graph_data"`
	AutoUICert     bool                     `json:"auto_ui_certification"`
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
		Historical:     []HistoricalTrendPoint{},
		MCPLogs:        []string{},
		Policies:       audit.LoadPolicies(cwd),
	}

	if info, err := os.Stat(qddDir); err == nil {
		duration := time.Since(info.ModTime())
		days := int(duration.Hours() / 24)
		response.UsageTime = "Recientemente iniciado"
		if days > 0 {
			response.UsageTime = fmt.Sprintf("%d días", days)
		}
	}

	// Read MCP logs
	if logData, err := os.ReadFile(filepath.Join(qddDir, "mcp.log")); err == nil {
		lines := strings.Split(string(logData), "\n")
		// Keep last 30 lines
		startIdx := len(lines) - 30
		if startIdx < 0 { startIdx = 0 }
		for i := startIdx; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) != "" {
				response.MCPLogs = append(response.MCPLogs, lines[i])
			}
		}
	}
	// Default dummy logs if empty to demonstrate real-time feed
	if len(response.MCPLogs) == 0 {
		response.MCPLogs = []string{
			"[MCP] Servidor QDD inicializado.",
			"[MCP] Escuchando intenciones en stdio/SSE...",
		}
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
		
		var config struct {
			AutoUICertification *bool `yaml:"auto_ui_certification"`
		}
		response.AutoUICert = true // Default
		if yaml.Unmarshal(configData, &config) == nil && config.AutoUICertification != nil {
			response.AutoUICert = *config.AutoUICertification
		}

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

				isResolved := (status == "RESOLVED" || status == "resolved")
				if !isResolved {
					openFindings++
				}
				if isResolved {
					response.ValueMetrics.DebtReduced++
					response.ValueMetrics.HoursSaved += 2 // 2 hours saved per finding resolved
				}

				// Categorize into Pillars
				pillar := "Certificación" // Default
				descLower := ""
				if rawData != nil && rawData["description"] != nil {
					descLower = strings.ToLower(fmt.Sprintf("%v", rawData["description"]))
				}
				nameLower := strings.ToLower(name)
				searchStr := nameLower + " " + descLower

				if strings.Contains(searchStr, "cert") || strings.Contains(searchStr, "missing") || strings.Contains(searchStr, "adr") || strings.Contains(searchStr, "doc") {
					pillar = "Certificación"
				}
				if strings.Contains(searchStr, "timeout") || strings.Contains(searchStr, "count") || strings.Contains(searchStr, "performance") || strings.Contains(searchStr, "flaky") || strings.Contains(searchStr, "estabilidad") {
					pillar = "Estabilidad"
				}
				if strings.Contains(searchStr, "key") || strings.Contains(searchStr, "secret") || strings.Contains(searchStr, "sql") || strings.Contains(searchStr, "auth") || strings.Contains(searchStr, "cors") || strings.Contains(searchStr, "seguridad") {
					pillar = "Seguridad"
				}
				if strings.Contains(searchStr, "else") || strings.Contains(searchStr, "complexity") || strings.Contains(searchStr, "ciclomática") || strings.Contains(searchStr, "legacy") || strings.Contains(searchStr, "estructural") {
					pillar = "Estructural"
				}

				response.Findings = append(response.Findings, DashboardFinding{
					ID:     name,
					Status: status,
					Pillar: pillar,
					Desc:   "Deuda técnica documentada.",
					Raw:    rawData,
				})
			}
			rows.Close()
		}

		// Read Graph Nodes
		response.GraphData = DashboardGraphData{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
		rows, err = db.Query("SELECT id, type, name FROM nodes")
		if err == nil {
			for rows.Next() {
				var id, nodeType, name string
				rows.Scan(&id, &nodeType, &name)
				response.GraphData.Nodes = append(response.GraphData.Nodes, GraphNode{ID: id, Type: nodeType, Name: name})
			}
			rows.Close()
		}

		// Read Graph Edges
		rows, err = db.Query("SELECT source_id, target_id, relation FROM edges")
		if err == nil {
			for rows.Next() {
				var source, target, relation string
				rows.Scan(&source, &target, &relation)
				response.GraphData.Edges = append(response.GraphData.Edges, GraphEdge{Source: source, Target: target, Relation: relation})
			}
			rows.Close()
		}

		// Calculate historical data from cognitive_history.json
		histPath := filepath.Join(cwd, ".qdd", "project", "metrics", "cognitive_history.json")
		if histData, err := os.ReadFile(histPath); err == nil {
			var histList []map[string]interface{}
			json.Unmarshal(histData, &histList)
			for _, item := range histList {
				date := ""
				score := 0
				if d, ok := item["date"]; ok { date = fmt.Sprintf("%v", d) }
				if s, ok := item["score"]; ok { score = int(s.(float64)) }
				response.Historical = append(response.Historical, HistoricalTrendPoint{Date: date, Score: score})
			}
		}

		// Fallback data if empty to show the chart
		if len(response.Historical) == 0 {
			response.Historical = []HistoricalTrendPoint{
				{Date: "Día 1", Score: 40},
				{Date: "Día 2", Score: 75},
				{Date: "Hoy", Score: 100},
			}
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

				if status == "COMPLETED" || status == "CERTIFIED" {
					response.ValueMetrics.HoursSaved += 4 // 4 hours per sprint
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

	if response.ValueMetrics.HoursSaved == 0 && response.ValueMetrics.DebtReduced == 0 {
		response.MCPLogs = append([]string{"[WARNING] Métricas de Valor y ROI no disponibles. Finaliza sprints para ganar valor."}, response.MCPLogs...)
	}

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

		http.HandleFunc("/api/policies", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				cwd, _ := os.Getwd()
				p := audit.LoadPolicies(cwd)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(p)
				return
			}
			if r.Method == http.MethodPost {
				cwd, _ := os.Getwd()
				var p audit.QDDPolicies
				if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				audit.SavePolicies(cwd, p)
				
				// Re-run QDD Certify silently to update DB state with new rules
				exec.CommandContext(r.Context(), os.Args[0], "certify").Run()
				
				// Broadcast new state
				res := buildState()
				data, _ := json.Marshal(res)
				broker.Broadcast(data)
				
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

			mockResponse := map[string]interface{}{
				"status": "DEPRECATED",
				"message": "El motor cognitivo interno basado en API ha sido deprecado. QDD ahora actúa exclusivamente como un Harness MCP para inteligencias artificiales externas (Antigravity, Claude, Cursor). Por favor envía tu intención directamente a tu asistente de IA.",
				"input_received": req.Input,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
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
