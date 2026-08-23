package cmd

import (
	"context"
	"encoding/json"
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

		checklist := runDeterministicChecks(cwd)
		if autoFix && checklist.HasFailures() {
			fmt.Println("[!] Doctor: Detectadas anomalías, ejecutando reparaciones selectivas...")
			runSelectiveFixes(cwd, checklist)
			fmt.Println()
			checklist = runDeterministicChecks(cwd) // Re-evaluar post-reparación
		}

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
	list.Groups = append(list.Groups, buildGroup1Checks(projectPath))
	list.Groups = append(list.Groups, buildGroup2Checks(projectPath))
	list.Groups = append(list.Groups, buildGroup3Checks(projectPath))
	return list
}

func buildGroup1Checks(projectPath string) *CheckGroup {
	g1 := &CheckGroup{Name: "1. Estructura de Archivos y Carpetas Base"}
	qddDir := filepath.Join(projectPath, ".qdd")
	g1.Items = append(g1.Items, checkDir(qddDir, "Directorio raíz .qdd/"))
	g1.Items = append(g1.Items, checkDir(filepath.Join(qddDir, "core"), "Directorio base .qdd/core/"))
	g1.Items = append(g1.Items, checkDir(filepath.Join(qddDir, "project"), "Directorio de proyecto .qdd/project/"))
	g1.Items = append(g1.Items, checkFile(filepath.Join(qddDir, "config.yaml"), "Configuración base config.yaml"))
	g1.Items = append(g1.Items, checkFile(filepath.Join(qddDir, "state.json"), "Estado de proyecto state.json"))
	return g1
}

func buildGroup2Checks(projectPath string) *CheckGroup {
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
	return g2
}

func buildGroup3Checks(projectPath string) *CheckGroup {
	g3 := &CheckGroup{Name: "3. Dashboard y Entorno Nativo"}
	g3.Items = append(g3.Items, buildDashboardCheck())
	g3.Items = append(g3.Items, buildHistoryCheck(projectPath))
	return g3
}

func buildDashboardCheck() *CheckItem {
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
	return dashCheck
}

func buildHistoryCheck(projectPath string) *CheckItem {
	historyCheck := &CheckItem{Name: "Contrato de Historial (cognitive_history.json)", Success: false, Error: "Archivo ausente"}
	historyPath := filepath.Join(projectPath, ".qdd", "project", "metrics", "cognitive_history.json")
	if fileExists(historyPath) {
		historyCheck.Error = "Archivo ausente o inválido"
		content, err := os.ReadFile(historyPath)
		if err == nil {
			var arr []interface{}
			historyCheck.Error = "Corrupción de JSON: no es un array válido"
			if err := json.Unmarshal(content, &arr); err == nil {
				historyCheck.Success = true
				historyCheck.Error = "Esquema JSON válido"
			}
		}
	}
	return historyCheck
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

func runSelectiveFixes(projectPath string, checklist *Checklist) {
	qddDir := filepath.Join(projectPath, ".qdd")
	
	for _, g := range checklist.Groups {
		for _, item := range g.Items {
			if !item.Success {
				dispatchRepairCommand(item.Name, projectPath, qddDir)
			}
		}
	}
}

func dispatchRepairCommand(itemName, projectPath, qddDir string) {
	repairRegistry := map[string]func(string, string){
		"Directorio raíz .qdd/": repairDirs,
		"Directorio base .qdd/core/": repairDirs,
		"Directorio de proyecto .qdd/project/": repairDirs,
		"Configuración base config.yaml": repairConfig,
		"Estado de proyecto state.json": repairState,
		"Integración Cursor (.cursor/mcp.json)": repairMCP,
		"Integración Claude (.clauderc)": repairMCP,
		"Integración Antigravity (.antigravityrules)": repairMCP,
		"Assets estáticos del Dashboard (Vue) embebidos": repairAssets,
		"Contrato de Historial (cognitive_history.json)": repairHistory,
	}
	
	if handler, ok := repairRegistry[itemName]; ok {
		handler(projectPath, qddDir)
	}
}

func repairDirs(projectPath, qddDir string) {
	fmt.Println("  [~] Reconstruyendo directorios base...")
	_ = createQDDDirectories(qddDir)
}

func repairConfig(projectPath, qddDir string) {
	fmt.Println("  [~] Restaurando configuración base...")
	meta := detectProjectMetadata(projectPath)
	_ = createConfigFile(qddDir, meta)
}

func repairState(projectPath, qddDir string) {
	fmt.Println("  [~] Restaurando estado de proyecto...")
	_ = createStateFile(qddDir)
}

func repairMCP(projectPath, qddDir string) {
	fmt.Println("  [~] Sincronizando perfiles de IA (MCP)...")
	manager := integration.NewIntegrationManager()
	_ = manager.SyncAll(projectPath)
}

func repairAssets(projectPath, qddDir string) {
	fmt.Println("  [~] Desempaquetando assets nativos...")
	_ = unpackCoreAssets(qddDir)
}

func repairHistory(projectPath, qddDir string) {
	fmt.Println("  [~] Reparando contratos JSON del Dashboard...")
	repairDashboardContracts(projectPath)
}

func repairDashboardContracts(projectPath string) {
	historyPath := filepath.Join(projectPath, ".qdd", "project", "metrics", "cognitive_history.json")
	_ = os.MkdirAll(filepath.Dir(historyPath), 0755)
	_ = os.WriteFile(historyPath, []byte("[]"), 0644)
}

// RunDoctorCheck ejecuta las pruebas deterministas del framework y deja un reporte.
func RunDoctorCheck(projectPath string, autoFix bool) (bool, int) {
	checklist := runDeterministicChecks(projectPath)
	if autoFix && checklist.HasFailures() {
		runSelectiveFixes(projectPath, checklist)
		checklist = runDeterministicChecks(projectPath)
	}
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
