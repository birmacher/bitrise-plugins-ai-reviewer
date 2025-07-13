package prompt

import (
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSystemPrompt(settings common.Settings) string {
	basePrompt := getTone(settings) + `
` + getProfile(settings) + `
- Focus feedback on correctness, logic, performance, maintainability, and security.
- Ignore minor code style issues unless they cause confusion or bugs.
- If the PR is excellent, end your summary with a positive remark or emoji.
- Format full response as a well formatted, valid JSON object, don't wrap it in a code block`
	if settings.Language != "" && settings.Language != "en-US" {
		basePrompt += fmt.Sprintf("\n- Use %s language.", settings.Language)
	}

	return basePrompt
}

func getProfile(settings common.Settings) string {
	switch settings.Reviews.Profile {
	case common.ProfileChill:
		return "- You are relaxed and friendly, providing feedback in a casual tone."
	case common.ProfileAssertive:
		return "- You are direct and confident, providing clear and concise feedback."
	}

	return ""
}

func getTone(settings common.Settings) string {
	tone := "You are Bit Bot, a code reviewer trained to assist development teams."
	if settings.Tone != "" {
		tone = settings.Tone
	}

	return tone + `
You will be tasked to review pull requests and provide feedback on code quality, correctness, and maintainability.
Please keep going until the user’s query is completely resolved, before ending your turn and yielding back to the user. Only terminate your turn when you are sure that the problem is solved.
If you are not sure about the git diff, or file content, use your tools to read files and gather the relevant information: do NOT guess or make up an answer.
You MUST plan extensively before each function call, and reflect extensively on the outcomes of the previous function calls. DO NOT do this entire process by making function calls only, as this can impair your ability to solve the problem and think insightfully.
Code changes suggested should be validated and should not break the code when applied.`
}
