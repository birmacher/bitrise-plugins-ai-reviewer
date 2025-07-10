package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsingResponse(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_response.json"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	summary, err := ParseSummary(string(data))
	if err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if summary.Summary == "" {
		t.Error("Expected summary to be non-empty")
	}
	if len(summary.Walkthrough) == 0 {
		t.Error("Expected walkthrough to contain at least one file change")
	}
	if summary.Haiku == "" {
		t.Error("Expected haiku to be non-empty")
	}
}

func TestParsingResponse_NewLine(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_response_new_line.json"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	summary, err := ParseSummary(string(data))
	if err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if summary.Summary == "" {
		t.Error("Expected summary to be non-empty")
	}
	if len(summary.Walkthrough) == 0 {
		t.Error("Expected walkthrough to contain at least one file change")
	}
	if summary.Haiku == "" {
		t.Error("Expected haiku to be non-empty")
	}
}

func TestParsingResponse_Tab(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_response_tab.json"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	summary, err := ParseSummary(string(data))
	if err != nil {
		t.Fatalf("Failed to parse summary: %v", err)
	}
	if summary.Summary == "" {
		t.Error("Expected summary to be non-empty")
	}
	if len(summary.Walkthrough) == 0 {
		t.Error("Expected walkthrough to contain at least one file change")
	}
	if summary.Haiku == "" {
		t.Error("Expected haiku to be non-empty")
	}
}
