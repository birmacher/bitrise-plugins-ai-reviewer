package prompt

import (
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
)

func GetLineLevelFeedbackPrompt(feedbacks []common.LineLevel) string {
	if len(feedbacks) == 0 {
		return ""
	}

	feedbackStr := ""
	for _, feedback := range feedbacks {
		feedbackStr += feedback.StringForAssistant() + "\n\n"
	}
	return `Feedback already given in the PR comments:

` + strings.TrimRight(feedbackStr, "\n")
}
