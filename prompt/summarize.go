package prompt

import (
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSummarizePrompt(settings common.Settings, commitHash, destBranch string) string {
	return `Provide your final response with the following content:
## Pull Request Details
- **Commit Hash**: ` + commitHash + `
- **Destination Branch**: ` + destBranch + `
` + getSummary(settings) + `
` + getWalkthrough(settings) + `
## Line Feedback
Return a list of issues found in the diff hunks, formatted as objects with these fields:
- "file": File path where the issue appears.
- "title": Short title of the issue.
- "category": One of "bug", "refactor", "improvement", "documentation", "nitpick", "test coverage", or "security".
- "issue": Brief description of the issue.
- "content": The exact line from the diff hunk that you are commenting on.
- "prompt": A short, clear instruction for an AI agent to fix the issue (imperative; do not include file or line number).
- "suggestion": (Optional) A valid code snippet that fixes the issue and replaces the code line(s) in \"content\". Must be indented with "\t". Avoid adding just comments or explanations.
Guidelines:
- Only include lines present in the diff hunk. Do not make up or synthesize lines.
- Focus on bugs, code smells, security issues, and code quality improvements. Categorize appropriately.
- For "nitpick", only flag truly minor, non-blocking style suggestions.
- If multiple lines should be replaced, the suggestion should include the full replacement block.
` + getHaiku(settings) + `
---
Avoid additional commentary as the response will be added as a comment on the GitHub pull request.
` + getResponseFormat(settings)
}

func getSummary(settings common.Settings) string {
	if settings.Reviews.Summary {
		return `## Summary
A high-level, to-the-point, short summary of the overall change instead of specific files within 80 words.`
	}
	return ""
}

func getWalkthrough(settings common.Settings) string {
	if settings.Reviews.Walkthrough {
		return `## Walkthrough
A markdown table of file(s) (multiple files should be a string, separated with commas) and their summaries. Group files 
with similar changes together into a single row to save space. Return the file name(s) ("files") and a brief summary of the changes ("summary") in each row.`
	}
	return ""
}

func getHaiku(settings common.Settings) string {
	if settings.Reviews.Haiku {
		return `## Haiku
Write a whimsical, short haiku to celebrate the changes as "Bit Bot".
Format the haiku as a quote using the ">" symbol and feel free to use emojis where relevant.`
	}
	return ""
}

func getResponseFormat(settings common.Settings) string {
	headers := []string{}
	if settings.Reviews.Summary {
		headers = append(headers, "**summary**")
	}
	if settings.Reviews.Walkthrough {
		headers = append(headers, "**walkthrough**")
	}
	headers = append(headers, "**line-feedback**")
	if settings.Reviews.Haiku {
		headers = append(headers, "**haiku**")
	}
	return strings.Join(headers, ", ")
}
