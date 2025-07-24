package cmd

import (
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/spf13/cobra"
)

var (
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "Bitrise Agent - A plugin for agent-based operations",
	Long:  `Bitrise Agent is a CLI plugin for the Bitrise CLI that helps automate tasks using AI.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.Info("Initializing Bitrise Agent with log level:", logLevel)
		logger.SetLevel(logLevel)
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Use --help' for available commands.")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"Set the logging level (debug, info, warn, error, dpanic, panic, fatal)")
}
