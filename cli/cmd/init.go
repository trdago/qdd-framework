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
	languages := detectLanguages(cwd)
	for _, lang := range languages {
		fmt.Printf("[+] Detectado: Lenguaje %s\n", lang)
	}

	if len(languages) == 0 {
		fmt.Println("[-] No se detectó un lenguaje soportado automáticamente.")
	}

	fmt.Println("[+] Creando estructura .qdd/")
	err := createQDDStructure(cwd, languages)
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
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func detectLanguages(dir string) []string {
	detectedMap := make(map[string]bool)

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		if shouldSkipDirForLangDetection(d) {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			checkLanguageFile(d.Name(), detectedMap)
		}

		return nil
	})

	return compileDetectedLanguages(detectedMap)
}

func shouldSkipDirForLangDetection(d os.DirEntry) bool {
	if !d.IsDir() {
		return false
	}
	name := d.Name()
	return name == ".git" || name == ".qdd" || name == "node_modules" || name == "vendor"
}

func checkLanguageFile(name string, detectedMap map[string]bool) {
	if name == "go.mod" {
		detectedMap["Go"] = true
	}
	if name == "package.json" {
		detectedMap["Node"] = true
	}
	if name == "pom.xml" {
		detectedMap["Java"] = true
	}
}

func compileDetectedLanguages(detectedMap map[string]bool) []string {
	var detected []string
	if detectedMap["Go"] {
		detected = append(detected, "Go")
	}
	if detectedMap["Node"] {
		detected = append(detected, "Node")
	}
	if detectedMap["Java"] {
		detected = append(detected, "Java")
	}
	return detected
}

func createQDDStructure(baseDir string, languages []string) error {
	qddDir := filepath.Join(baseDir, ".qdd")

	if err := os.MkdirAll(qddDir, 0755); err != nil {
		return err
	}

	if err := createQDDDirectories(qddDir); err != nil {
		return err
	}

	if err := createConfigFile(qddDir, languages); err != nil {
		return err
	}

	if err := createStateFile(qddDir); err != nil {
		return err
	}

	return unpackCoreAssets(qddDir)
}

func createQDDDirectories(qddDir string) error {
	dirsToCreate := []string{
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
		"dashboard",
	}

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


func createConfigFile(qddDir string, languages []string) error {
	configPath := filepath.Join(qddDir, "config.yaml")

	if fileExists(configPath) {
		return nil
	}

	config := map[string]interface{}{
		"project":   "auto-detected",
		"languages": languages,
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
