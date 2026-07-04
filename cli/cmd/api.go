package cmd

import (
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Audit or generate backend API endpoints following QDD REST/GraphQL standards",
	Run: func(cmd *cobra.Command, args []string) {
		runQCL("api")
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
}
