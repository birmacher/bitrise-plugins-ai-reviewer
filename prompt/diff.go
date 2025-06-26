package prompt

func GetDiffPrompt(diffContent string) string {
	return `Below is the raw PR diff for context.

[PR Diff Start]
` + diffContent + `
[PR Diff End]`
}
