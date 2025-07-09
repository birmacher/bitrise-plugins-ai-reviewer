package prompt

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetSummarizePrompt(settings common.Settings) string {
	return `Provide your final response with the following content:
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
- (Optional) "suggestion": A valid code snippet that fully replaces the line(s) in "content". Only provide a suggestion if you know the correct fix. Match the indentation style of the project. Should be correctly indented, always with "\t".
Guidelines:
- Only include lines present in the diff hunk. Do not make up or synthesize lines.
- Focus on bugs, code smells, security issues, and code quality improvements. Categorize appropriately.
- For "nitpick", only flag truly minor, non-blocking style suggestions.
- If multiple lines should be replaced, the suggestion should include the full replacement block.
` + getHaiku(settings) + `
---
Avoid additional commentary as the response will be added as a comment on the GitHub pull request.
Example response:
` + getExampleResponse(settings)
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
An array of objects, each containing:
- "files": A string of file names (multiple files should be comma-separated). Group files 
with similar changes together into a single row to save space.
- "summary": A brief summary of the changes made in that file.`
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

func getExampleResponse(settings common.Settings) string {
	response := make([]string, 0)

	if settings.Reviews.Summary {
		response = append(response, "summary: \"...\"")
	}
	if settings.Reviews.Walkthrough {
		response = append(response, "walkthrough: [\n{\nfiles: \"...\",\nsummary: \"...\"\n}\n]")
	}
	response = append(response, "line_feedback: [\n{\nfile: \"...\",\ntitle: \"...\",\ncategory: \"...\",\nissue: \"...\",\ncontent: \"...\",\nprompt: \"...\",\nsuggestion: \"...\"\n}\n]")
	if settings.Reviews.Haiku {
		response = append(response, "haiku: \"...\"")
	}

	return fmt.Sprintf("{\n%s\n}", strings.Join(response, ",\n"))
}
