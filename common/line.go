package common

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ChangedLine struct {
	LineNumber  int
	LineContent string
}

func GetLineNumber(fileName string, fileContent []byte, diffContent []byte, matchLine string) (int, error) {
	fileContentStr, err := getFileContentFromString(string(fileContent), fileName)
	if err != nil {
		return 0, fmt.Errorf("failed to get file content for '%s': %w", fileName, err)
	}

	matches := getMatchingLines([]byte(fileContentStr), matchLine)
	changed := parseDiffChangedLines(diffContent, fileName)

	for _, ln := range matches {
		if changed[ln] {
			return ln, nil
		}
	}

	return 0, fmt.Errorf("line '%s' not found in the diff", matchLine)
}

func getFileContentFromString(input string, targetFile string) (string, error) {
	lines := strings.Split(input, "\n")

	var currentFile string
	var currentContent []string
	found := false

	for _, line := range lines {
		if strings.HasPrefix(line, "===== FILE: ") {
			// If we were collecting the target file, return it now
			if found {
				return strings.TrimRight(strings.Join(currentContent, "\n"), "\n"), nil
			}
			currentFile = strings.TrimSuffix(strings.TrimPrefix(line, "===== FILE: "), " =====")
			currentContent = []string{}
			found = (currentFile == targetFile)
		} else if found {
			currentContent = append(currentContent, line)
		}
	}
	// In case the target file was the last in the input
	if found {
		return strings.TrimRight(strings.Join(currentContent, "\n"), "\n"), nil
	}

	return "", errors.New("file not found: " + targetFile)
}

func getMatchingLines(fileContent []byte, matchLine string) []int {
	scanner := bufio.NewScanner(bytes.NewReader(fileContent))
	var lineNumbers []int
	lineNum := 1
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == strings.TrimSpace(matchLine) {
			lineNumbers = append(lineNumbers, lineNum)
		}
		lineNum++
	}
	return lineNumbers
}

func parseDiffChangedLines(diff []byte, targetFile string) map[int]bool {
	changed := map[int]bool{}
	lines := strings.Split(string(diff), "\n")
	re := regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
	var newLineNum int
	inTargetFile := false
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Diff hunk header
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				newLineNum = atoi(matches[1])
				// The diff hunk may have a range (e.g., +42,7)
			}
			continue
		}
		if strings.HasPrefix(line, "+++") {
			// Example: +++ b/foo.go
			currentFile := strings.TrimPrefix(strings.TrimSpace(line[4:]), "b/")
			inTargetFile = (currentFile == targetFile)
			continue
		}
		if inTargetFile {
			if len(line) == 0 || line[0] == '-' || line[0] == ' ' {
				if len(line) > 0 && line[0] != '-' {
					newLineNum++
				}
				continue
			}
			if line[0] == '+' {
				// This is an added line in the new file (not context)
				changed[newLineNum] = true
				newLineNum++
			}
		}
	}
	return changed
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
