package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/listas"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Los blueprints a un clic en el panel (pedido del usuario, 2026-11-01): qdd entrega el inventario sin red ni IA.

var consolaBlueprintsJSON bool

var consolaBlueprintsCmd = &cobra.Command{
	Use:   "blueprints",
	Short: "Los blueprints del proyecto para el panel: instalados, disponibles y qué se actualiza sin la IA (sin red)",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(consolaBlueprints(estadoRaiz(), consolaBlueprintsJSON, os.Stdout))
	},
}

func init() {
	consolaBlueprintsCmd.Flags().BoolVar(&consolaBlueprintsJSON, "json", false, "en JSON, para el panel")
	consolaCmd.AddCommand(consolaBlueprintsCmd)
}

func consolaBlueprints(root string, enJSON bool, out io.Writer) int {
	d, err := consola.Blueprints(root)
	if err != nil {
		fmt.Fprintf(out, "[!] %v\n", err)
		return 1
	}
	if enJSON {
		b, err := listas.Marshal(d)
		if err != nil {
			fmt.Fprintf(out, "[!] %v\n", err)
			return 1
		}
		fmt.Fprintln(out, string(b))
		return 0
	}
	inv := d.Inventario
	fmt.Fprintf(out, "%d instalados · %d se actualizan sin la IA · %d cambian la receta · %d disponibles\n", inv.Instalados, inv.SinIA, inv.ConIA, inv.Disponibles)
	for _, e := range inv.Entradas {
		fmt.Fprintf(out, "  %-12s %-28s %s\n", e.Estado, e.ID, strings.TrimSpace(e.Instalada+" → "+e.Publicada))
	}
	for _, p := range inv.Problemas {
		fmt.Fprintf(out, "  [!] %s\n", p)
	}
	return 0
}

// accionesPara: una acción de blueprints se vuelve a calcular con el inventario; las demás, con las del panel.
func accionesPara(root, id string) []consola.Accion {
	if strings.HasPrefix(id, consola.PrefijoBlueprint) {
		return consola.AccionesBlueprints(root)
	}
	return consola.Acciones(root, consola.HayEstado(root))
}
