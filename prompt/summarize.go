package prompt

func GetSummarizePrompt() string {
	return `Provide your final response with the following content:

- **Summary**: A high-level, to-the-point, short summary of the overall change instead of 
specific files within 80 words.
- **Walkthrough**: A markdown table of files and their summaries. Group files 
with similar changes together into a single row to save space.
- **Haiku**: Write a whimsical, short haiku to celebrate the changes as "Bit Bot".
Format the haiku as a quote using the ">" symbol and feel free to use emojis where relevant.

Avoid additional commentary as this summary will be added as a comment on the 
GitHub pull request. Use the title "Summary", "Walkthrough", "Haiku" and it must be H2.`
}
