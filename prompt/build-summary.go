package prompt

func GetBuildSummaryPrompt(buildID, appID string) string {
	return `Provide your final response with the following content:
## Build Summary
- **Build ID**: {{build_id}}
- **App ID**: {{app_id}}
- **CI Provider**: {{ci_provider}}
## During review
- get_build_log immediately when starting the review.
## Finished
Once the summary and optional suggestion are posted, you should reply with a "done" message, and do not call any more tools.
## Guidelines
- Focus on the build errors, their root causes, and any code, or configuration changes needed to fix them.
- Avoid additional commentary.
## Task
Can you review the log errors for this Build ID ` + buildID + ` with App ID ` + appID + `?`
}
