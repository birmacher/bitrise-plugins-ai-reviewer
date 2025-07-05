package common

import (
	"fmt"
	"strings"
)

// Walkthrough represents information about changes to specific files
type Walkthrough struct {
	Files   string `json:"files"`   // List of files changed
	Summary string `json:"summary"` // Summary of the changes
}

// Summary represents a comprehensive review summary with multiple components
type Summary struct {
	Summary     string        `json:"summary"`     // Overall summary of the changes
	Walkthrough []Walkthrough `json:"walkthrough"` // Detailed walkthrough of individual file changes
	Haiku       string        `json:"haiku"`       // Haiku celebrating the changes
}

// Header returns the HTML comment that identifies this as a summary from the plugin
func (s Summary) Header() string {
	return "[bitrise-plugin-ai-reviewer]: summary"
}

// String formats the complete summary as a markdown string
func (s Summary) String(settings Settings, collapsible bool) string {
	var builder strings.Builder
	builder.WriteString(s.Header() + "\n\n")

	if collapsible && settings.Reviews.CollapseWalkthrough {
		builder.WriteString("<details>\n")
		builder.WriteString("<summary>📝 Summary of changes</summary>\n\n")
	}

	if settings.Reviews.Summary && len(s.Summary) > 0 {
		builder.WriteString(s.Header())
		builder.WriteString("\n\n## Summary\n")
		builder.WriteString(s.Summary + "\n")
	}

	if settings.Reviews.Walkthrough && len(s.Walkthrough) > 0 {
		builder.WriteString("\n\n## Walkthrough\n")
		builder.WriteString(formatWalkthrough(s.Walkthrough) + "\n")
	}

	if collapsible && settings.Reviews.CollapseWalkthrough {
		builder.WriteString("</details>\n\n")
	}

	if settings.Reviews.Haiku && len(s.Haiku) > 0 {
		builder.WriteString("---\n")
		builder.WriteString("### Haiku\n")
		builder.WriteString(s.Haiku + "\n")
	}

	return builder.String()
}

// InitiatedString returns a message indicating the review has started
func (s Summary) InitiatedString() string {
	var builder strings.Builder
	builder.WriteString(s.Header())
	builder.WriteString("\n## Summary\n")
	builder.WriteString(s.Summary)
	builder.WriteString("\n\nReviewing the PR\n\n")
	builder.WriteString("![](https://media2.giphy.com/media/v1.Y2lkPTc5MGI3NjExYWplN3oxMjV0NDc0bW1lazBreGpibHRsZW40emFvZTMydTY2Mjg2bCZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/7NoNw4pMNTvgc/200w.gif)")

	return builder.String()
}

// formatFilePaths splits file paths by comma, truncates each if longer than maxLength,
// and rejoins them with comma
func formatFilePaths(files string, maxLength int) string {
	if len(files) == 0 {
		return ""
	}

	if maxLength <= 3 {
		fmt.Println("Warning: maxLength must be greater than 3 to allow truncation")
		return files
	}

	paths := strings.Split(files, ",")
	for i, path := range paths {
		path = strings.TrimSpace(path)
		if len(path) > maxLength {
			// Find the position to start truncating from
			truncStart := len(path) - maxLength + 3
			if truncStart < 0 {
				paths[i] = path
			} else {
				paths[i] = "..." + path[truncStart:]
			}
		} else {
			paths[i] = path
		}
	}

	return strings.Join(paths, ", ")
}

// formatWalkthrough creates a markdown table from walkthrough data
func formatWalkthrough(walkthrough []Walkthrough) string {
	if len(walkthrough) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("| File | Summary |\n")
	builder.WriteString("|------|---------|\n")

	for _, w := range walkthrough {
		builder.WriteString("| ")
		builder.WriteString(formatFilePaths(w.Files, 40))
		builder.WriteString(" | ")
		builder.WriteString(w.Summary)
		builder.WriteString(" |\n")
	}

	return builder.String()
}
