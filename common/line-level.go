package common

import (
	"fmt"
	"strings"

	"github.com/birmacher/bitrise-plugins-ai-reviewer/git"
)

type LineLevel struct {
	File           string `json:"file"`
	Line           string `json:"content"`
	LineNumber     int    `json:"line"`
	LastLineNumber int    `json:"last_line"`
	Body           string `json:"issue"`
}

type LineLevelFeedback struct {
	Lines []LineLevel `json:"line-feedback"`
}

func (l LineLevel) Header(client *git.Client, commitHash string) string {
	lineNumber := fmt.Sprintf("%d", l.LineNumber)
	if l.IsMultiline() {
		lineNumber = fmt.Sprintf("%d-%d", l.LineNumber, l.LastLineNumber)
	}
	gitBlame, err := client.GetBlameForFileLine(commitHash, l.File, l.LineNumber)
	if err != nil {
		gitBlame = "unknown"
	}
	return fmt.Sprintf("<!-- bitrise-plugin-ai-reviewer: %s:%s:%s -->", l.File, lineNumber, gitBlame)
}

func (l LineLevel) String() string {
	return l.Body
}

func (l LineLevel) IsMultiline() bool {
	return strings.Contains(l.Line, "\n")
}

func (l LineLevel) FirstLine() string {
	return strings.Split(l.Line, "\n")[0]
}

func (l LineLevel) LastLine() string {
	lines := strings.Split(l.Line, "\n")
	return lines[len(lines)-1]
}
