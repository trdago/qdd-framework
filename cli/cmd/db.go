package cmd

import (
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Audit or generate database schemas and queries",
	Run: func(cmd *cobra.Command, args []string) {
		runQCL("db")
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)
}
