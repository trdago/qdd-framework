package cmd

import (
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check for structural and architectural anomalies in the codebase",
	Run: func(cmd *cobra.Command, args []string) {
		runQCL("validate")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
