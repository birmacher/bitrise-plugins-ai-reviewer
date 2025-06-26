package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner defines an interface for running git commands
type Runner interface {
	Run(name string, args ...string) (string, error)
}

// DefaultRunner implements the Runner interface using exec.Command
type DefaultRunner struct {
	RepoPath string
}

// NewDefaultRunner creates a new instance of DefaultRunner
func NewDefaultRunner(repoPath string) *DefaultRunner {
	return &DefaultRunner{
		RepoPath: repoPath,
	}
}

// Run executes a git command and returns its output
func (r *DefaultRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if r.RepoPath != "" {
		cmd.Dir = r.RepoPath
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running command: %s\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Client provides Git operations for AI code review
type Client struct {
	runner Runner
}

// NewClient creates a new Git client
func NewClient(runner Runner) *Client {
	return &Client{
		runner: runner,
	}
}

// GetDiffWithParent returns the diff between the current commit and its parent
func (c *Client) GetDiffWithParent(commitHash string) (string, error) {
	if commitHash == "" {
		return "", errors.New("commit hash cannot be empty")
	}

	// Get the diff between the commit and its parent
	return c.runner.Run("git", "diff", fmt.Sprintf("%s^..%s", commitHash, commitHash))
}

// GetDiffWithMergeBase returns the diff between the current commit and the merge base with the provided branch
func (c *Client) GetDiffWithMergeBase(commitHash, branchName string) (string, error) {
	if commitHash == "" || branchName == "" {
		return "", errors.New("commit hash and branch name cannot be empty")
	}

	// Find the merge base
	mergeBase, err := c.runner.Run("git", "merge-base", commitHash, branchName)
	if err != nil {
		return "", fmt.Errorf("error finding merge base: %w", err)
	}

	// Get the diff between the merge base and the current commit
	return c.runner.Run("git", "diff", fmt.Sprintf("%s..%s", mergeBase, commitHash))
}

// GetCurrentCommitHash returns the hash of the current commit
func (c *Client) GetCurrentCommitHash() (string, error) {
	return c.runner.Run("git", "rev-parse", "HEAD")
}

// GetChangedFiles returns a list of files changed between two commits
func (c *Client) GetChangedFiles(from, to string) ([]string, error) {
	if from == "" || to == "" {
		return nil, errors.New("from and to commits cannot be empty")
	}

	output, err := c.runner.Run("git", "diff", "--name-only", fmt.Sprintf("%s..%s", from, to))
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}
