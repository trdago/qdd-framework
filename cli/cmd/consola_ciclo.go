package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd consola ciclo (RFC 0006 §3.9): el mod de Claude Code lo llama con cada mensaje de la persona. Lo hecho antes sale
// del tablero, el modo de permisos de la IA queda para qdd_project_context y el mensaje cuenta como uso real de qdd.

var (
	consolaModo     string
	consolaContexto bool
)

var consolaCicloCmd = &cobra.Command{
	Use:   "ciclo",
	Short: "Marca un ciclo (un mensaje de la persona) con el modo de permisos de la IA; nunca falla; con --contexto imprime qué espera el panel de la persona, solo si cambió",
	Run: func(cmd *cobra.Command, args []string) { correrCiclo(estadoRaiz(), os.Stdout) },
}

// correrCiclo marca el ciclo y, con --contexto, escribe lo que el panel espera de la persona.
func correrCiclo(root string, out io.Writer) {
	registrarUso(root)
	_ = consola.MarcarCiclo(root, consolaModo, time.Now())
	if consolaContexto {
		fmt.Fprint(out, consola.ContextoCiclo(root, consolaSesion, time.Now()))
	}
}

func init() {
	consolaCicloCmd.Flags().BoolVar(&consolaContexto, "contexto", false, "imprime para la IA lo que el panel espera de la persona (pedidos, fichas, ⏳ suyos), solo si cambió desde el último mensaje")
	consolaCicloCmd.Flags().StringVar(&consolaModo, "modo", "", "permission_mode de Claude Code (default, acceptEdits, plan, auto, dontAsk, bypassPermissions)")
	consolaCicloCmd.Flags().StringVar(&consolaSesion, "sesion", "", "sesión de Claude Code que recibe el contexto: se le avisa si se retiró un pedido suyo")
	consolaCmd.AddCommand(consolaCicloCmd)
}

// modoDeLaIA devuelve el modo del último mensaje y su hora, o vacío si no se sabe.
func modoDeLaIA(root string) (string, string) {
	c, ok := consola.ModoVigente(root, time.Now())
	if !ok {
		return "", ""
	}
	return c.Modo, c.En.Local().Format("15:04")
}
