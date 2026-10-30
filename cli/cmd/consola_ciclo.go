package cmd

import (
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd consola ciclo (RFC 0006 §3.9): el mod de Claude Code lo llama con cada mensaje de la persona. Lo hecho antes sale
// del tablero, el modo de permisos de la IA queda para qdd_project_context y el mensaje cuenta como uso real de qdd.

var consolaModo string

var consolaCicloCmd = &cobra.Command{
	Use:   "ciclo",
	Short: "Marca un ciclo (un mensaje de la persona) con el modo de permisos de la IA; nunca falla ni imprime",
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		registrarUso(root)
		_ = consola.MarcarCiclo(root, consolaModo, time.Now())
	},
}

func init() {
	consolaCicloCmd.Flags().StringVar(&consolaModo, "modo", "", "permission_mode de Claude Code (default, acceptEdits, plan, auto, dontAsk, bypassPermissions)")
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
