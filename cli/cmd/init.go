package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/qcl/auth"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa el entorno QDD en el proyecto actual",
	Long: `Escanea el directorio actual para detectar el ecosistema (lenguaje, framework)
y configura el directorio .qdd con el runtime y la especificación apropiada.

Reconoce Node, Go, Java/Kotlin, Python, Dart/Flutter, Rust, PHP, Ruby y .NET,
también dentro de subcarpetas (monorepos). En un proyecto sin código todavía,
declara el stack:

  qdd init --lenguaje Go --arquitectura "API REST"

Es seguro repetirlo: no pisa lo que ya configuraste.`,
	Run: runInit,
}

var (
	initLenguajes    []string
	initArquitectura string
)

func init() {
	initCmd.Flags().StringSliceVar(&initLenguajes, "lenguaje", nil, "Lenguaje(s) del proyecto, si no se detectan (p. ej. Go o Python,Node)")
	initCmd.Flags().StringVar(&initArquitectura, "arquitectura", "", "Tipo de proyecto, si no se detecta (p. ej. \"API REST\")")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Inicializando QDD Framework...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		return
	}
	cwd = integration.FindProjectRoot(cwd)
	ensureAdopcion(cwd, !fileExists(filepath.Join(cwd, ".qdd", "config.yaml")))
	executeInitLoop(cwd)
}

// ensureAdopcion deja a los proyectos nuevos en adopción gradual (FND-040): las
// reglas de madurez se informan sin bloquear. Un proyecto que ya existía, o que
// ya declaró sus políticas, queda como estaba.
func ensureAdopcion(cwd string, nuevo bool) {
	if !nuevo || fileExists(filepath.Join(cwd, ".qdd", "policies.yaml")) {
		return
	}
	p := audit.DefaultPolicies()
	p.Adopcion = audit.AdopcionGradual
	_ = audit.SavePolicies(cwd, p)
}

func executeInitLoop(cwd string) {
	iteration := 1
	lastFailCount := -1

	for {
		success, failCount := runInitIteration(cwd)
		if success {
			printInitSummary(cwd)
			return
		}

		checkStagnation(iteration, failCount, lastFailCount)

		lastFailCount = failCount
		iteration++
		fmt.Printf("[~] QDD Doctor corrigió %d detalle(s); verificando de nuevo (intento %d)...\n", failCount, iteration)
	}
}

func printInitSummary(cwd string) {
	meta := readDeclaredMetadata(cwd)
	fmt.Println("\n[✓] QDD quedó listo y QDD Doctor lo verificó.")
	if len(meta.Languages) == 0 {
		fmt.Println("\n[i] Aún no hay código, así que QDD no sabe qué stack usarás. Decláralo para habilitar audit y certify:")
		fmt.Println("      qdd init --lenguaje Go --arquitectura \"API REST\"")
		fmt.Println("    (o abre tu IA en esta carpeta y pídele que ejecute qdd_learn)")
		return
	}
	fmt.Printf("    Proyecto: %s · %s\n", strings.Join(meta.Languages, ", "), meta.Architecture)
	fmt.Println("\nSiguientes pasos:")
	fmt.Println("  1. qdd audit   → revisa el proyecto (lo que es recomendación no bloquea)")
	fmt.Println("     Lo temporal (scripts de un solo uso, logs, salidas) va en ./tmp: git la ignora.")
	fmt.Println("  2. Abre tu IA (Claude Code, Cursor, Antigravity, OpenCode) en esta carpeta: ya quedó conectada a QDD.")
	fmt.Println("     Pídele: \"ejecuta qdd_learn\" para que aprenda la arquitectura.")
}

// readDeclaredMetadata lee lenguaje y arquitectura tal como quedaron en config.yaml.
func readDeclaredMetadata(cwd string) ProjectMetadata {
	var cfg struct {
		Architecture string   `yaml:"architecture"`
		Languages    []string `yaml:"languages"`
	}
	data, _ := os.ReadFile(filepath.Join(cwd, ".qdd", "config.yaml"))
	_ = yaml.Unmarshal(data, &cfg)
	return ProjectMetadata{Architecture: cfg.Architecture, Languages: cfg.Languages}
}

func checkStagnation(iteration, failCount, lastFailCount int) {
	if lastFailCount == -1 {
		return
	}
	if failCount >= lastFailCount {
		fmt.Printf("[!] Error crítico: QDD Doctor no pudo reparar el entorno tras %d iteraciones (anomalías estancadas en %d). Revisa la evidencia generada por QDD Doctor.\n", iteration, failCount)
		os.Exit(1)
	}
}

