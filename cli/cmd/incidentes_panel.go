package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// FND-189 (MEJ-050): cada vista del panel se dibuja protegida. Si una lanza, el panel muestra «✗ La vista X falló» y
// deja la falla aquí, una vez por vista y mensaje, para que llegue a QDD como cualquier otro incidente. Solo viajan el
// nombre de la vista, el mensaje (anonimizado) y los nombres de las funciones del panel.

const maxMensajeDelPanel = 300

var (
	nombreDelPanel   = regexp.MustCompile(`^[a-z][a-z-]{0,29}$`)
	funcionDelPanel  = regexp.MustCompile(`^[\w$.]{1,80}$`)
	incidenteTrazas  []string
	maxTrazaDelPanel = 8
)

var incidentesPanelCmd = &cobra.Command{
	Use:   "panel <vista> <mensaje> [--traza <función>]…",
	Short: "Registra una vista del panel que falló al dibujarse (lo llama el panel, una vez por falla)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(incidenteDelPanel(args, incidenteTrazas, cmd.OutOrStdout()))
	},
}

func incidenteDelPanel(args, traza []string, out io.Writer) int {
	vista, mensaje := args[0], strings.TrimSpace(args[1])
	if !nombreDelPanel.MatchString(vista) {
		fmt.Fprintf(out, "[!] vista %q: usa minúsculas y guiones\n", vista)
		return 1
	}
	if len(traza) > maxTrazaDelPanel {
		traza = traza[:maxTrazaDelPanel]
	}
	for _, f := range traza {
		if !funcionDelPanel.MatchString(f) {
			fmt.Fprintf(out, "[!] %q no es el nombre de una función del panel\n", f)
			return 1
		}
	}
	if r := []rune(mensaje); len(r) > maxMensajeDelPanel {
		mensaje = string(r[:maxMensajeDelPanel-1]) + "…"
	}
	registrarIncidente(incidentes.TipoPanel, "panel "+vista, "", mensaje, traza, "")
	return 0
}

func init() {
	incidentesPanelCmd.Flags().StringArrayVar(&incidenteTrazas, "traza", nil, "nombre de una función del panel, de la más interna a la más externa")
	incidentesCmd.AddCommand(incidentesPanelCmd)
}
