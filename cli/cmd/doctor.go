package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/qcl/wisdom"
	"github.com/qdd-framework/qdd/ui"
	"github.com/spf13/cobra"
)

var autoFix bool

type CheckItem struct {
	Name    string
	Success bool
	Error   string
}

type CheckGroup struct {
	Name  string
	Items []*CheckItem
}

type Checklist struct {
	Groups []*CheckGroup
}

func (c *Checklist) HasFailures() bool {
	for _, g := range c.Groups {
		for _, item := range g.Items {
			if !item.Success {
				return true
			}
		}
	}
	return false
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verifica que el entorno QDD esté completamente funcional (Checklist Determinista)",
	Long:  `Ejecuta pruebas deterministas para asegurar que la estructura y las integraciones del framework QDD están operando correctamente, evaluando de forma rígida cada aspecto del framework.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("[!] Error obteniendo directorio actual: %v\n", err)
			os.Exit(1)
		}
		cwd = integration.FindProjectRoot(cwd)

		if autoFix {
			fmt.Println("[!] Doctor: Ejecutando auto-reparación inicial...")
			runAutoFix(cwd)
			fmt.Println()
		}

		checklist := runDeterministicChecks(cwd)
		printChecklist(checklist)
		generateReport(cwd, checklist)

		if checklist.HasFailures() {
			fmt.Println("\n[-] QDD Doctor detectó anomalías estructurales.")
			if !autoFix {
				fmt.Println("    Ejecuta `qdd doctor --fix` para intentar repararlas automáticamente.")
			}
			os.Exit(1)
		}

		fmt.Println("\n[+] QDD Doctor: Todo el entorno se encuentra 100% operativo y sincronizado.")
	},
}

func init() {
	doctorCmd.Flags().BoolVarP(&autoFix, "fix", "f", false, "Intenta auto-reparar los archivos o configuraciones del core dañadas")
	rootCmd.AddCommand(doctorCmd)
}

func runDeterministicChecks(projectPath string) *Checklist {
	list := &Checklist{}

	// Grupo 1: Estructura de Archivos
	g1 := &CheckGroup{Name: "1. Estructura de Archivos y Carpetas Base"}
	qddDir := filepath.Join(projectPath, ".qdd")
	
	g1.Items = append(g1.Items, checkDir(qddDir, "Directorio raíz .qdd/"))
	g1.Items = append(g1.Items, checkDir(filepath.Join(qddDir, "core"), "Directorio base .qdd/core/"))
	g1.Items = append(g1.Items, checkDir(filepath.Join(qddDir, "project"), "Directorio de proyecto .qdd/project/"))
	g1.Items = append(g1.Items, checkFile(filepath.Join(qddDir, "config.yaml"), "Configuración base config.yaml"))
	g1.Items = append(g1.Items, checkFile(filepath.Join(qddDir, "state.json"), "Estado de proyecto state.json"))
	list.Groups = append(list.Groups, g1)

	// Grupo 2: Infraestructura IA y MCP
	g2 := &CheckGroup{Name: "2. Infraestructura de IA y MCP"}
	g2.Items = append(g2.Items, checkFile(filepath.Join(projectPath, ".cursor", "mcp.json"), "Integración Cursor (.cursor/mcp.json)"))
	g2.Items = append(g2.Items, checkFile(filepath.Join(projectPath, ".clauderc"), "Integración Claude (.clauderc)"))
	g2.Items = append(g2.Items, checkFile(filepath.Join(projectPath, ".antigravityrules"), "Integración Antigravity (.antigravityrules)"))
	
	wisdomCheck := &CheckItem{Name: "Conexión a Cloud Wisdom (Oráculo AI)", Success: true, Error: "Offline (Graceful Degradation)"}
	wClient := wisdom.NewClient(projectPath)
	manifest, err := wClient.FetchRulesManifest(context.Background())
	if err == nil && manifest != nil {
		wisdomCheck.Error = "Conectado"
	}
	g2.Items = append(g2.Items, wisdomCheck)
	list.Groups = append(list.Groups, g2)

	// Grupo 3: Dashboard y Telemetría
	g3 := &CheckGroup{Name: "3. Dashboard y Entorno Nativo"}
	dashCheck := &CheckItem{Name: "Assets estáticos del Dashboard (Vue) embebidos", Success: false, Error: "No se encontró el subdirectorio 'dist' en el binario"}
	distFs, err := fs.Sub(ui.StaticFiles, "dist")
	
	if err == nil {
		dashCheck.Error = "index.html ausente en la compilación embebida"
		_, errHtml := fs.Stat(distFs, "index.html")
		if errHtml == nil {
			dashCheck.Success = true
			dashCheck.Error = "Embebido correctamente"
		}
	}
	
	g3.Items = append(g3.Items, dashCheck)
	list.Groups = append(list.Groups, g3)

	return list
}

func checkDir(path, name string) *CheckItem {
	info, err := os.Stat(path)
	item := &CheckItem{Name: name, Success: false, Error: "Directorio ausente o inválido"}
	if os.IsNotExist(err) || !info.IsDir() {
		return item
	}
	item.Success = true
	item.Error = ""
	return item
}

func checkFile(path, name string) *CheckItem {
	item := &CheckItem{Name: name, Success: false, Error: "Archivo no encontrado"}
	if !fileExists(path) {
		return item
	}
	item.Success = true
	item.Error = ""
	return item
}

func printChecklist(c *Checklist) {
	fmt.Println("=== QDD Framework Doctor: Deterministic Health Check ===")
	for _, g := range c.Groups {
		fmt.Printf("\n[%s]\n", g.Name)
		for _, item := range g.Items {
			printChecklistItem(item)
		}
	}
}

func printChecklistItem(item *CheckItem) {
	if !item.Success {
		fmt.Printf("  ❌ %s (%s)\n", item.Name, item.Error)
		return
	}
	extra := ""
	if item.Error != "" && item.Error != "Embebido correctamente" && item.Error != "Conectado" {
		extra = fmt.Sprintf(" (%s)", item.Error)
	}
	fmt.Printf("  ✅ %s%s\n", item.Name, extra)
}

func generateReport(projectPath string, c *Checklist) {
	qddDir := filepath.Join(projectPath, ".qdd")
	evidenceDir := filepath.Join(qddDir, "project", "evidence", "doctor")
	_ = os.MkdirAll(evidenceDir, 0755)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	reportPath := filepath.Join(evidenceDir, fmt.Sprintf("report_%s.md", timestamp))

	content := "# QDD Doctor Report\n\n"
	content += fmt.Sprintf("Date: %s\n\n", time.Now().Format(time.RFC3339))

	for _, g := range c.Groups {
		content += generateGroupReport(g)
	}

	stateStr := "HEALTHY"
	if c.HasFailures() {
		stateStr = "CRITICAL_FAILURES"
	}
	content += fmt.Sprintf("\n**Estado Global:** %s\n", stateStr)

	_ = os.WriteFile(reportPath, []byte(content), 0644)
}

func generateGroupReport(g *CheckGroup) string {
	content := fmt.Sprintf("## %s\n\n", g.Name)
	content += "| Validación | Estado | Detalle |\n"
	content += "|---|---|---|\n"
	for _, item := range g.Items {
		status := "✅ OK"
		detail := "-"
		if !item.Success {
			status = "❌ FAIL"
			detail = item.Error
		}
		if item.Success && item.Error != "" {
			detail = item.Error
		}
		content += fmt.Sprintf("| %s | %s | %s |\n", item.Name, status, detail)
	}
	content += "\n"
	return content
}

func runAutoFix(projectPath string) {
	qddDir := filepath.Join(projectPath, ".qdd")
	
	fmt.Println("  [~] Reconstruyendo directorios base...")
	_ = createQDDDirectories(qddDir)
	
	fmt.Println("  [~] Restaurando configuración y estado...")
	meta := detectProjectMetadata(projectPath)
	_ = createConfigFile(qddDir, meta)
	_ = createStateFile(qddDir)
	
	fmt.Println("  [~] Desempaquetando assets nativos...")
	_ = unpackCoreAssets(qddDir)

	fmt.Println("  [~] Sincronizando perfiles de IA...")
	manager := integration.NewIntegrationManager()
	_ = manager.SyncAll(projectPath)
}

// RunDoctorCheck ejecuta las pruebas deterministas del framework y deja un reporte.
func RunDoctorCheck(projectPath string, autoFix bool) (bool, int) {
	if autoFix {
		runAutoFix(projectPath)
	}
	checklist := runDeterministicChecks(projectPath)
	generateReport(projectPath, checklist)

	if checklist.HasFailures() {
		return false, countFailures(checklist)
	}
	return true, 0
}

func countFailures(checklist *Checklist) int {
	failures := 0
	for _, g := range checklist.Groups {
		for _, item := range g.Items {
			if !item.Success {
				failures++
			}
		}
	}
	return failures
}
