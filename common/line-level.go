package common

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
)

// LineLevel represents a review comment for a specific line of code
type LineLevel struct {
	File           string `json:"file"`                  // Path to the file being commented on
	Line           string `json:"content"`               // Content of the line being commented on
	LineNumber     int    `json:"line"`                  // Line number in the file
	LastLineNumber int    `json:"last_line"`             // Last line number for multi-line comments
	Suggestion     string `json:"suggestion,omitempty"`  // Suggested replacement for the line
	Body           string `json:"issue"`                 // Main body of the review comment
	CommitHash     string `json:"commit_hash,omitempty"` // Commit hash for the line being commented on
}

// LineLevelFeedback represents a collection of line-level feedback items
type LineLevelFeedback struct {
	Lines []LineLevel `json:"line-feedback"` // List of line-level feedback items
}

// Header generates a header string for the comment with file, line and blame information
func (l LineLevel) Header(client *git.Client, commitHash string) string {
	lineNumber := fmt.Sprintf("%d", l.LineNumber)
	if l.IsMultiline() {
		lineNumber = fmt.Sprintf("%d-%d", l.LineNumber, l.LastLineNumber)
	}

	gitBlame := "unknown"
	if client != nil {
		blame, err := client.GetBlameForFileLine(commitHash, l.File, l.LineNumber)
		if err == nil {
			gitBlame = blame
		}
	}

	return fmt.Sprintf("<!-- bitrise-plugin-ai-reviewer: %s:%s:%s -->", l.File, lineNumber, gitBlame)
}

// String formats the complete comment with header, body and suggestion
func (l LineLevel) String(client *git.Client, commitHash string) string {
	body := l.Body
	if len(l.Suggestion) > 0 {
		body += fmt.Sprintf("\n\n**Suggestion:**\n```suggestion\n%s\n```\n", l.Suggestion)
	}
	return fmt.Sprintf("%s\n%s", l.Header(client, commitHash), body)
}

func (l LineLevel) StringForAssistant() string {
	return `===== Line Level Feedback On File: ` + l.File + ` =====
` + l.Body + `
===== END =====`
}

// IsMultiline checks if the line content spans multiple lines
func (l LineLevel) IsMultiline() bool {
	return l.Line != "" && strings.Contains(l.Line, "\n")
}

// FirstLine returns the first line of the content
func (l LineLevel) FirstLine() string {
	if l.Line == "" {
		return ""
	}
	return strings.Split(l.Line, "\n")[0]
}

// LastLine returns the last line of the content
func (l LineLevel) LastLine() string {
	if l.Line == "" {
		return ""
	}
	lines := strings.Split(l.Line, "\n")
	return lines[len(lines)-1]
}
