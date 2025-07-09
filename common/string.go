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
func GetIndentationStringFromFileContent(input, targetFile string) string {
	logger.Debugf("Getting indentation for file: %s", targetFile)
	fileSource, err := GetFileContentFromString(input, targetFile)
	if err != nil {
		return ""
	}

	indentationType, indentationCount := DetectLogicalIndent(fileSource)
	logger.Debugf("Detected indentation type: '%s', count: %d", indentationType, indentationCount)

	return strings.Repeat(indentationType, indentationCount)
}

// ReplaceTabIndentation replaces tab indentation with the specified indentation string
// and adds an optional prefix to each line.
func ReplaceTabIndentation(input, indentation, prefix string) string {
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return ""
	}

	baseIndentation := lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " \t"))]

	// Normalize all lines
	for i, line := range lines {
		lines[i] = line[:len(baseIndentation)]
	}

	indentationMarker := "  "
	for i, line := range lines {
		// Count leading indentationMarker occurrences
		count := 0
		for strings.HasPrefix(line, indentationMarker) {
			count++
			line = line[len(indentationMarker):]
		}

		lines[i] = prefix + strings.Repeat(indentation, count) + line
	}

	return strings.Join(lines, "\n")
}
