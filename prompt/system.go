package prompt

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSystemPrompt(settings common.Settings, agent string) string {
	prompt := []string{
		getTone(settings),
		GetProfile(settings),
	}

	if settings.Language != "" && settings.Language != "en-US" {
		prompt = append(prompt, fmt.Sprintf("\n- Use %s language.", settings.Language))
	}

	prompt = append(prompt, getTools(agent))

	return strings.Join(prompt, "\n")
}

func getTone(settings common.Settings) string {
	if settings.Tone != "" {
		return settings.Tone
	}

	return "You are Bit Bot, a code reviewer trained to assist development teams."
}

func getTools(agent string) string {
	prompt := []string{
		`Use tools specified below, do NOT guess or make up an answer.
Please keep going until the user’s query is completely resolved, before ending your turn and yielding back to the user. Only terminate your turn when you are sure that the problem is solved.
You MUST plan extensively before each function call, and reflect extensively on the outcomes of the previous function calls. DO NOT do this entire process by making function calls only, as this can impair your ability to solve the problem and think insightfully.
Code changes suggested should be validated and should not break the code when applied.

## You can use the following tools:`,
	}

	switch agent {
	case "summarize":
		prompt = append(prompt, `- get_pull_request_details: Use to get details about the pull request, such as title, description, and author.
- list_directory: Use to understand the project structure or locate files.
- get_git_diff: See what changed between branches or commits.
- read_file: Use to read any file if the diff is unclear.
- search_codebase: Use if a function, class, or symbol appears in the diff and you want to know where else it is used or defined.
- get_git_blame: Use to see who last modified a line or to understand why a change was made.
- post_line_feedback: Use to post line-level feedback on specific lines of code, including suggestions for improvement.
- post_summary: Use to post a summary of the review findings, including any haiku or walkthrough.

## Follow these steps:
1. **Before Review**
- Get the pull request details first to understand the context.
2. **During Review**
- Get the diff to see what changed with get_git_diff.
- If the diff references a function not defined there, search for it in the codebase with search_codebase.
- If you want to know if a change might break usages elsewhere, search for where it’s used with search_codebase.
- If you want to suggest a refactor, search for all usages with search_codebase.
- If you need context about why something is written a certain way, use get_git_blame.
- After identifying the issues, immediately call post_line_feedback for it, using the exact lines from the diff.
3. **After Review**
- Post a summary of the review findings, including any haiku or walkthrough.`)
	case `build-summary`:
		prompt = append(prompt, `- list_directory: Use to understand the project structure or locate files.
- get_git_diff: See what changed between branches or commits.
- read_file: Use to read any file if additional context is needed.
- search_codebase: Use if a function, class, or symbol appears in the diff and you want to know where else it is used or defined.
- get_git_blame: Use to see who last modified a line or to understand why a change was made.

## Follow these steps:
1. **Before Review**
- Get the build logs understand the context and the error.
2. **During Review**
- Look for the error in the logs and identify the root cause.
- If code changes are needed for additional context, get the diff to see what changed with get_git_diff.
- If the diff references a function not defined there, search for it in the codebase with search_codebase.
- If you want to know if a change might break usages elsewhere, search for where it’s used with search_codebase.
- If you want to suggest a refactor, search for all usages with search_codebase.
- If you need context about why something is written a certain way, use get_git_blame.
- After identifying the issues, immediately call post_line_feedback for it, using the exact lines from the diff.
3. **After Review**
- Summarize the findings clearly and concisely, provide a clear fix only if you are sure. Post to findings with post_build_summary.`)
	}

	return strings.Join(prompt, "\n")
}
