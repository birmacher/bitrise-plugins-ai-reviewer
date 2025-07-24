package prompt

import (
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSummarizePrompt(settings common.Settings, repoOwner, repoName, pr, commitHash, destBranch string) string {
	return `Provide your final response with the following content:
## Pull Request Details
- **Repository**: ` + repoOwner + `/` + repoName + `
- **Pull Request**: ` + pr + `
- **Commit Hash**: ` + commitHash + `
- **Destination Branch**: ` + destBranch + `
## Task
Can you review PR ` + pr + ` on repo ` + repoOwner + `/` + repoName + ` (commit: ` + commitHash + `, branch: ` + destBranch + `)?`
}

func GetPRSummaryToolPrompt(availableTools string) string {
	return `## You can use the following tools:
` + availableTools + `
## Guidelines
- Only include lines present in the diff hunk. Do not make up or synthesize lines.
- Focus on bugs, code smells, security issues, and code quality improvements. Categorize appropriately.
- For "nitpick", only flag truly minor, non-blocking style suggestions.
- If multiple lines should be replaced, the suggestion should include the full replacement block.
- Avoid additional commentary as the response will be added as a comment on the GitHub pull request.
## Follow these steps:
1. **Before Review**
- Get the pull request details first to understand the context.
2. **During Review**
- Get the diff to see what has changed.
- If the diff references a function not defined there, search for it in the codebase.
- If you want to know if a change might break usages elsewhere, search for it in the codebase.
- If you want to suggest a refactor, search for all usages in the codebase.
- If you need context about why something is written a certain way, search for it in the codebase.
- After identifying the issues, immediately post line feedback for it, using the exact lines from the diff.
3. **After Review**
- Post a summary of the review findings, including any haiku or walkthrough.`
}
