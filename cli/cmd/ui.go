package cmd

import (
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Generate or audit frontend UI components based on QDD SaaS premium standards",
	Run: func(cmd *cobra.Command, args []string) {
		runQCL("ui")
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
