package prompt

func GetDiffPrompt(diffContent string) string {
	return `

===== PR DIFF =====

` + diffContent + `

===== PR DIFF END =====

`
}
