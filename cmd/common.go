package cmd

import "github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"

func parseSettings() common.Settings {
	return common.WithYamlFile()
}
