package cmd

import (
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review the current code changes against QDD guidelines",
	Run: func(cmd *cobra.Command, args []string) {
		runQCL("review")
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
