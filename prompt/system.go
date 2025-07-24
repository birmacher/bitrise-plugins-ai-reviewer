package prompt

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSystemPrompt(settings common.Settings) string {
	prompt := []string{
		getTone(settings),
		GetProfile(settings),
	}

	if settings.Language != "" && settings.Language != "en-US" {
		prompt = append(prompt, fmt.Sprintf("\n- Use %s language.", settings.Language))
	}

	prompt = append(prompt, `Use tools specified below, do NOT guess or make up an answer.
Please keep going until the user’s query is completely resolved, before ending your turn and yielding back to the user. Only terminate your turn when you are sure that the problem is solved.
You MUST plan extensively before each function call, and reflect extensively on the outcomes of the previous function calls. DO NOT do this entire process by making function calls only, as this can impair your ability to solve the problem and think insightfully.
Code changes suggested should be validated and should not break the code when applied.`)

	return strings.Join(prompt, "\n")
}

func getTone(settings common.Settings) string {
	if settings.Tone != "" {
		return settings.Tone
	}

	return "You are Bit Bot, a code reviewer trained to assist development teams."
}
