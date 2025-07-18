package prompt

func GetBuildSummaryPrompt(provider, buildID, appID, commitHash string) string {
	return `Adding the context for your task:
## Context
### Build Summary
- **Build ID**: ` + buildID + `
- **App ID**: ` + appID + `
- **CI Provider**: ` + provider + `
- **Commit Hash**: ` + commitHash + `
### Before review
- get_build_log immediately when starting the review.
### During review
- Review the builds logs and the errors in it
- If you need to look up anything in the code use "search_codebase"
- If you need to look up code diff changes, use "get_git_diff"
- If you need additional context about functions, classes, symbols or files, use "read_file" or "get_git_blame"
### Finished
- Post the summary and any optional suggestions with post_build_summary.
- Once summary is posted you should reply with a "done" message, and do not call any more tools.
### Guidelines
- Focus on the build errors, their root causes, and any code, or configuration changes needed to fix them.
- Avoid additional commentary.
## Task
Can you review the log errors for this Build ID ` + buildID + ` with App ID ` + appID + `?`
}
