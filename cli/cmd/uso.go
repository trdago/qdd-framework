package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/uso"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd uso (pedido del usuario, 2026-10-30): el uso real medido de cada versión de qdd. Una minor o una major sale
// primero como preview y pasa a latest con las horas que pide uso_minimo_horas (VER-05).

var usoCmd = &cobra.Command{
	Use:   "uso",
	Short: "Uso real medido de cada versión de qdd (lo que pide VER-05 para pasar de preview a latest)",
	Long: `qdd anota su uso real en los proyectos QDD: comandos, herramientas MCP, pulsos del agente y cada mensaje en una
sesión con el panel. Dos eventos separados por menos de 5 minutos son uso continuo; un evento aislado no suma.

  qdd uso                       lo medido de cada versión, y la meta de la versión que corre
  qdd uso evidencia <versión>   deja las horas medidas en .qdd/project/evidence/uso/ para publicar (VER-05)`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		imprimirUso(os.Stdout, cmd.Root().Version, audit.LoadPolicies(integration.FindProjectRoot(cwd)).UsoMinimoHoras)
	},
}

var usoEvidenciaCmd = &cobra.Command{
	Use:   "evidencia <versión>",
	Short: "Deja en el proyecto las horas de uso real medidas de una versión y sus previews",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		ruta, ev, err := uso.Evidencia(integration.FindProjectRoot(cwd), args[0], time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] %s: %.2f h de uso real medido (%s)\n", ev.Version, ev.Horas, strings.Join(ev.Versiones, ", "))
		fmt.Printf("[+] Evidencia en %s: commitéala con la release.\n", ruta)
	},
}

func init() {
	usoCmd.AddCommand(usoEvidenciaCmd)
	rootCmd.AddCommand(usoCmd)
}

// registrarUso anota un evento de uso real de la versión que corre si dir está en un proyecto QDD. Nunca falla ni
// demora: el uso se mide, no estorba.
func registrarUso(dir string) {
	registrarUsoEn(Version, dir, time.Now())
}

func registrarUsoEn(version, dir string, now time.Time) {
	root := integration.FindProjectRoot(dir)
	if _, err := os.Stat(filepath.Join(root, ".qdd")); err != nil {
		return
	}
	_ = uso.Registrar(version, filepath.Base(estado.ClaveLocal(root)), now)
}

func imprimirUso(out io.Writer, version string, minimo float64) {
	usos, ilegibles := uso.Todas()
	if len(usos) == 0 {
		fmt.Fprintln(out, "Sin uso medido todavía: se anota al usar qdd en un proyecto QDD.")
	}
	for _, u := range usos {
		fmt.Fprintf(out, "  %-14s %s  (desde %s)\n", u.Version, duracion(u.Duracion()), u.Desde)
	}
	for _, n := range ilegibles {
		fmt.Fprintf(out, "  [!] %s ilegible: qdd doctor --fix lo aparta\n", n)
	}
	if minimo <= 0 || version == "" || version == "dev" {
		return
	}
	total, _ := uso.Total(version)
	fmt.Fprintf(out, "\n%s: %s de %.0f h de uso real para pasar a latest (VER-05; un fix no espera).\n", strings.TrimPrefix(version, "v"), duracion(total), minimo)
}

func duracion(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %02d min", h, m)
}
