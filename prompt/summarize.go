package prompt

func GetSummarizePrompt() string {
	return `Provide your final response with the following content:

- **Summary**: A high-level, to-the-point, short summary of the overall change instead of 
specific files within 80 words.
- **Walkthrough**: A markdown table of files and their summaries. Group files 
with similar changes together into a single row to save space.
- **Line Feedback**: A list of specific line feedback only for the diff, with each item containing:
file name, line number (use the git diff header to identify the correct line number), and a short description of the issue. You can include code suggestions
if you are confident in your suggestions.
- **Haiku**: Write a whimsical, short haiku to celebrate the changes as "Bit Bot".
Format the haiku as a quote using the ">" symbol and feel free to use emojis where relevant.

Avoid additional commentary as this summary will be added as a comment on the 
GitHub pull request. Use the json keys as "summary", "walkthrough", "line-feedback", "haiku".`
}
