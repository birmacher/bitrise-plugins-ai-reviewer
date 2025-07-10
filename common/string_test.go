package common

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	spaceIndentedLine = "    This line is indented with spaces.\n"
	tabIndentedLine   = "\t\t\tThis line is indented with a tab.\n"
)

func TestIndentationForLine(t *testing.T) {
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
