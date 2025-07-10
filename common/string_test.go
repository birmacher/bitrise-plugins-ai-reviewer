package common

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	longSuggestion    = "This is a very long suggestion that should be wrapped to fit within the specified width of 50 characters. It contains multiple sentences and should be broken down into smaller lines for better readability.\n"
	noIndentationLine = "This line has no indentation.\n"
	spaceIndentedLine = "    This line is indented with spaces.\n"
	tabIndentedLine   = "\t\t\tThis line is indented with a tab.\n"
)

func TestWrapString(t *testing.T) {
	wrapped := WrapString(longSuggestion, 50)
	expected := "This is a very long suggestion that should be\n" +
		"wrapped to fit within the specified width of 50\n" +
		"characters. It contains multiple sentences and\n" +
		"should be broken down into smaller lines for\n" +
		"better readability.\n"
	if wrapped != expected {
		t.Errorf("Expected wrapped string to be:\n%s\n\nGot:\n%s\n",
			expected, wrapped)
	}
}

func TestIndentationForLine(t *testing.T) {
	if GetIndentation(noIndentationLine) != "" {
		t.Error("Expected no indentation for line with no indentation")
	}

	if GetIndentation(spaceIndentedLine) != "    " {
		t.Error("Expected space indentation for line with spaces")
	}

	if GetIndentation(tabIndentedLine) != "\t\t\t" {
		t.Error("Expected tab indentation for line with tabs")
	}
}

func TestDetectLogicalIndent(t *testing.T) {
	tabFile, err := os.ReadFile(filepath.Join("testdata", "test_file_tab.go"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	spaceFile, err := os.ReadFile(filepath.Join("testdata", "test_file_space.swift"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	indentation, repetition := DetectLogicalIndent(string(tabFile))
	if indentation != "\t" {
		t.Errorf("Expected tab indentation, got %s", indentation)
	}
	if repetition != 1 {
		t.Errorf("Expected 1 repetition, got %d", repetition)
	}

	indentation, repetition = DetectLogicalIndent(string(spaceFile))
	if indentation != " " {
		t.Errorf("Expected tab indentation, got %s", indentation)
	}
	if repetition != 4 {
		t.Errorf("Expected 1 repetition, got %d", repetition)
	}
}

func TestGetIndentationString(t *testing.T) {
	tabFile, err := os.ReadFile(filepath.Join("testdata", "test_file_tab.go"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	spaceFile, err := os.ReadFile(filepath.Join("testdata", "test_file_space.swift"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	indentationStr := GetIndentationString(string(tabFile))
	if indentationStr != "\t" {
		t.Errorf("Expected tab indentation string, got %s", indentationStr)
	}
	indentationStr = GetIndentationString(string(spaceFile))
	if indentationStr != "    " {
		t.Errorf("Expected space indentation string, got %s", indentationStr)
	}
}

func TestFixIndentation_Root(t *testing.T) {
	suggestionLines := []string{
		"func example() {",
		"    fmt.Println(\"Hello, World!\")",
		"}",
	}

	fileIndentation := "\t"
	originalLine := "func example() {"

	indentedLines := FixIndentation(fileIndentation, originalLine, suggestionLines)

	expectedLines := []string{
		"func example() {",
		"    fmt.Println(\"Hello, World!\")",
		"}",
	}

	for i, line := range indentedLines {
		if line != expectedLines[i] {
			t.Errorf("Expected line %d to be '%s', got '%s'", i+1, expectedLines[i], line)
		}
	}
}

func TestFixIndentation_TabSameLevel(t *testing.T) {
	suggestionLines := []string{
		"\tfunc example() {",
		"\t\tfmt.Println(\"Hello, World!\")",
		"\t}",
	}

	fileIndentation := "\t"
	originalLine := "\tfunc example() {"

	indentedLines := FixIndentation(fileIndentation, originalLine, suggestionLines)

	expectedLines := []string{
		"	func example() {",
		"		fmt.Println(\"Hello, World!\")",
		"	}",
	}

	for i, line := range indentedLines {
		if line != expectedLines[i] {
			t.Errorf("Expected line %d to be '%s', got '%s'", i+1, expectedLines[i], line)
		}
	}
}

func TestFixIndentation_TabIndented(t *testing.T) {
	suggestionLines := []string{
		"func example() {",
		"	fmt.Println(\"Hello, World!\")",
		"}",
	}

	fileIndentation := "\t"
	originalLine := "\t\tfunc example() {"

	indentedLines := FixIndentation(fileIndentation, originalLine, suggestionLines)

	expectedLines := []string{
		"		func example() {",
		"			fmt.Println(\"Hello, World!\")",
		"		}",
	}

	for i, line := range indentedLines {
		if line != expectedLines[i] {
			t.Errorf("Expected line %d to be '%s', got '%s'", i+1, expectedLines[i], line)
		}
	}
}

func TestFixIndentation_SpaceSameLevel(t *testing.T) {
	suggestionLines := []string{
		"  func example() {",
		"    fmt.Println(\"Hello, World!\")",
		"  }",
	}

	fileIndentation := "  "
	originalLine := "  func example() {"

	indentedLines := FixIndentation(fileIndentation, originalLine, suggestionLines)

	expectedLines := []string{
		"  func example() {",
		"    fmt.Println(\"Hello, World!\")",
		"  }",
	}

	for i, line := range indentedLines {
		if line != expectedLines[i] {
			t.Errorf("Expected line %d to be '%s', got '%s'", i+1, expectedLines[i], line)
		}
	}
}

func TestFixIndentation_SpaceIndented(t *testing.T) {
	suggestionLines := []string{
		"  func example() {",
		"    fmt.Println(\"Hello, World!\")",
		"  }",
	}

	fileIndentation := "  "
	originalLine := "    func example() {"

	indentedLines := FixIndentation(fileIndentation, originalLine, suggestionLines)

	expectedLines := []string{
		"    func example() {",
		"      fmt.Println(\"Hello, World!\")",
		"    }",
	}

	for i, line := range indentedLines {
		if line != expectedLines[i] {
			t.Errorf("Expected line %d to be '%s', got '%s'", i+1, expectedLines[i], line)
		}
	}
}
