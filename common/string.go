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

	// First pass: detect indentation character type
	indentChar := detectIndentationCharacter(text)
	if indentChar == "" {
		return "", 0
	}

	// Second pass: collect indentation levels using the detected character
	var indentLevels []int
	var prevIndentLevel int

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue // Skip empty lines
		}

		// Count indentation using only the detected character
		currentIndentLevel := countIndentationCharacters(line, indentChar)

		// Record level changes (both increases and decreases)
		if currentIndentLevel != prevIndentLevel {
			indentLevels = append(indentLevels, currentIndentLevel)
		}
		prevIndentLevel = currentIndentLevel
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		logger.Debug("Error scanning text:", err)
		return "", 0
	}

	// Calculate the most common indentation increment
	indentSize := calculateIndentationSize(indentLevels)
	return indentChar, indentSize
}

// detectIndentationCharacter determines whether the text uses spaces or tabs for indentation
func detectIndentationCharacter(text string) string {
	spaceIndentedLines := 0
	tabIndentedLines := 0

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue // Skip empty lines
		}

		// Check if line starts with indentation
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			switch line[0] {
			case ' ':
				spaceIndentedLines++
			case '\t':
				tabIndentedLines++
			}
		}
	}

	// Return the more common indentation character
	if spaceIndentedLines > tabIndentedLines {
		return " "
	} else if tabIndentedLines > 0 {
		return "\t"
	}
	return ""
}

// countIndentationCharacters counts the number of indentation characters at the start of a line
func countIndentationCharacters(line, indentChar string) int {
	count := 0
	for i := 0; i < len(line); i++ {
		if string(line[i]) == indentChar {
			count++
		} else {
			break
		}
	}
	return count
}

// calculateIndentationSize determines the most common indentation size from a list of indentation levels
func calculateIndentationSize(levels []int) int {
	if len(levels) == 0 {
		return 0
	}

	// Calculate differences between consecutive levels
	var diffs []int
	for i := 1; i < len(levels); i++ {
		diff := levels[i] - levels[i-1]
		if diff > 0 { // Only consider positive differences (indentation increases)
			diffs = append(diffs, diff)
		}
	}

	if len(diffs) == 0 {
		return 0
	}

	// Find the most common positive difference
	counts := make(map[int]int)
	for _, d := range diffs {
		counts[d]++
	}

	maxCount := 0
	mostCommonDiff := 0
	for diff, count := range counts {
		if count > maxCount {
			maxCount = count
			mostCommonDiff = diff
		}
	}

	return mostCommonDiff
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

// ReplaceTabIndentation replaces tab indentation with the specified indentation string
// and adds an optional prefix to each line.
func ReplaceTabIndentation(input, indentation, prefix string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		// Count leading tabs
		tabCount := 0
		for j := 0; j < len(line); j++ {
			if line[j] == '\t' {
				tabCount++
			} else {
				break
			}
		}

		lines[i] = prefix + strings.Repeat(indentation, tabCount) + line[tabCount:]
	}
	return strings.Join(lines, "\n")
}
