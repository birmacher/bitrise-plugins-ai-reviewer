package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ai-reviewer",
	Short: "Bitrise AI Reviewer - A plugin for code review using AI",
	Long: `Bitrise AI Reviewer is a CLI plugin for the Bitrise CLI that helps review code changes using AI.
It can analyze pull requests and provide feedback, suggestions, and potential issue detection.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommands are provided
		cmd.Help()
	},
}

// Execute runs the root command and handles errors
func Execute() error {
	// Subcommands are added in their respective init() functions
	return rootCmd.Execute()
}

func init() {
	// Add persistent flags that will be available to all subcommands
	// rootCmd.PersistentFlags().StringVar(&someVar, "flag", "default", "description")
}
