package prompt

func GetFileContentPrompt(fileContents string) string {
	return `

===== CHANGED FILES CONTENT =====

` + fileContents + `

===== CHANGED FILES CONTENT END =====

`
}
