package common

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
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

// Add this function before the main command
func CleanJSONResponse(jsonStr, key string) string {
	// Handle "content" fields
	pattern := fmt.Sprintf(`"%s":\s*"((?:[^"\\]|\\.)*)"\s*,`, key)
	contentRe := regexp.MustCompile(pattern)

	jsonStr = contentRe.ReplaceAllStringFunc(jsonStr, func(match string) string {
		submatches := contentRe.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		// Use json.Marshal to properly escape the content
		escaped, err := json.Marshal(submatches[1])
		if err != nil {
			return match // Return original if marshaling fails
		}
		// Remove the surrounding quotes that json.Marshal adds
		if len(escaped) >= 2 {
			escapedStr := string(escaped[1 : len(escaped)-1])
			return fmt.Sprintf(`"%s": "%s",`, key, escapedStr)
		}
		return match
	})

	// ...existing code...
	return jsonStr
}

// ... (keep existing code until FixJSON)

func FixJSON(jsonStr string) string {
	var builder strings.Builder
	builder.Grow(len(jsonStr))

	inString := false
	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]

		if char == '"' {
			// Check if this quote is escaped
			if i == 0 || jsonStr[i-1] != '\\' {
				inString = !inString
			}
		}

		if inString && char == '\\' {
			// We are inside a string and found a backslash
			if i+1 >= len(jsonStr) {
				// Dangling backslash at the end of the string
				builder.WriteByte(char)
				continue
			}

			nextChar := jsonStr[i+1]
			switch nextChar {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				// This is a valid escape sequence inside a string.
				// Write both characters and skip the next one.
				builder.WriteByte(char)
				builder.WriteByte(nextChar)
				i++
			case 'u':
				// Unicode escape, check for 4 hex digits
				if i+5 < len(jsonStr) {
					builder.WriteString(jsonStr[i : i+6])
					i += 5
				} else {
					builder.WriteByte(char)
				}
			default:
				// This is an invalid escape sequence (e.g., `\s`).
				// Escape the backslash itself.
				builder.WriteString(`\\`)
			}
		} else if inString && char < 32 {
			// Unescaped control character inside a string
			switch char {
			case '\t':
				builder.WriteString(`\t`)
			case '\n':
				builder.WriteString(`\n`)
			case '\r':
				builder.WriteString(`\r`)
			case '\b':
				builder.WriteString(`\b`)
			case '\f':
				builder.WriteString(`\f`)
			default:
				builder.WriteString(fmt.Sprintf(`\u%04x`, char))
			}
		} else {
			// Character is not a special case, just write it.
			builder.WriteByte(char)
		}
	}

	return builder.String()
}

func Base64EncodeJSONValue(jsonStr, key string) string {
	// This regex finds the key and captures its string value, handling escaped quotes.
	pattern := fmt.Sprintf(`"%s":\s*"((?:\\.|[^"\\])*)"`, key)
	re := regexp.MustCompile(pattern)

	return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			// Should not happen if the main regex matched, but as a safeguard.
			return match
		}

		// The captured value is in submatches[1].
		originalValue := submatches[1]

		// Base64 encode the original value.
		encodedValue := base64.StdEncoding.EncodeToString([]byte(originalValue))

		// Return the key with the new Base64 encoded value.
		return fmt.Sprintf(`"%s": "%s"`, key, encodedValue)
	})
}
