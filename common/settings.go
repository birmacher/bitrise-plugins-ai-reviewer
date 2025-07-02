package common

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	ProfileChill     = "chill"
	ProfileAssertive = "assertive"
)

type Reviews struct {
	Profile             string `yaml:"profile"`
	Summary             bool   `yaml:"summary"`
	Walkthrough         bool   `yaml:"walkthrough"`
	CollapseWalkthrough bool   `yaml:"collapse_walkthrough"`
	Haiku               bool   `yaml:"haiku"`
	PathFilters         string `yaml:"path_filters"`
	PathInstructions    string `yaml:"path_instructions"`
}

type Settings struct {
	Language string  `yaml:"language"`
	Tone     string  `yaml:"tone_instructions"`
	Reviews  Reviews `yaml:"reviews"`
}

func WithDefaultSettings() Settings {
	return Settings{
		Language: "en-US",
		Reviews: Reviews{
			Summary:             true,
			Walkthrough:         true,
			CollapseWalkthrough: true,
			Haiku:               true,
			Profile:             ProfileChill,
		},
	}
}

func WithYamlFile() Settings {
	settings := WithDefaultSettings()

	paths := []string{"review.bitrise.yml", "review.bitrise.yaml"}
	var filePath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			filePath = p
			break
		}
	}
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			yaml.Unmarshal(data, &settings)
		}
	}
	return settings
}
