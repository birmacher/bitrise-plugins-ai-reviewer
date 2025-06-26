package git

import (
	"testing"
)

func TestGetDiffWithMergeBase(t *testing.T) {
	// For this test, we need to handle multiple calls to the mock
	// We'll use a variable to track how many times Run is called
	callCount := 0

	customRunner := &CustomMockRunner{
		RunFunc: func(name string, args ...string) (string, error) {
			callCount++

			if callCount == 1 {
				// First call should be merge-base
				if args[0] != "merge-base" || args[1] != "abc123" || args[2] != "main" {
					t.Errorf("Expected first call args to be ['merge-base', 'abc123', 'main'], got %v", args)
				}
				return "mergebase123", nil
			} else {
				// Second call should be diff
				if args[0] != "diff" || args[1] != "mergebase123..abc123" {
					t.Errorf("Expected second call args to be ['diff', 'mergebase123..abc123'], got %v", args)
				}
				return "mock diff output", nil
			}
		},
	}

	client := NewClient(customRunner)
	diff, err := client.GetDiffWithMergeBase("abc123", "main")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if diff != "mock diff output" {
		t.Errorf("Expected 'mock diff output', got %s", diff)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls to Run, got %d", callCount)
	}
}

// CustomMockRunner is another mock for testing multiple calls
type CustomMockRunner struct {
	RunFunc func(name string, args ...string) (string, error)
}

func (r *CustomMockRunner) Run(name string, args ...string) (string, error) {
	return r.RunFunc(name, args...)
}