func runInitIteration(cwd string) (bool, int) {
	fmt.Println("[+] Detectando entorno...")
	meta := aplicarDeclaracion(detectProjectMetadata(cwd), initLenguajes, initArquitectura)
	printMetadata(meta)

	fmt.Println("[+] Creando estructura .qdd/")
	if err := createQDDStructure(cwd, meta); err != nil {
		fmt.Printf("[✗] Error creando estructura: %v\n", err)
		return false, 1 // Artificial error count to force retry or stagnation
	}
	declararStack(cwd, meta)
	limpiarPycacheDelCore(filepath.Join(cwd, ".qdd"))
	if aviso := ensureTmpDir(cwd); aviso != "" {
		fmt.Printf("[i] %s\n", aviso)
	}

	fmt.Println("[+] Conectando asistentes de IA:")
	manager := integration.NewIntegrationManager()
	if err := manager.SyncAll(cwd); err != nil {
		fmt.Printf("[!] No se pudieron configurar los asistentes de IA: %v\n", err)
	}

	fmt.Println("[+] Verificando la instalación con QDD Doctor...")
	return RunDoctorCheck(cwd, true)
}

func printMetadata(meta ProjectMetadata) {
	for _, lang := range meta.Languages {
		fmt.Printf("[+] Detectado: Lenguaje %s\n", lang)
	}
	if meta.Architecture != "" {
		fmt.Printf("[+] Detectada Arquitectura: %s\n", meta.Architecture)
	}
	if len(meta.Languages) == 0 {
		fmt.Println("[i] No encontré archivos de proyecto (go.mod, package.json, requirements.txt, pubspec.yaml...).")
	}
}

// declararStack escribe en config.yaml lo pasado por --lenguaje/--arquitectura,
// aunque ya hubiera valores: es una declaración explícita del usuario.
func declararStack(cwd string, meta ProjectMetadata) {
	if len(initLenguajes) == 0 && strings.TrimSpace(initArquitectura) == "" {
		return
	}
	if err := completarConfig(filepath.Join(cwd, ".qdd", "config.yaml"), meta, true); err != nil {
		fmt.Printf("[✗] No se pudo guardar el stack declarado: %v\n", err)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

type ProjectMetadata struct {
	Name         string
	Architecture string
	Languages    []string
}

// detectProjectMetadata reconoce el stack en la raíz y, si no hay nada ahí, en
// las subcarpetas (monorepos: backend/, frontend/, apps/*...).
func detectProjectMetadata(dir string) ProjectMetadata {
	meta := detectarEn(dir)
	if len(meta.Languages) > 0 {
		return meta
	}
	subs := detectarSubproyectos(dir, 1)
	if len(subs) == 0 {
		return meta
	}
	return metadataDeMonorepo(dir, subs)
}

func detectNodeJS(dir string, meta *ProjectMetadata) {
	if !fileExists(filepath.Join(dir, "package.json")) {
		return
	}
	meta.Languages = append(meta.Languages, "Node")
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return
	}
	var pkg map[string]interface{}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	extractNodeJSName(pkg, meta)
	deps := extractDependencies(pkg)
	assignNodeJSArchitecture(deps, meta)
}

func extractNodeJSName(pkg map[string]interface{}, meta *ProjectMetadata) {
	if name, ok := pkg["name"].(string); ok && name != "" {
		meta.Name = name
	}
}

func assignNodeJSArchitecture(deps map[string]bool, meta *ProjectMetadata) {
	meta.Architecture = "Node.js Application"
	if hasServerless(deps) {
		meta.Architecture = "Serverless Cloud Functions"
	}
	if hasRestDeps(deps) {
		meta.Architecture = "Backend REST API"
	}
	if hasWebDeps(deps) {
		meta.Architecture = "Frontend Web App"
	}
}

func hasServerless(deps map[string]bool) bool {
	return deps["serverless"] || deps["aws-sdk"]
}

func hasRestDeps(deps map[string]bool) bool {
	return deps["express"] || deps["nestjs"] || deps["fastify"]
}

func hasWebDeps(deps map[string]bool) bool {
	return deps["next"] || deps["react"] || deps["vue"] || deps["angular"]
}

func detectGo(dir string, meta *ProjectMetadata) {
	if !fileExists(filepath.Join(dir, "go.mod")) {
		return
	}
	meta.Languages = append(meta.Languages, "Go")
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		processGoModLine(line, meta)
	}
	if meta.Architecture == "" {
		meta.Architecture = "Go Application"
	}
}

func processGoModLine(line string, meta *ProjectMetadata) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "module ") {
		meta.Name = strings.TrimSpace(strings.TrimPrefix(line, "module "))
	}
	if isGoRest(line) {
		meta.Architecture = "Backend REST API (Go)"
	}
	if isGoCLI(line) {
		meta.Architecture = "CLI Application (Go)"
	}
}

func isGoRest(line string) bool {
	return strings.Contains(line, "gin-gonic/gin") || strings.Contains(line, "gofiber/fiber") || strings.Contains(line, "labstack/echo")
}

