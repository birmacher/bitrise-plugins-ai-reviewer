package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
)

const (
	CategoryBug           = "bug"
	CategoryRefactor      = "refactor"
	CategoryImprovement   = "improvement"
	CategoryDocumentation = "documentation"
	CategoryNitpick       = "nitpick"
	CategoryTestCoverage  = "test coverage"
	CategorySecurity      = "security"
)

// LineLevel represents a review comment for a specific line of code
type LineLevel struct {
	File           string `json:"file"`                  // Path to the file being commented on
	Line           string `json:"content"`               // Content of the line being commented on
	Category       string `json:"category,omitempty"`    // Category of the issue (e.g., "bug", "style", "performance")
	LineNumber     int    `json:"line"`                  // Line number in the file
	LastLineNumber int    `json:"last_line"`             // Last line number for multi-line comments
	Suggestion     string `json:"suggestion,omitempty"`  // Suggested replacement for the line
	Title          string `json:"title,omitempty"`       // Short title for the issue
	Body           string `json:"issue"`                 // Main body of the review comment
	CommitHash     string `json:"commit_hash,omitempty"` // Commit hash for the line being commented on
	Prompt         string `json:"prompt,omitempty"`      // Optional prompt for AI agents to fix the issue
}

// LineLevelFeedback represents a collection of line-level feedback items
type LineLevelFeedback struct {
	Lines []LineLevel `json:"line-feedback"` // List of line-level feedback items
}

func ParseLineLevelFeedback(jsonData string) (LineLevelFeedback, error) {
	llf := LineLevelFeedback{}

	// Encode vialators as the LLM can respond with invalid JSON ( new lines and tabs)
	jsonData = EncodeLLMKey(jsonData, "content")
	jsonData = EncodeLLMKey(jsonData, "suggestion")
	jsonData = EncodeLLMKey(jsonData, "haiku")

	if err := json.Unmarshal([]byte(jsonData), &llf); err != nil {
		return llf, fmt.Errorf("failed to parse LineLevelFeedback JSON: %v", err)
	}

	for idx, line := range llf.Lines {
		// Decode the line content
		decodedLine, err := DecodeLLMValue(line.Line)
		if err != nil {
			return llf, fmt.Errorf("failed to decode line content for file %s: %v", line.File, err)
		}
		llf.Lines[idx].Line = decodedLine

		// Decode the suggestion if it exists
		if line.Suggestion != "" {
			decodedSuggestion, err := DecodeLLMValue(line.Suggestion)
			if err != nil {
				return llf, fmt.Errorf("failed to decode suggestion for file %s: %v", line.File, err)
			}
			llf.Lines[idx].Suggestion = decodedSuggestion
		}
	}

	return llf, nil
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

	return fmt.Sprintf("[bitrise-plugin-ai-reviewer]: %s:%s:%s", l.File, lineNumber, gitBlame)
}

// String formats the complete comment with header, body and suggestion
func (l LineLevel) String(provider string, client *git.Client, commitHash string) string {
	if l.File == "" || l.LineNumber <= 0 || l.Body == "" {
		return ""
	}

	body := []string{}

	// Setup title
	title := []string{}
	if category := l.getCategoryString(); category != "" {
		title = append(title, category)
	}
	if l.Title != "" {
		title = append(title, l.Title)
	}
	if len(title) > 0 {
		body = append(body, fmt.Sprintf("**%s**", strings.Join(title, ": ")))
	}

	// Setup issue body
	body = append(body, l.Body)

	// Setup helpers
	if provider == "bitbucket" {
		if len(l.Prompt) > 0 {
			body = append(body, fmt.Sprintf("🤖 Prompt for AI Agents:\n\n```\n%s\n```\n\n", l.getAIPrompt()))
		}
	} else {
		if len(l.getCategoryString()) > 0 && l.getCategoryString() != CategoryNitpick && len(l.Prompt) > 0 {
			body = append(body, fmt.Sprintf("<details>\n<summary>🤖 Prompt for AI Agents:</summary>\n\n```\n%s\n```\n\n</details>", l.getAIPrompt()))
		}
	}

	if len(l.Suggestion) > 0 {
		var suggestionStr string
		switch provider {
		case "bitbucket":
			suggestionStr = "Replace with the following code:\n\n"
			suggestionStr += "Current implementation\n"
			suggestionStr += fmt.Sprintf("```\n%s\n```", l.Line)
			suggestionStr += "\n\n"
			suggestionStr += "Suggested changes\n"
			suggestionStr += fmt.Sprintf("```\n%s\n```", l.Suggestion)
		case "github":
			suggestionStr = "```suggestion\n" + l.Suggestion + "\n```"
		}
		body = append(body, fmt.Sprintf("🔄 Suggestion:\n%s", suggestionStr))
	}
	return fmt.Sprintf("%s\n%s", l.Header(client, commitHash), strings.Join(body, "\n\n"))
}

func (l LineLevel) StringForAssistant() string {
	return `===== Line Level Feedback On File: ` + l.File + ` =====
` + l.Body + `
===== END =====`
}

// IsMultiline checks if the line content spans multiple lines
func (l LineLevel) IsMultiline() bool {
	if l.Line == "" {
		return false
	}

	// Normalize line endings to \n and strip trailing newlines
	normalizedLine := strings.ReplaceAll(strings.ReplaceAll(l.Line, "\r\n", "\n"), "\r", "\n")
	trimmedLine := strings.TrimRight(normalizedLine, "\n")

	return len(strings.Split(trimmedLine, "\n")) > 1
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

func (l LineLevel) getCategoryString() string {
	switch l.Category {
	case CategoryBug:
		return "🐛 Bug"
	case CategoryRefactor:
		return "🔧 Refactor Suggestion"
	case CategoryImprovement:
		return "💡 Improvement"
	case CategoryDocumentation:
		return "📚 Documentation"
	case CategoryNitpick:
		return "🧹 Nitpick"
	case CategoryTestCoverage:
		return "🧪 Test Coverage"
	case CategorySecurity:
		return "🔒 Security Issue"
	}

	return ""
}

func (llf LineLevelFeedback) GetNitpickFeedback() []LineLevel {
	var nitpicks []LineLevel
	for _, line := range llf.Lines {
		if line.Category == CategoryNitpick {
			nitpicks = append(nitpicks, line)
		}
	}
	return nitpicks
}

func (llf LineLevelFeedback) GetLineFeedback() []LineLevel {
	var feedback []LineLevel
	for _, line := range llf.Lines {
		if line.Category != CategoryNitpick {
			feedback = append(feedback, line)
		}
	}
	return feedback
}

func (ll LineLevel) getAIPrompt() string {
	if ll.Prompt == "" || ll.File == "" || ll.LineNumber <= 0 {
		return ""
	}

	line := fmt.Sprintf("line %d", ll.LineNumber)
	if ll.IsMultiline() && ll.LastLineNumber > ll.LineNumber {
		line = fmt.Sprintf("lines %d and %d", ll.LineNumber, ll.LastLineNumber)
	}

	result := fmt.Sprintf("In %s at %s, %s", ll.File, line, ll.Prompt)
	return WrapString(result, 80)
}
