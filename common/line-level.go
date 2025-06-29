package common

import (
	"fmt"
	"strings"
)

type LineLevel struct {
	File           string `json:"file"`
	Line           string `json:"content"`
	LineNumber     int
	LastLineNumber int
	Body           string `json:"issue"`
}

type LineLevelFeedback struct {
	Lines []LineLevel `json:"line-feedback"`
}

func (l LineLevel) Header(gitblame string) string {
	lineNumber := fmt.Sprintf("%d", l.LineNumber)
	if l.IsMultiline() {
		lineNumber = fmt.Sprintf("%d-%d", l.LineNumber, l.LastLineNumber)
	}
	return fmt.Sprintf("<!-- bitrise-plugin-ai-reviewer: %s:%s:%s -->", l.File, lineNumber, gitblame)
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
	return strings.Split(l.Line, "\n")[len(strings.Split(l.Line, "\n"))-1]
}
