package prompt

func GetCommentsPrompt(comments string) string {
	return `Here are all the PR Line Level Reviews. When suggesting code changes, make sure you don't duplicate any of these reviews as it would spam the PR:

` + comments
}
