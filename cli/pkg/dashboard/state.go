package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/cognitive"
	"github.com/qdd-framework/qdd/pkg/findings"
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
	"github.com/qdd-framework/qdd/pkg/topology"
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
    B[qdd_sprint MCP<br/>Define Requerimientos]:::default
    C[qdd 'prompt'<br/>Delegación a IA]:::agent
    D{Gatekeeper<br/>Pre-Flight Check}:::gatekeeper
    E[qdd_learn MCP<br/>Absorber Arquitectura e Intelligence Report]:::default
    F[Modo Consultivo<br/>Propuesta de Estándares]:::agent
    G[qdd audit<br/>Inspección Técnica]:::warning
    H[qdd certify<br/>Sello de Gobernanza]:::success
    I[qdd_release MCP<br/>Git Tag]:::success

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

var Broker = &StateBroker{
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

// Ultimo devuelve el último estado difundido, para servir a un cliente nuevo
// sin reconstruir.
func (b *StateBroker) Ultimo() ([]byte, bool) {
	b.RLock()
	defer b.RUnlock()
	return b.cache, len(b.cache) > 0
}

// ventanaDeEventos agrupa una ráfaga de cambios (un editor que guarda, un git
// checkout) en una sola reconstrucción. No hace falta más precisión.
const ventanaDeEventos = time.Second

// RunWatcher vigila el proyecto del directorio actual mientras viva el proceso.
func RunWatcher() {
	cwd, _ := os.Getwd()
	Vigilar(context.Background(), cwd, nil)
}

// Vigilar difunde el estado del proyecto cada vez que cambia una de sus fuentes
// en .qdd/. Es dirigido por eventos: con el proyecto quieto no reconstruye,
// no escribe y no envía nada (FND-058). listo (opcional) se cierra cuando el
// watcher ya observa y el estado inicial se difundió.
func Vigilar(ctx context.Context, cwd string, listo chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creando fsnotify:", err)
		return
	}
	defer watcher.Close()

	observarArbol(watcher, filepath.Join(cwd, ".qdd"))
	difundirEstado()
	if listo != nil {
		close(listo)
	}

	var pendiente *time.Timer
	defer func() {
		if pendiente != nil {
			pendiente.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("Watcher error:", err)
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !esEventoDelProyecto(event) {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				observarArbol(watcher, event.Name)
			}
			if pendiente != nil {
				pendiente.Stop()
			}
			pendiente = time.AfterFunc(ventanaDeEventos, difundirEstado)
		}
	}
}

// observarArbol agrega al watcher una carpeta y sus subcarpetas (fsnotify no es
// recursivo). Si la ruta es un archivo, no hace nada.
func observarArbol(watcher *fsnotify.Watcher, raiz string) {
	filepath.Walk(raiz, func(path string, info fs.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})
}

func difundirEstado() {
	data, _ := json.Marshal(BuildState())
	Broker.Broadcast(data)
}

// esEventoDelProyecto descarta lo que escribe el propio dashboard (el grafo en
// knowledge.db) y los temporales de editores y escrituras atómicas: si contaran,
// cada reconstrucción dispararía la siguiente.
func esEventoDelProyecto(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}
	nombre := filepath.Base(event.Name)
	if strings.HasPrefix(nombre, "knowledge.db") {
		return false
	}
	for _, sufijo := range []string{".tmp", ".swp", ".swx", "~"} {
		if strings.HasSuffix(nombre, sufijo) {
			return false
		}
	}
	return true
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
	Name    string                 `json:"name"`
	Version string                 `json:"version"`
	Status  string                 `json:"status"`
	Type    string                 `json:"type"`
	Raw     map[string]interface{} `json:"raw,omitempty"`
	History []CertificationRun     `json:"history,omitempty"`
}

