package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd consola revisar (pedido de la persona, 2026-11-26): la tecla l del panel lo llama y le manda el texto a la IA,
// que vuelve a evaluar todo lo que espera a la persona y deja solo lo que aporta valor (pkg/consola/revisar.go).

var consolaRevisarCmd = &cobra.Command{
	Use:   "revisar",
	Short: "Lo que espera a la persona, como mensaje para que la IA lo vuelva a evaluar (JSON: texto y total); solo lee",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(revisarPanel(estadoRaiz(), time.Now(), os.Stdout))
	},
}

func init() {
	consolaRevisarCmd.Flags().Bool("json", true, "salida JSON (la única que hay)")
	consolaCmd.AddCommand(consolaRevisarCmd)
}

func revisarPanel(root string, now time.Time, salida io.Writer) int {
	b, err := json.Marshal(consola.Revision(root, now))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, string(b))
	return 0
}
