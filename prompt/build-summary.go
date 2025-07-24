package prompt

func GetBuildSummaryPrompt(provider, buildID, appID, commitHash string) string {
	return `You are helping to identify the error in a CI environment. The goal is to clearly indicate the error and suggest a fix.
## CI Properties
- **CI Provider**: ` + provider + `
- **App ID**: ` + appID + `
- **Build ID**: ` + buildID + `
- **Commit Hash**: ` + commitHash + `
## Task
Please review the log errors for this Build ID ` + buildID + ` with App ID ` + appID + `.`
}

func GetBuildSummaryToolPrompt(availableTools string) string {
	return `## You can use the following tools:
` + availableTools + `
### Guidelines
- Focus on the build errors, their root causes, and any code, or configuration changes needed to fix them.
- Avoid additional commentary.
## Follow these steps:
1. **Before Review**
- Get the build log first to understand the context.
2. **During Review**
- Look for the error in the logs and identify the root cause.
- If code changes are needed for additional context, get the git diff to see what changed.
- If the diff references a function not defined there, search for it in the codebase.
- If you want to know if a change might break usages elsewhere, search for where it’s used in the codebase.
- If you want to suggest a refactor, search for all usages in the codebase.
- If you need context about why something is written a certain way use git blame
3. **After Review**
- Once you have identified the issues, immediately call post the CI Build Summary for it. Summary should be short, clear and concise. Include the error details and any code or configuration changes needed to fix them.
- If you have a suggestion to fix the issue, add it as well.`
}
