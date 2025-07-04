package cmd

import (
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/spf13/cobra"
)

var (
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "ai-reviewer",
	Short: "Bitrise AI Reviewer - A plugin for code review using AI",
	Long: `Bitrise AI Reviewer is a CLI plugin for the Bitrise CLI that helps review code changes using AI.
It can analyze pull requests and provide feedback, suggestions, and potential issue detection.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetLevel(logLevel)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"Set the logging level (debug, info, warn, error, dpanic, panic, fatal)")
}
