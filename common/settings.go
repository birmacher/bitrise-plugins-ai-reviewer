package common

import (
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
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

	var filePath string
	filenames := []string{"review.bitrise.yml", "review.bitrise.yaml"}

	for _, name := range filenames {
		if _, err := os.Stat(name); err == nil {
			filePath = name
			break
		}
	}

	if filePath == "" {
		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filePath != "" {
				return filepath.SkipDir
			}
			for _, name := range filenames {
				if !info.IsDir() && info.Name() == name {
					filePath = path
					return filepath.SkipDir
				}
			}
			return nil
		})
	}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			if err := yaml.Unmarshal(data, &settings); err != nil {
				logger.Infof("Failed to parse YAML file %s: %v", filePath, err)
			} else {
				logger.Infof("Using settings from YAML file: %s", filePath)
			}
		}
	} else {
		logger.Infof("No YAML file found in the current directory or subdirectories. Using default settings.")
	}
	return settings
}
