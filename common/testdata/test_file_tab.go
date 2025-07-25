package common

import (
	"bufio"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
)

// WrapString wraps a string to the specified width, attempting to break at spaces.
// It returns a new string with newlines inserted to maintain the specified width.
func WrapString(s string, width int) string {
	var lines []string
	for len(s) > width {
		splitAt := width
		// Try to split at the last space before the specified width
		for i := width; i > 0; i-- {
			if i < len(s) && s[i] == ' ' { // Added length check to prevent index out of range
				splitAt = i
				break
			}
		}
		lines = append(lines, s[:splitAt])
		s = s[splitAt:]
		// Remove leading space
		s = strings.TrimLeft(s, " ")
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n")
}

// GetIndentation returns the leading whitespace (spaces and tabs) of a line.
// It returns an empty string if there is no indentation.
func GetIndentation(line string) string {
	logger.Debug("GetIndentation called with line:", line)
	for i, c := range line {
		if c != ' ' && c != '\t' {
			logger.Debugf("Indentation found: `%s`", line[:i])
			return line[:i]
		}
	}
	logger.Debug("No indentation found")
	return ""
}

// DetectLogicalIndent analyzes text to determine the indentation style (spaces or tabs)
// and the number of indentation characters commonly used.
// Returns the indentation character and count, or empty strings and 0 if no pattern is found.
func DetectLogicalIndent(text string) (string, int) {
	if text == "" {
		return "", 0
	}
	var prevIndentLen int
	var diffs []int
	var indentChar string

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// Get leading whitespace
		leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if leading == "" {
			prevIndentLen = 0
			continue
		}
		// Determine indentation character (prefer the first one found)
		if indentChar == "" && strings.HasPrefix(leading, " ") {
			indentChar = " "
		}
		if indentChar == "" && strings.HasPrefix(leading, "\t") {
			indentChar = "\t"
		}
		// If indent increased, record the diff
		if len(leading) > prevIndentLen {
			diffs = append(diffs, len(leading)-prevIndentLen)
		}
		prevIndentLen = len(leading)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		logger.Debug("Error scanning text:", err)
		return "", 0
	}

	// Find the most common diff
	counts := map[int]int{}
	for _, d := range diffs {
		counts[d]++
	}
	max := 0
	val := 0
	for k, v := range counts {
		if v > max {
			max = v
			val = k
		}
	}
	return indentChar, val
}

// GetIndentationString determines the indentation style of a file and returns
// a string with the appropriate number of indentation characters.
func GetIndentationString(fileSource string) string {
	if fileSource == "" {
		return ""
	}

	indentationType, indentationCount := DetectLogicalIndent(fileSource)
	return strings.Repeat(indentationType, indentationCount)
}
