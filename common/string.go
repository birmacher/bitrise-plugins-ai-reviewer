package common

import (
	"bufio"
	"strings"
)

func WrapString(s string, width int) string {
	var lines []string
	for len(s) > width {
		splitAt := width
		// Try to split at the last space before the specified width
		for i := width; i > 0; i-- {
			if s[i] == ' ' {
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

func GetIndentation(line string) string {
	for i, c := range line {
		if c != ' ' && c != '\t' {
			return line[:i]
		}
	}
	return ""
}

func DetectLogicalIndent(text string) (string, int) {
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
		// Only spaces or tabs
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

func GetIndentationString(fileSource string) string {
	if fileSource == "" {
		return ""
	}

	indentationType, indentationCount := DetectLogicalIndent(fileSource)
	return strings.Repeat(indentationType, indentationCount)
}

func GetIndentationForLine(line string) string {
	for i, c := range line {
		if c != ' ' && c != '\t' {
			return line[:i]
		}
	}
	return line
}

func ReplaceTabIndentation(input, indentation, prefix string) string {
	if input == "" || indentation == "" {
		return input
	}

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

		if tabCount > 0 {
			// Replace all leading tabs with the correct amount of indentation
			lines[i] = prefix + strings.Repeat(indentation, tabCount) + line[tabCount:]
		}
	}
	return strings.Join(lines, "\n")
}
