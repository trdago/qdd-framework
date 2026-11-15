package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// FND-226: sin este subcomando, 'qdd version' llegaba al intérprete de intenciones y mostraba el aviso del motor
// deprecado. Dice lo mismo que 'qdd --version' y, como ese, no pasa por el gatekeeper: sirve en cualquier carpeta.
var versionCmd = &cobra.Command{
	Use:               "version",
	Short:             "Muestra la versión de qdd (igual que qdd --version)",
	Args:              cobra.NoArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "qdd version %s\n", rootCmd.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
