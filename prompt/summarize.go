package prompt

func GetSummarizePrompt(lineFeedback string) string {
	return `Provide your final response with the following content:

## Summary
A high-level, to-the-point, short summary of the overall change instead of specific files within 80 words.

## Walkthrough
A markdown table of file(s) (multiple files should be a string, separated with commas) and their summaries. Group files 
with similar changes together into a single row to save space. Return the file name(s) ("files") and a brief summary of the changes ("summary") in each row.

## Line Feedback
A list of issues found in the diff hunks. Return the file ("file"), issue ("issue") and the exact line content ("content") you are commenting on.
Only include lines that appear in the diff hunk. Only include feedback for lines that are additions in the diff. Do not make up lines.
Quote the entire target line exactly as it appears in the diff.
If you are sure how to fix the issue, you can include a "suggestion" field with a code snippet that fixes the issue. The suggestion should replace the flagged line(s) content.
Focus on bugs, smells, security issues, and code quality improvements.

Line Feedback that has been already added to the PR can be found below and should not be repeated - avoid the same or similar feedback.
` + lineFeedback + `

## Haiku
Write a whimsical, short haiku to celebrate the changes as "Bit Bot".
Format the haiku as a quote using the ">" symbol and feel free to use emojis where relevant.

Avoid additional commentary as this summary will be added as a comment on the 
GitHub pull request. Use the json keys as "summary", "walkthrough", "line-feedback", "haiku".`
}
