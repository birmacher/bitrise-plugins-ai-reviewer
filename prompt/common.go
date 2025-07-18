package prompt

import "github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"

func GetProfile(settings common.Settings) string {
	switch settings.Reviews.Profile {
	case common.ProfileChill:
		return "- You are relaxed and friendly, providing feedback in a casual tone."
	case common.ProfileAssertive:
		return "- You are direct and confident, providing clear and concise feedback."
	}

	return ""
}
