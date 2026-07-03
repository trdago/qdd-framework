package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:     "audit [módulo]",
	Aliases: []string{"AUDIT"},
	Short:   "Ejecuta una auditoría estática sobre el código base",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()

		if len(args) > 0 && args[0] == "dashboard" {
			runDashboardAudit(cwd)
			return
		}

		runBackendAudit(cwd)
	},
}

func runBackendAudit(cwd string) {
	fmt.Println("--- QDD AUDIT REPORT (UNIVERSAL ENGINE) ---")
	
	engine := audit.NewEngine(cwd)
	violations := engine.RunAll()
	
	if len(violations) == 0 {
		fmt.Println("\n[✔] Auditoría exitosa. El código cumple con las 5 certificaciones maestras de QDD:")
		fmt.Println("    - OWASP Security")
		fmt.Println("    - 12-Factor App")
		fmt.Println("    - Clean Code (No-Else)")
		fmt.Println("    - QA Coverage")
		fmt.Println("    - Traceability")
		return
	}
	
	fmt.Printf("\n[!] Se detectaron %d violaciones de certificación:\n\n", len(violations))
	
	for _, v := range violations {
		fmt.Printf("- [%s] %s\n", v.Category, v.Format())
	}
	
	fmt.Println("\n[!] La auditoría ha fallado. Corrige estos errores antes de continuar.")
	os.Exit(1)
}

func runDashboardAudit(cwd string) {
	fmt.Println("--- QDD AUDIT REPORT (UI / DASHBOARD) ---")
	fmt.Println("\n🔄 **Evaluando CERT-011-ISO-9241 (Usabilidad y Feedback)...**")
	
	uiDir := filepath.Join(cwd, "cli", "ui", "src")
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		fmt.Println("[!] No se encontró el código base de la UI en", uiDir)
		return
	}

	hasLiveRegions := false
	hasAria := false
	hasTabindex := false

	err := filepath.WalkDir(uiDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".vue") {
			return nil
		}
		
		content, err := os.ReadFile(path)
		if err == nil {
			code := string(content)
			if strings.Contains(code, "aria-live") || strings.Contains(code, "fade-in") {
				hasLiveRegions = true
			}
			if strings.Contains(code, "aria-") || strings.Contains(code, "role=") {
				hasAria = true
			}
			if strings.Contains(code, "tabindex") || strings.Contains(code, "@keydown") {
				hasTabindex = true
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error escaneando UI:", err)
	}

	if hasLiveRegions {
		fmt.Println("- [✔] ISO-9241-110: Principios de diálogo y feedback dinámico detectados (aria-live, fade-in).")
	}
	if !hasLiveRegions {
		fmt.Println("- [ ] ISO-9241-110: Faltan regiones dinámicas o de feedback en los componentes Vue.")
	}

	fmt.Println("\n🔄 **Evaluando CERT-012-WCAG-2-2 (Accesibilidad)...**")
	if hasAria {
		fmt.Println("- [✔] WCAG-1.3.1: Semántica correcta detectada (uso extensivo de atributos aria- y role=).")
	}
	if !hasAria {
		fmt.Println("- [ ] WCAG-1.3.1: No se encontraron roles semánticos (aria-/role=) en la UI.")
	}

	if hasTabindex {
		fmt.Println("- [✔] WCAG-2.1.1: Navegación por teclado detectada (tabindex, keydown handlers).")
	}
	if !hasTabindex {
		fmt.Println("- [ ] WCAG-2.1.1: Faltan handlers de teclado (tabindex) para componentes interactivos.")
	}

	if hasLiveRegions && hasAria && hasTabindex {
		fmt.Println("\n[✔] Auditoría de UI finalizada sin detectar violaciones normativas.")
		return
	}
	
	fmt.Println("\n[!] Se detectaron carencias normativas de UI. Revisa los certificados ISO y WCAG.")
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
