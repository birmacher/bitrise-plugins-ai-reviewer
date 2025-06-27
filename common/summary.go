package common

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
	return s.Header() + "\n" +
		"## Summary\n" + s.Summary + "\n\n" +
		"## Walkthrough\n" + formatWalkthrough(s.Walkthrough) + "\n\n" +
		"## Haiku\n" + s.Haiku
}

func formatWalkthrough(walkthrough []Walkthrough) string {
	result := "| File | Summary |\n"
	result += "|------|---------|\n"
	for _, w := range walkthrough {
		result += "| " + w.Files + " | " + w.Summary + " |\n"
	}
	return result
}
