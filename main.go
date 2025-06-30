package main

import (
	"os"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/cmd"
	_ "github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"  // Import for PR review functionality
	_ "github.com/bitrise-io/bitrise-plugins-ai-reviewer/version" // Import for version info
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
