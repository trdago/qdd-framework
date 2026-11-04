package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Modo goal (pkg/consola/goal.go, RFC 0015): el trabajo del proyecto sigue hasta terminar sin esperar a la persona. Lo
// activa solo ella: una IA tiene CLAUDECODE en su entorno y aquí se rechaza. El panel y el teléfono lo limpian al correr
// lo que la persona confirmó. Es un resguardo, no una frontera de seguridad: la IA corre en la misma máquina.

var (
	goalJSON     bool
	goalDesde    string
	goalSilencio int
	goalOcupada  bool
	goalSesion   string
)

var goalCmd = &cobra.Command{
	Use:   "goal",
	Short: "El modo goal del proyecto (activar | desactivar | estado | ciclo)",
	// Corre cada 5 s y funciona en cualquier proyecto con tablero: sin el gatekeeper ni el registro de trabajo de la raíz.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Long: `Con el modo goal activo, qdd elige la opción recomendada de cada pregunta, corre los pedidos de la IA y aprueba
las fichas, sin esperar a la persona. Con 2 min de silencio le recuerda a la IA lo que queda; tras 3 recordatorios sin
avance real se detiene y avisa. Sin una sesión abierta se pausa: nunca abre una.

Siempre esperan a la persona: las preguntas de diseño de un blueprint, las que no traen opción recomendada, los permisos
de Claude Code y los PRs que qdd ofrece solo. Cada elección queda registrada (qdd goal estado).

  qdd goal activar [--desde <dónde>]     lo activa solo la persona: una IA no puede
  qdd goal desactivar
  qdd goal estado [--json]               en qué está y las últimas elecciones
  qdd goal ciclo [--json]                una vuelta del vigilante (lo llama el panel)`,
}

var goalActivarCmd = &cobra.Command{
	Use: "activar", Short: "Activa el modo goal del proyecto (solo la persona)", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(goalActivar(estadoRaiz(), goalDesde, os.Getenv("CLAUDECODE"), time.Now(), os.Stdout, os.Stderr))
	},
}

var goalDesactivarCmd = &cobra.Command{
	Use: "desactivar", Short: "Apaga el modo goal del proyecto", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(goalDesactivar(estadoRaiz(), time.Now(), os.Stdout, os.Stderr))
	},
}

var goalEstadoCmd = &cobra.Command{
	Use: "estado", Short: "Dice en qué está el modo goal y sus últimas elecciones", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(goalEstado(estadoRaiz(), goalJSON, time.Now(), os.Stdout))
	},
}

var goalCicloCmd = &cobra.Command{
	Use: "ciclo", Short: "Una vuelta del vigilante del modo goal (lo llama el panel)", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		entrada := consola.EntradaGoal{Sesion: goalSesion, Silencio: time.Duration(goalSilencio) * time.Second, Ocupada: goalOcupada}
		os.Exit(goalCiclo(estadoRaiz(), entrada, time.Now(), os.Stdout, os.Stderr))
	},
}

func init() {
	goalActivarCmd.Flags().StringVar(&goalDesde, "desde", "terminal", "dónde lo activó la persona")
	goalEstadoCmd.Flags().BoolVar(&goalJSON, "json", false, "salida JSON")
	goalCicloCmd.Flags().BoolVar(&goalJSON, "json", false, "salida JSON")
	goalCicloCmd.Flags().IntVar(&goalSilencio, "silencio", 0, "segundos de silencio de la consola")
	goalCicloCmd.Flags().BoolVar(&goalOcupada, "ocupada", false, "la sesión está haciendo algo")
	goalCicloCmd.Flags().StringVar(&goalSesion, "sesion", "", "la sesión que pregunta")
	goalCmd.AddCommand(goalActivarCmd, goalDesactivarCmd, goalEstadoCmd, goalCicloCmd)
	rootCmd.AddCommand(goalCmd)
}

func goalActivar(root, desde, claudecode string, now time.Time, salida, errores io.Writer) int {
	if claudecode != "" {
		fmt.Fprintln(errores, "[!] El modo goal lo activa solo la persona, desde el panel o el teléfono: una IA no puede.")
		return 1
	}
	if err := consola.ActivarGoal(root, desde, now); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, "[+] Modo goal activo: elige lo recomendado, corre los pedidos y aprueba las fichas sin preguntarte.")
	return 0
}

func goalDesactivar(root string, now time.Time, salida, errores io.Writer) int {
	if err := consola.DesactivarGoal(root, now); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, "[+] Modo goal apagado.")
	return 0
}

func goalEstado(root string, comoJSON bool, now time.Time, salida io.Writer) int {
	g := consola.VistaGoal(root, now)
	if comoJSON {
		return imprimirGoal(salida, g)
	}
	fmt.Fprintln(salida, textoGoal(g))
	return 0
}

func textoGoal(g consola.Goal) string {
	cabeza := "🎯 Modo goal apagado"
	if g.Activo {
		cabeza = "🎯 Modo goal: " + g.Estado
	}
	if g.Motivo != "" {
		cabeza += " (" + g.Motivo + ")"
	}
	if g.Activo {
		cabeza += fmt.Sprintf(" · %d recordatorios seguidos", g.Recordatorios)
	}
	lineas := []string{cabeza}
	for _, e := range g.Elegidas {
		lineas = append(lineas, fmt.Sprintf("  %s  %s: %s → %s", e.En.Format("2006-01-02 15:04"), e.Tipo, e.Etiqueta, e.Eleccion))
	}
	return strings.Join(lineas, "\n")
}

func goalCiclo(root string, entrada consola.EntradaGoal, now time.Time, salida, errores io.Writer) int {
	bin, err := os.Executable()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	r, err := consola.CicloGoal(root, bin, entrada, now)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	return imprimirGoal(salida, r)
}

func imprimirGoal(salida io.Writer, v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 1
	}
	fmt.Fprintln(salida, string(b))
	return 0
}
