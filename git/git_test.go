package git

import (
	"testing"
)

// MockRunner is a mock implementation of the Runner interface for testing
type MockRunner struct {
	ReturnOutput string
	ReturnError  error
	CommandRun   string
	ArgsRun      []string
}

// Run implements the Runner interface
func (m *MockRunner) Run(name string, args ...string) (string, error) {
	m.CommandRun = name
	m.ArgsRun = args
	return m.ReturnOutput, m.ReturnError
}

func TestGetDiffWithParent(t *testing.T) {
	mockRunner := &MockRunner{
		ReturnOutput: "mock diff output",
		ReturnError:  nil,
	}
	
	client := NewClient(mockRunner)
	diff, err := client.GetDiffWithParent("abc123")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if diff != "mock diff output" {
		t.Errorf("Expected 'mock diff output', got %s", diff)
	}
	
	if mockRunner.CommandRun != "git" {
		t.Errorf("Expected command 'git', got %s", mockRunner.CommandRun)
	}
	
	if len(mockRunner.ArgsRun) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(mockRunner.ArgsRun))
		return
	}
	
	if mockRunner.ArgsRun[0] != "diff" {
		t.Errorf("Expected first argument to be 'diff', got '%s'", mockRunner.ArgsRun[0])
	}
	
	expectedArg := "abc123^..abc123"
	if mockRunner.ArgsRun[1] != expectedArg {
		t.Errorf("Expected second argument to be '%s', got '%s'", expectedArg, mockRunner.ArgsRun[1])
	}
}