func isGoCLI(line string) bool {
	return strings.Contains(line, "spf13/cobra") || strings.Contains(line, "urfave/cli")
}

func detectJava(dir string, meta *ProjectMetadata) {
	if !fileExists(filepath.Join(dir, "pom.xml")) && !fileExists(filepath.Join(dir, "build.gradle")) && !fileExists(filepath.Join(dir, "build.gradle.kts")) {
		return
	}
	meta.Languages = append(meta.Languages, "Java")
	setArchitecture(meta, "Java Application")
}

func extractDependencies(pkg map[string]interface{}) map[string]bool {
	deps := make(map[string]bool)
	if d, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k := range d {
			deps[k] = true
		}
	}
	if d, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k := range d {
			deps[k] = true
		}
	}
	return deps
}

func createQDDStructure(baseDir string, meta ProjectMetadata) error {
	qddDir := filepath.Join(baseDir, ".qdd")

	if err := os.MkdirAll(qddDir, 0755); err != nil {
		return err
	}

	if err := createQDDDirectories(qddDir); err != nil {
		return err
	}

	if err := createConfigFile(qddDir, meta); err != nil {
		return err
	}

	if err := createStateFile(qddDir); err != nil {
		return err
	}

	if err := createCognitiveHistory(baseDir); err != nil {
		return err
	}

	return unpackCoreAssets(qddDir)
}

// createCognitiveHistory deja el contrato que espera el dashboard; sin él, el
// primer init mostraba una "reparación" de QDD Doctor (FND-043).
func createCognitiveHistory(baseDir string) error {
	historyPath := filepath.Join(baseDir, ".qdd", "project", "metrics", "cognitive_history.json")
	if fileExists(historyPath) {
		return nil
	}
	return os.WriteFile(historyPath, []byte("[]"), 0644)
}

func GetQDDDirectories() []string {
	return []string{
		"core/runtime",
		"core/specification",
		"core/templates",
		"core/plugins",
		"core/certification",
		"core/wisdom",
		"project/findings",
		"project/sprints",
		"project/certification",
		"project/evidence",
		"project/metrics",
		"project/adr",
		"project/goldensets",
		"dashboard",
	}
}

func createQDDDirectories(qddDir string) error {
	dirsToCreate := GetQDDDirectories()

	for _, d := range dirsToCreate {
		if err := os.MkdirAll(filepath.Join(qddDir, d), 0755); err != nil {
			return err
		}
	}
	return nil
}

//go:embed all:core_assets/core
var coreAssets embed.FS

func unpackCoreAssets(qddDir string) error {
	basePath := "core_assets/core"
	return fs.WalkDir(coreAssets, basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || esCachePython(path) {
			return nil
		}
		return extractSingleCoreAsset(qddDir, basePath, path)
	})
}

func extractSingleCoreAsset(qddDir, basePath, path string) error {
	relPath := strings.TrimPrefix(path, basePath+"/")
	destPath := filepath.Join(qddDir, "core", relPath)
	
	if fileExists(destPath) {
		return nil
	}
	
	content, err := coreAssets.ReadFile(path)
	if err != nil {
		return err
	}
	
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	
	if errGuard := auth.GuardCoreWriteAccess(destPath); errGuard != nil {
		return errGuard
	}
	
	return os.WriteFile(destPath, content, 0644)
}


func createConfigFile(qddDir string, meta ProjectMetadata) error {
	configPath := filepath.Join(qddDir, "config.yaml")

	if fileExists(configPath) {
		return completarConfig(configPath, meta, false)
	}

	config := map[string]interface{}{
		"project":      meta.Name,
		"architecture": meta.Architecture,
		"languages":    meta.Languages,
		"governance": map[string]interface{}{
			"certification_first":         true,
			"evidence_required_for_fixes": true,
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func createStateFile(qddDir string) error {
	statePath := filepath.Join(qddDir, "state.json")
	var state map[string]interface{}

	if fileExists(statePath) {
		data, err := os.ReadFile(statePath)
		if err == nil {
			json.Unmarshal(data, &state)
		}
	}
	
	if state == nil {
		state = map[string]interface{}{
			"status": "initialized",
		}
	}

	state["version"] = rootCmd.Version

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

func detectLanguages(dir string) []string {
	var languages []string
	hasGo, hasNode := scanForLanguages(dir)

	if hasGo {
		languages = append(languages, "Go")
	}
	if hasNode {
		languages = append(languages, "Node")
	}
	return languages
}

func scanForLanguages(dir string) (bool, bool) {
	hasGo := false
	hasNode := false
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == "go.mod" {
			hasGo = true
		}
		if d.Name() == "package.json" {
			hasNode = true
		}
		return nil
	})
	return hasGo, hasNode
}
