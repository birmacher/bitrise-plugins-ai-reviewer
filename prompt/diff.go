package prompt

func GetDiffPrompt(diffContent string) string {
	return `Below is the raw PR diff for context.

[PR Diff Start]
` + diffContent + `
[PR Diff End]`
}

func GetFileContentPrompt(fileContents string) string {
	return `Here is the full content of the changed files for better context:

` + fileContents
}
