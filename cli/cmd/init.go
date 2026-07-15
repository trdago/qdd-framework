package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/qcl/auth"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa el entorno QDD en el proyecto actual",
	Long: `Escanea el directorio actual para detectar el ecosistema (lenguaje, framework)
y configura el directorio .qdd con el runtime y la especificación apropiada.`,
	Run: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
	fmt.Println("[+] Inicializando QDD Framework...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
		return
	}
	executeInitLoop(cwd)
}

func executeInitLoop(cwd string) {
	iteration := 1
	lastFailCount := -1

	for {
		fmt.Printf("[+] Iteración %d de inicialización y chequeo...\n", iteration)
		
		success, failCount := runInitIteration(cwd)
		if success {
			fmt.Println("[!] QDD inicializado exitosamente y validado por QDD Doctor.")
			fmt.Println("[!] Siguiente paso: ejecuta `qdd learn`")
			return
		}

		checkStagnation(iteration, failCount, lastFailCount)

		lastFailCount = failCount
		iteration++
		fmt.Println("[-] QDD Doctor detectó anomalías. Reintentando y auto-corrigiendo de forma segura...")
	}
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
	meta := detectProjectMetadata(cwd)
	for _, lang := range meta.Languages {
		fmt.Printf("[+] Detectado: Lenguaje %s\n", lang)
	}
	if meta.Architecture != "" {
		fmt.Printf("[+] Detectada Arquitectura: %s\n", meta.Architecture)
	}

	if len(meta.Languages) == 0 {
		fmt.Println("[-] No se detectó un lenguaje soportado automáticamente.")
	}

	fmt.Println("[+] Creando estructura .qdd/")
	err := createQDDStructure(cwd, meta)
	if err != nil {
		fmt.Printf("[!] Error creando estructura: %v\n", err)
		return false, 1 // Artificial error count to force retry or stagnation
	}

	fmt.Println("[+] Sincronizando integraciones de Inteligencia Artificial (QDD Adapters)...")
	manager := integration.NewIntegrationManager()
	if err := manager.SyncAll(cwd); err != nil {
		fmt.Printf("[!] Advertencia: fallo al sincronizar adaptadores IA: %v\n", err)
	}

	fmt.Println("[+] Ejecutando QDD Doctor para certificar funcionalidad...")
	return RunDoctorCheck(cwd, true)
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

func detectProjectMetadata(dir string) ProjectMetadata {
	meta := ProjectMetadata{
		Name:         "auto-detected",
		Architecture: "",
		Languages:    []string{},
	}

	if fileExists(filepath.Join(dir, "package.json")) {
		meta.Languages = append(meta.Languages, "Node")
		data, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(data, &pkg) == nil {
				if name, ok := pkg["name"].(string); ok && name != "" {
					meta.Name = name
				}

				deps := extractDependencies(pkg)
				meta.Architecture = "Node.js Application"
				if deps["serverless"] || deps["aws-sdk"] {
					meta.Architecture = "Serverless Cloud Functions"
				}
				if deps["express"] || deps["nestjs"] || deps["fastify"] {
					meta.Architecture = "Backend REST API"
				}
				if deps["next"] || deps["react"] || deps["vue"] || deps["angular"] {
					meta.Architecture = "Frontend Web App"
				}
			}
		}
	}

	if fileExists(filepath.Join(dir, "go.mod")) {
		meta.Languages = append(meta.Languages, "Go")
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					meta.Name = strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
				if strings.Contains(line, "github.com/gin-gonic/gin") || strings.Contains(line, "github.com/gofiber/fiber") || strings.Contains(line, "github.com/labstack/echo") {
					meta.Architecture = "Backend REST API (Go)"
				}
				if strings.Contains(line, "github.com/spf13/cobra") || strings.Contains(line, "github.com/urfave/cli") {
					meta.Architecture = "CLI Application (Go)"
				}
			}
			if meta.Architecture == "" {
				meta.Architecture = "Go Application"
			}
		}
	}

	if fileExists(filepath.Join(dir, "pom.xml")) || fileExists(filepath.Join(dir, "build.gradle")) {
		meta.Languages = append(meta.Languages, "Java")
		if meta.Architecture == "" {
			meta.Architecture = "Java Application"
		}
	}

	return meta
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

	return unpackCoreAssets(qddDir)
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
		if d.IsDir() {
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
		return nil
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
	hasGo := false
	hasNode := false

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if d.Name() == "go.mod" {
				hasGo = true
			}
			if d.Name() == "package.json" {
				hasNode = true
			}
		}
		return nil
	})

	if hasGo {
		languages = append(languages, "Go")
	}
	if hasNode {
		languages = append(languages, "Node")
	}
	return languages
}
