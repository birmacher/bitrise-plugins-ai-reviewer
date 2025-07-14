package prompt

import (
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSummarizePrompt(settings common.Settings, repoOwner, repoName, pr, commitHash, destBranch string) string {
	return `Provide your final response with the following content:
## Pull Request Details
- **Repository**: ` + repoOwner + `/` + repoName + `
- **Pull Request**: ` + pr + `
- **Commit Hash**: ` + commitHash + `
- **Destination Branch**: ` + destBranch + `
## Summary
` + getSummary(settings) + `
## Finished
Once review finished reply with a "done" message, and do not call any more tools.
## Guidelines
- Only include lines present in the diff hunk. Do not make up or synthesize lines.
- Focus on bugs, code smells, security issues, and code quality improvements. Categorize appropriately.
- For "nitpick", only flag truly minor, non-blocking style suggestions.
- If multiple lines should be replaced, the suggestion should include the full replacement block.
- "content" and "suggestion" should be always valid code blocks, formatted with triple backticks ` + "(```)" + `.
- Avoid additional commentary as the response will be added as a comment on the GitHub pull request.
## Task
Can you review PR ` + pr + ` on repo ` + repoOwner + `/` + repoName + ` (commit: ` + commitHash + `, branch: ` + destBranch + `)?`
}

func getSummary(settings common.Settings) string {
	if settings.Reviews.Summary {
		include := []string{}
		if settings.Reviews.Summary {
			include = append(include, "summary")
		}
		if settings.Reviews.Walkthrough {
			include = append(include, "walkthrough")
		}
		if settings.Reviews.Haiku {
			include = append(include, "haiku")
		}

		return `Use the post_summary tool to provide a summary of the changes in the pull request. Include ` + strings.Join(include, ", ") + `.`
	}

	return `Skip sending summary, no need to use the post_summary tool.`
}
