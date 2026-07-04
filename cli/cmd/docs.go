package cmd

import (
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Regenerate QDD documentation based on the current project state",
	Run: func(cmd *cobra.Command, args []string) {
		runQCL("docs")
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
