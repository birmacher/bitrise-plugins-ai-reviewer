package main

import (
	"os"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/cmd"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	_ "github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"  // Import for PR review functionality
	_ "github.com/bitrise-io/bitrise-plugins-ai-reviewer/version" // Import for version info
)

func main() {
	// Initialize logger with info level
	logger.Init("info")
	defer logger.Sync()

	if err := cmd.Execute(); err != nil {
		logger.Errorf("Execution failed: %v", err)
		os.Exit(1)
	}
}
