package common

import "strings"

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
