package prompt

func GetSummarizePrompt() string {
	return `Provide your final response with the following content:

- **Summary**: A high-level, to-the-point, short summary of the overall change instead of 
specific files within 80 words.
- **Walkthrough**: A markdown table of files and their summaries. Group files 
with similar changes together into a single row to save space.
- **Line Feedback**: A list of issues found in the diff hunks. Return the file ("file"), issue ("issue") and the exact line content ("content") you are commenting on.
Only include lines that appear in this diff hunk. Do not make up lines. Quote the entire target line exactly as it appears in the diff.
Focus on bugs, smalls, security issues, and code quality improvements.
- **Haiku**: Write a whimsical, short haiku to celebrate the changes as "Bit Bot".
Format the haiku as a quote using the ">" symbol and feel free to use emojis where relevant.

Avoid additional commentary as this summary will be added as a comment on the 
GitHub pull request. Use the json keys as "summary", "walkthrough", "line-feedback", "haiku".`
}
