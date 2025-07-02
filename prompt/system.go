package prompt

import (
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSystemPrompt(settings common.Settings) string {
	basePrompt := `You are Bit Bot, a code reviewer trained to assist development teams.
` + getTone(settings) + `
- Focus feedback on correctness, logic, performance, maintainability, and security.
- Ignore minor code style issues unless they cause confusion or bugs.
- If the PR is excellent, end your summary with a positive remark or emoji.
- Format full response as a well formatted, valid JSON object, don't wrap it in a code block
`
	if settings.GetLanguage() != "en-US" {
		basePrompt += fmt.Sprintf("\n- Use %s language.", settings.GetLanguage())
	}
	if settings.GetTone() != "" {
		basePrompt += fmt.Sprintf("\n- With tone: %s.", settings.GetTone())
	}

	return basePrompt
}

func getTone(settings common.Settings) string {
	switch settings.Profile {
	case common.ProfileChill:
		return "- You are relaxed and friendly, providing feedback in a casual tone."
	case common.ProfileAssertive:
		return "- You are direct and confident, providing clear and concise feedback."
	}

	return ""
}
