package common

import "strings"

type Walkthrough struct {
	Files   string `json:"files"`   // List of files changed
	Summary string `json:"summary"` // Summary of the changes
}

type Summary struct {
	Summary     string        `json:"summary"`     // Summary of the response
	Walkthrough []Walkthrough `json:"walkthrough"` // Walkthrough of the changes
	Haiku       string        `json:"haiku"`       // Haiku celebrating the changes
}

func (s Summary) Header() string {
	return "<!-- bitrise-plugin-ai-reviewer: summary -->"
}

func (s Summary) String() string {
	if s.Summary == "" {
		return s.InitiatedString()
	}

	summaryBody := []string{
		s.Header(),
		"## Summary\n" + s.Summary,
	}
	if len(s.Walkthrough) > 0 {
		summaryBody = append(summaryBody, "## Walkthrough\n"+formatWalkthrough(s.Walkthrough))
	}
	if len(s.Haiku) > 0 {
		summaryBody = append(summaryBody, "## Haiku\n"+s.Haiku)
	}

	return strings.Join(summaryBody, "\n\n")
}

func (s Summary) InitiatedString() string {
	return s.Header() + "\n" +
		"## Summary\n" + s.Summary + "\n\n" +
		"Reviewing the PR\n\n" +
		"![](https://media2.giphy.com/media/v1.Y2lkPTc5MGI3NjExYWplN3oxMjV0NDc0bW1lazBreGpibHRsZW40emFvZTMydTY2Mjg2bCZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/7NoNw4pMNTvgc/200w.gif)\n\n"
}

func formatWalkthrough(walkthrough []Walkthrough) string {
	result := "| File | Summary |\n"
	result += "|------|---------|\n"
	for _, w := range walkthrough {
		result += "| " + w.Files + " | " + w.Summary + " |\n"
	}
	return result
}
