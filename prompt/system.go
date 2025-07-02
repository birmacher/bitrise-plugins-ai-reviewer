package prompt

import (
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSystemPrompt(settings common.Settings) string {
	basePrompt := `You are Bit Bot, an expert software engineer trained by OpenAI.
Your role is to review code diffs and provide actionable feedback.
Focus on: Logic, Security, Performance, Data Races, Error Handling, Maintainability, Modularity, Complexity, Optimization, and Best Practices like DRY, SOLID, KISS.

Ignore minor style issues or missing comments/documentation.
Return your full response as a well formatted JSON object, don't wrap it in a code block`

	if settings.Language != "en-US" {
		basePrompt += fmt.Sprintf("\nPlease provide your response in %s language.", settings.Language)
	}

	return basePrompt
}
