package prompt

func GetBuildSummaryPrompt(provider, buildID, appID, commitHash string) string {
	return `Adding the context for your task:
## Context
### Build Summary
- **Build ID**: ` + buildID + `
- **App ID**: ` + appID + `
- **CI Provider**: ` + provider + `
- **Commit Hash**: ` + commitHash + `
### During review
- get_build_log immediately when starting the review.
### Finished
Once the summary and optional suggestion are posted, you should reply with a "done" message, and do not call any more tools.
### Guidelines
- Focus on the build errors, their root causes, and any code, or configuration changes needed to fix them.
- Avoid additional commentary.
## Task
Can you review the log errors for this Build ID ` + buildID + ` with App ID ` + appID + `?`
}