type CertificationRun struct {
	RunID     string `json:"run_id"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
	Duration  string `json:"duration"`
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
	Uptime      string `json:"uptime"`
	MemoryAlloc string `json:"memory_alloc"`
	MemorySys   string `json:"memory_sys"`
	Goroutines  int    `json:"goroutines"`
}

type DashboardUnderstanding struct {
	Summary    string   `json:"summary"`
	Components []string `json:"components"`
	Objectives []string `json:"objectives"`
	Guidelines []string `json:"guidelines"`
	NextSteps  string   `json:"next_steps"`
}

// ValueMetrics solo contiene lo que se puede contar en el proyecto. Las
// "horas ahorradas" se calculaban con constantes y se quitaron (FND-055).
type ValueMetrics struct {
	DebtReduced int `json:"debt_reduced"`
}

type HistoricalTrendPoint struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

type GraphNode struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	Content string                 `json:"content"`
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
	Score          int                       `json:"score"`
	Grade          string                    `json:"grade"`
	Version        string                    `json:"version"`
	QualityStatus  string                    `json:"quality_status"`
	Findings       []DashboardFinding        `json:"findings"`
	Certifications []DashboardCertification  `json:"certifications"`
	Sprints        []DashboardSprint         `json:"sprints"`
	Knowledge      []DashboardKnowledgeDoc   `json:"knowledge"`
	Understanding  *DashboardUnderstanding   `json:"understanding"`
	Topology       *topology.ProjectTopology `json:"topology"`
	Config         map[string]interface{}    `json:"config"`
	Telemetry      DashboardTelemetry        `json:"telemetry"`
	WorkingOn      string                    `json:"working_on"`
	ProjectName    string                    `json:"project_name"`
	ValueMetrics   ValueMetrics              `json:"value_metrics"`
	Historical     []HistoricalTrendPoint    `json:"historical_trends"`
	MCPLogs        []string                  `json:"mcp_logs"`
	UsageTime      string                    `json:"usage_time"`
	Policies       audit.QDDPolicies         `json:"policies"`
	GraphData      DashboardGraphData        `json:"graph_data"`
	LocalLLM       cognitive.LocalLLMInfo    `json:"local_llm"`
	AutoUICert     bool                      `json:"auto_ui_certification"`
	FindingsConteo findings.Conteo           `json:"findings_conteo"`
}

func BuildState() QDDState {
	cwd, _ := os.Getwd()
	qddDir := filepath.Join(cwd, ".qdd")

	response := initDefaultState(cwd)

	updateUsageTime(&response, qddDir)

	loadMCPLogs(&response, qddDir)
	loadTelemetry(&response)
	loadStateJson(&response, qddDir)
	loadWorkingOn(&response, qddDir)
	loadConfigAndKnowledge(&response, cwd, qddDir)
	loadUnderstandingAndConfig(&response, qddDir)
	loadTopology(&response, cwd, qddDir)

	loadGraphData(&response, cwd)
	response.FindingsConteo = findings.ContarEn(cwd)

	loadHistoricalTrends(&response, cwd)

	score, grade, _ := puntajeDelDashboard(cwd)
	response.Score = score
	response.Grade = grade
	response.QualityStatus = determineQualityStatus(response.FindingsConteo.Abiertos)
	response.LocalLLM = cognitive.DetectLocalLLM()

	if response.ValueMetrics.DebtReduced == 0 {
		response.MCPLogs = append([]string{"[FILOSOFÍA QDD] 'Lo que no se mide no se mejora'. Finaliza sprints y certificaciones para generar métricas de valor cuantificables."}, response.MCPLogs...)
	}

	return response
}

func updateUsageTime(response *QDDState, qddDir string) {
	if info, err := os.Stat(qddDir); err == nil {
		duration := time.Since(info.ModTime())
		days := int(duration.Hours() / 24)
		response.UsageTime = "Recientemente iniciado"
		if days > 0 {
			response.UsageTime = fmt.Sprintf("%d días", days)
		}
	}
}

func loadGraphData(response *QDDState, cwd string) (int, int) {
	openFindings, finalScore := 0, 100
	dbGraph, errDB := graph.InitDB()
	if errDB == nil {
		defer dbGraph.Close()
		dbGraph.SyncToGraph(cwd)
		db := dbGraph.GetDB()

		loadCertifications(response, db)
		openFindings = loadFindings(response, db)
		loadGraphNodes(response, db)
		loadGraphEdges(response, db)
		loadSprints(response, db)
	}
	return openFindings, finalScore
}

func initDefaultState(cwd string) QDDState {
	return QDDState{
		Score:          0,
		Grade:          gradoSinMedir,
		Version:        "v0.1.1",
		QualityStatus:  "PASS",
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
}

func loadMCPLogs(response *QDDState, qddDir string) {
	if logData, err := os.ReadFile(filepath.Join(qddDir, "mcp.log")); err == nil {
		processMCPLogLines(response, string(logData))
	}
	ensureDefaultMCPLogs(response)
}

func processMCPLogLines(response *QDDState, logData string) {
	lines := strings.Split(logData, "\n")
	startIdx := len(lines) - 30
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			response.MCPLogs = append(response.MCPLogs, line)
		}
	}
}

func ensureDefaultMCPLogs(response *QDDState) {
	if len(response.MCPLogs) == 0 {
		response.MCPLogs = []string{
			"[MCP] Servidor QDD inicializado.",
			"[MCP] Escuchando intenciones en stdio/SSE...",
		}
	}
}

func loadTelemetry(response *QDDState) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	response.Telemetry = DashboardTelemetry{
		Uptime:      time.Since(startTime).Round(time.Second).String(),
		MemoryAlloc: fmt.Sprintf("%.2f MB", float64(m.Alloc)/1024/1024),
		MemorySys:   fmt.Sprintf("%.2f MB", float64(m.Sys)/1024/1024),
		Goroutines:  runtime.NumGoroutine(),
	}
}

func loadStateJson(response *QDDState, qddDir string) {
	stateData, err := os.ReadFile(filepath.Join(qddDir, "state.json"))
	if err == nil {
		var state map[string]interface{}
		json.Unmarshal(stateData, &state)
		if ver, ok := state["version"]; ok {
			response.Version = fmt.Sprintf("%v", ver)
		}
	}
}

func loadWorkingOn(response *QDDState, qddDir string) {
	workData, err := os.ReadFile(filepath.Join(qddDir, "working"))
	if err == nil {
		response.WorkingOn = strings.TrimSpace(string(workData))
	}
}

func loadConfigAndKnowledge(response *QDDState, cwd, qddDir string) {
	configData, err := os.ReadFile(filepath.Join(qddDir, "config.yaml"))
	if err == nil {
		parseConfigAndDocs(response, configData, cwd)
	}
	ensureLifecycleDocExists(response)
}

func parseConfigAndDocs(response *QDDState, configData []byte, cwd string) {
	var config map[string]interface{}
	yaml.Unmarshal(configData, &config)
	response.Config = config

	if docs, ok := config["documentation_index"].([]interface{}); ok {
		for _, docPath := range docs {
			if p, ok := docPath.(string); ok {
				addDocToKnowledge(response, cwd, p)
			}
		}
	}
}

func addDocToKnowledge(response *QDDState, cwd, p string) {
	if content, err := os.ReadFile(filepath.Join(cwd, p)); err == nil {
		response.Knowledge = append(response.Knowledge, DashboardKnowledgeDoc{
			ID:      filepath.Base(p),
			Path:    p,
			Content: string(content),
		})
	}
}

func ensureLifecycleDocExists(response *QDDState) {
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
}

func loadUnderstandingAndConfig(response *QDDState, qddDir string) {
	configData, _ := os.ReadFile(filepath.Join(qddDir, "config.yaml"))
	var config struct {
		AutoUICertification *bool `yaml:"auto_ui_certification"`
	}
	response.AutoUICert = true
	if yaml.Unmarshal(configData, &config) == nil && config.AutoUICertification != nil {
		response.AutoUICert = *config.AutoUICertification
	}

	undData, err := os.ReadFile(filepath.Join(qddDir, "understanding.json"))
	if err == nil {
		var und DashboardUnderstanding
		if err := json.Unmarshal(undData, &und); err == nil {
			response.Understanding = &und
		}
	}
}

func loadTopology(response *QDDState, cwd, qddDir string) {
	topData, errTop := os.ReadFile(filepath.Join(qddDir, "project", "topology.json"))
	if errTop != nil {
		if top, mapErr := topology.MapProject(cwd); mapErr == nil {
			response.Topology = top
		}
		return
	}
	var top topology.ProjectTopology
	if err := json.Unmarshal(topData, &top); err == nil {
		response.Topology = &top
	}
}

func loadCertifications(response *QDDState, db *sql.DB) {
	rows, err := db.QueryContext(context.Background(), "SELECT id, name, metadata FROM nodes WHERE type = 'rule'")
	if err == nil {
		for rows.Next() {
			processCertificationRow(response, rows)
		}
		rows.Close()
	}
}

func processCertificationRow(response *QDDState, rows *sql.Rows) {
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

func loadFindings(response *QDDState, db *sql.DB) int {
	openFindings := 0
	rows, err := db.QueryContext(context.Background(), "SELECT id, name, metadata FROM nodes WHERE type = 'finding'")
	if err == nil {
		for rows.Next() {
			openFindings += processFindingRow(response, rows)
		}
		rows.Close()
	}
	return openFindings
}

func processFindingRow(response *QDDState, rows *sql.Rows) int {
	var id, name, metaStr string
	rows.Scan(&id, &name, &metaStr)
	var rawData map[string]interface{}
	json.Unmarshal([]byte(metaStr), &rawData)

	estado := findings.Normalizar(textoDe(rawData, "status"))
	if estado == findings.Resuelto {
		response.ValueMetrics.DebtReduced++
	}

	response.Findings = append(response.Findings, DashboardFinding{
		ID:     name,
		Status: etiquetaDeEstado[estado],
		Pillar: determineFindingPillar(name, rawData),
		Desc:   descripcionDe(rawData),
		Raw:    rawData,
	})

	if estado == findings.Abierto {
		return 1
	}
	return 0
}

// etiquetaDeEstado es lo que muestra la UI para cada estado normalizado.
var etiquetaDeEstado = map[findings.Estado]string{
	findings.Abierto:    "OPEN",
	findings.Resuelto:   "RESOLVED",
	findings.Descartado: "WONTFIX",
	findings.SinEstado:  "SIN ESTADO",
}

func textoDe(raw map[string]interface{}, clave string) string {
	if raw == nil || raw[clave] == nil {
		return ""
	}
	return fmt.Sprintf("%v", raw[clave])
}

// descripcionDe usa lo que dice el finding. Antes todos mostraban el texto
// fijo "Deuda técnica documentada." (FND-055).
func descripcionDe(raw map[string]interface{}) string {
	if d := textoDe(raw, "description"); d != "" {
		return d
	}
	return textoDe(raw, "title")
}

func determineFindingPillar(name string, rawData map[string]interface{}) string {
	descLower := getLowerDescription(rawData)
	searchStr := strings.ToLower(name) + " " + descLower

	return checkPillars(searchStr)
}

func getLowerDescription(rawData map[string]interface{}) string {
	if rawData != nil && rawData["description"] != nil {
		return strings.ToLower(fmt.Sprintf("%v", rawData["description"]))
	}
	return ""
}

func checkPillars(searchStr string) string {
	if isCertPillar(searchStr) {
		return "Certificación"
	}
	if isStabilityPillar(searchStr) {
		return "Estabilidad"
	}
	if isSecurityPillar(searchStr) {
		return "Seguridad"
	}
	if isStructuralPillar(searchStr) {
		return "Estructural"
	}
	return "Certificación"
}

func containsAny(s string, keywords []string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func isCertPillar(s string) bool {
	return containsAny(s, []string{"cert", "missing", "adr", "doc"})
}

func isStabilityPillar(s string) bool {
	return containsAny(s, []string{"timeout", "count", "performance", "flaky", "estabilidad"})
}

func isSecurityPillar(s string) bool {
	return containsAny(s, []string{"key", "secret", "sql", "auth", "cors", "seguridad"})
}

func isStructuralPillar(s string) bool {
	return containsAny(s, []string{"else", "complexity", "ciclomática", "legacy", "estructural"})
}

func loadGraphNodes(response *QDDState, db *sql.DB) {
	response.GraphData = DashboardGraphData{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	rows, err := db.QueryContext(context.Background(), "SELECT id, type, name, content FROM nodes")
	if err == nil {
		for rows.Next() {
			var id, nodeType, name, content sql.NullString
			rows.Scan(&id, &nodeType, &name, &content)
			response.GraphData.Nodes = append(response.GraphData.Nodes, GraphNode{ID: id.String, Type: nodeType.String, Name: name.String, Content: content.String})
		}
		rows.Close()
	}
}

func loadGraphEdges(response *QDDState, db *sql.DB) {
	rows, err := db.QueryContext(context.Background(), "SELECT source_id, target_id, relation FROM edges")
	if err == nil {
		for rows.Next() {
			var source, target, relation string
			rows.Scan(&source, &target, &relation)
			response.GraphData.Edges = append(response.GraphData.Edges, GraphEdge{Source: source, Target: target, Relation: relation})
		}
		rows.Close()
	}
}

// loadHistoricalTrends arma la tendencia desde las certificaciones reales
// (certificate_history.json, que escribe audit/certify): el último puntaje de
// cada día con la escala actual. Sin certificaciones el gráfico queda vacío
// (FND-051). cognitive_history.json no sirve: mezcla eventos del MCP y del
// benchmark sin puntaje (FND-058).
func loadHistoricalTrends(response *QDDState, cwd string) {
	response.Historical = []HistoricalTrendPoint{}
	data, err := os.ReadFile(filepath.Join(cwd, ".qdd", "project", "metrics", "certificate_history.json"))
	if err != nil {
		return
	}
	var history []audit.Certificate
	if json.Unmarshal(data, &history) != nil {
		return
	}
	for _, cert := range history {
		agregarPuntoDelDia(response, cert)
	}
}

func agregarPuntoDelDia(response *QDDState, cert audit.Certificate) {
	if cert.Escala != audit.EscalaActual || len(cert.Timestamp) < 10 {
		return
	}
	punto := HistoricalTrendPoint{Date: cert.Timestamp[:10], Score: cert.Score}
	n := len(response.Historical)
	if n > 0 && response.Historical[n-1].Date == punto.Date {
		response.Historical[n-1] = punto
		return
	}
	response.Historical = append(response.Historical, punto)
}

func loadSprints(response *QDDState, db *sql.DB) {
	rows, err := db.QueryContext(context.Background(), "SELECT id, name, content, metadata FROM nodes WHERE type = 'task'")
	if err == nil {
		for rows.Next() {
			processSprintRow(response, rows)
		}
		rows.Close()
	}
}

func processSprintRow(response *QDDState, rows *sql.Rows) {
	var id, name, contentStr, metaStr string
	rows.Scan(&id, &name, &contentStr, &metaStr)
	var rawData map[string]interface{}
	json.Unmarshal([]byte(metaStr), &rawData)

	status := determineSprintStatus(contentStr, rawData)

	response.Sprints = append(response.Sprints, DashboardSprint{
		ID:     name,
		Status: status,
		Raw:    rawData,
	})
}

func determineSprintStatus(contentStr string, rawData map[string]interface{}) string {
	if rawData != nil && rawData["status"] != nil {
		return fmt.Sprintf("%v", rawData["status"])
	}
	return extractStatusFromContent(contentStr)
}

func extractStatusFromContent(contentStr string) string {
	hasChecked := strings.Contains(contentStr, "- [x]")
	hasUnchecked := strings.Contains(contentStr, "- [ ]")

	if hasChecked && !hasUnchecked {
		return "COMPLETED"
	}
	if !hasChecked && !hasUnchecked {
		return "BACKLOG"
	}
	return "IN-PROGRESS"
}

// gradoSinMedir se muestra mientras el proyecto no tiene una certificación con
// la escala actual: el dashboard nunca inventa un puntaje.
const gradoSinMedir = audit.GradoSinMedir

// puntajeDelDashboard muestra el puntaje único de QDD (audit.Score) tal como
// quedó en la última certificación (FND-051), con la escala única (FND-057).
// No corre la auditoría: eso ejecutaría las pruebas en cada refresco.
func puntajeDelDashboard(cwd string) (int, string, bool) {
	score, medido := audit.UltimoPuntaje(cwd)
	if !medido {
		return 0, gradoSinMedir, false
	}
	return score, audit.Grado(score), true
}

// determineQualityStatus reflects open-findings count, not qdd audit's
// static violations (see FND-021) — the two are independent signals.
func determineQualityStatus(openFindings int) string {
	if openFindings > 0 {
		return "FAIL (Deuda Técnica Detectada)"
	}
	return "PASS"
}
