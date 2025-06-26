package main

import (
	"os"

	"github.com/birmacher/bitrise-plugins-ai-reviewer/cmd"
	_ "github.com/birmacher/bitrise-plugins-ai-reviewer/version" // Import for version info
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
