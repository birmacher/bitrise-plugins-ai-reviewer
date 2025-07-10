package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsingLineLevelFeedbackResponse(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_response.json"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	summary, err := ParseLineLevelFeedback(string(data))
	if err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if len(summary.Lines) == 0 {
		t.Error("Expected summary to be non-empty")
	}

	if summary.Lines[0].File == "" {
		t.Error("Expected file name to be non-empty")
	}

	if summary.Lines[0].Body == "" {
		t.Error("Expected body to be non-empty")
	}

	if summary.Lines[0].Suggestion == "" {
		t.Error("Expected suggestion to be non-empty")
	}

	if summary.Lines[0].Category == "" {
		t.Error("Expected category to be non-empty")
	}

	if summary.Lines[0].Title == "" {
		t.Error("Expected title to be non-empty")
	}

	if summary.Lines[0].Line == "" {
		t.Error("Expected line to be non-empty")
	}
}

func TestParsingLineLevelFeedbackResponse_NewLine(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_response.json"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	summary, err := ParseLineLevelFeedback(string(data))
	if err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if len(summary.Lines) == 0 {
		t.Error("Expected summary to be non-empty")
	}

	if summary.Lines[0].File == "" {
		t.Error("Expected file name to be non-empty")
	}

	if summary.Lines[0].Body == "" {
		t.Error("Expected body to be non-empty")
	}

	if summary.Lines[0].Suggestion == "" {
		t.Error("Expected suggestion to be non-empty")
	}

	if summary.Lines[0].Category == "" {
		t.Error("Expected category to be non-empty")
	}

	if summary.Lines[0].Title == "" {
		t.Error("Expected title to be non-empty")
	}

	if summary.Lines[0].Line == "" {
		t.Error("Expected line to be non-empty")
	}
}

func TestParsingLineLevelFeedbackResponse_Tab(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_response.json"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	summary, err := ParseLineLevelFeedback(string(data))
	if err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if len(summary.Lines) == 0 {
		t.Error("Expected summary to be non-empty")
	}

	if summary.Lines[0].File == "" {
		t.Error("Expected file name to be non-empty")
	}

	if summary.Lines[0].Body == "" {
		t.Error("Expected body to be non-empty")
	}

	if summary.Lines[0].Suggestion == "" {
		t.Error("Expected suggestion to be non-empty")
	}

	if summary.Lines[0].Category == "" {
		t.Error("Expected category to be non-empty")
	}

	if summary.Lines[0].Title == "" {
		t.Error("Expected title to be non-empty")
	}

	if summary.Lines[0].Line == "" {
		t.Error("Expected line to be non-empty")
	}
}
