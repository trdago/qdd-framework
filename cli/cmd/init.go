package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize QDD in the current project",
	Long: `Scans the current directory to detect the environment (language, framework, cloud)
and sets up the .qdd directory with the appropriate runtime and specification.`,
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

	fmt.Println("[+] Detectando entorno...")
	languages := detectLanguages(cwd)
	for _, lang := range languages {
		fmt.Printf("[+] Detectado: Lenguaje %s\n", lang)
	}

	if len(languages) == 0 {
		fmt.Println("[-] No se detectó un lenguaje soportado automáticamente.")
	}

	fmt.Println("[+] Creando estructura .qdd/")
	err = createQDDStructure(cwd, languages)
	if err != nil {
		fmt.Printf("[!] Error creando estructura: %v\n", err)
		return
	}

	fmt.Println("[+] Sincronizando integraciones de Inteligencia Artificial (QDD Adapters)...")
	manager := integration.NewIntegrationManager()
	if err := manager.SyncAll(cwd); err != nil {
		fmt.Printf("[!] Advertencia: fallo al sincronizar adaptadores IA: %v\n", err)
	}

	fmt.Println("[!] QDD inicializado exitosamente. Siguiente paso: ejecuta `qdd learn`")
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
		
		// Evitamos escanear dependencias o configuraciones internas
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".qdd" || d.Name() == "node_modules" || d.Name() == "vendor") {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			if d.Name() == "go.mod" {
				detectedMap["Go"] = true
			}
			if d.Name() == "package.json" {
				detectedMap["Node"] = true
			}
			if d.Name() == "pom.xml" {
				detectedMap["Java"] = true
			}
		}

		return nil
	})

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

	err := os.MkdirAll(qddDir, 0755)
	if err != nil {
		return err
	}

	dirsToCreate := []string{
		"runtime",
		"specification",
		"templates",
		"plugins",
		"findings",
		"certification",
		"evidence",
		"metrics",
		"dashboard",
	}

	for _, d := range dirsToCreate {
		err := os.MkdirAll(filepath.Join(qddDir, d), 0755)
		if err != nil {
			return err
		}
	}

	err = createConfigFile(qddDir, languages)
	if err != nil {
		return err
	}

	err = createStateFile(qddDir)
	if err != nil {
		return err
	}

	err = createSystemSpecFile(qddDir)
	if err != nil {
		return err
	}

	return nil
}

func createSystemSpecFile(qddDir string) error {
	specPath := filepath.Join(qddDir, "specification", "qdd-spec.md")
	
	if fileExists(specPath) {
		return nil
	}

	return os.WriteFile(specPath, []byte(SystemPromptTemplate), 0644)
}

func createConfigFile(qddDir string, languages []string) error {
	configPath := filepath.Join(qddDir, "config.yaml")

	if fileExists(configPath) {
		return nil
	}

	content := fmt.Sprintf(`project: auto-detected
languages: %v
governance:
  certification_first: true
  evidence_required_for_fixes: true
`, languages)

	return os.WriteFile(configPath, []byte(content), 0644)
}

func createStateFile(qddDir string) error {
	statePath := filepath.Join(qddDir, "state.json")

	if fileExists(statePath) {
		return nil
	}

	state := map[string]interface{}{
		"status":  "initialized",
		"version": "v0.1.0",
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}
