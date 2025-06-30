package cmd_test

import (
	"testing"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/version"
)

func TestVersionIsNotEmpty(t *testing.T) {
	if version.Version == "" {
		t.Error("Version should not be empty")
	}
}
