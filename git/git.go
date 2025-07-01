package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	// DefaultRenameThreshold is the default threshold for detecting file renames
	DefaultRenameThreshold = "90%"
	// DefaultDiffAlgorithm is the default algorithm for computing diffs
	DefaultDiffAlgorithm = "minimal"
)

// Runner defines an interface for running git commands
type Runner interface {
	Run(name string, args ...string) (string, error)
}

// Ensure DefaultRunner implements Runner interface
var _ Runner = (*DefaultRunner)(nil)

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

func (c *Client) GetDiff(commitHash, targetBranch string) (string, error) {
	fmt.Println("")
	fmt.Println("Generating diff...")

	commitHash, err := c.GetCommitHash(commitHash)
	if err != nil {
		return "", fmt.Errorf("error getting commit hash: %w", err)
	}
	fmt.Println("Using commit hash:", commitHash)

	if targetBranch != "" {
		fmt.Println("Using target branch for merge-base:", targetBranch)
		return c.GetDiffWithMergeBase(commitHash, targetBranch, false)
	}

	return c.GetDiffWithParent(commitHash, false)
}

// GetFileContents retrieves the content of all files changed in the specified commit.
// If targetBranch is provided, it compares against the merge-base with that branch.
// Returns a formatted string containing the content of all changed files.
func (c *Client) GetFileContents(commitHash, targetBranch string) (string, error) {
	fmt.Println("")
	fmt.Println("Generating file contents...")

	commitHash, err := c.GetCommitHash(commitHash)
	if err != nil {
		return "", fmt.Errorf("error getting commit hash: %w", err)
	}
	fmt.Println("Using commit hash:", commitHash)

	files, err := c.getChangedFilesForCommit(commitHash, targetBranch)
	if err != nil {
		return "", fmt.Errorf("error getting changed files: %w", err)
	}

	fileOutput := []string{}
	for _, filePath := range files {
		fmt.Println(" -> Processing file:", filePath)
		output, err := c.getFileContent(commitHash, filePath)
		if err != nil {
			return "", err
		}
		if output == "" {
			fmt.Println(" [!] File not found or empty:", filePath)
			continue
		}
		fileOutput = append(fileOutput, fmt.Sprintf("===== FILE: %s =====\n%s\n===== END =====\n\n", filePath, output))
	}

	return strings.Join(fileOutput, "\n\n"), nil
}

// GetBlameForFileLine retrieves the commit hash that last modified the specified line in a file.
// Returns the commit hash responsible for the given line.
func (c *Client) GetBlameForFileLine(commitHash string, filePath string, lineNumber int) (string, error) {
	if commitHash == "" || filePath == "" || lineNumber <= 0 {
		return "", errors.New("commit hash, file path and line number cannot be empty")
	}

	output, err := c.runner.Run("git", "blame", "-L", fmt.Sprintf("%d,%d", lineNumber, lineNumber), commitHash, "--", filePath)
	if err != nil {
		return "", fmt.Errorf("error getting blame for file line: %w", err)
	}

	parts := strings.Split(output, " ")
	if len(parts) < 2 {
		return "", errors.New("invalid blame output")
	}
	return parts[0], nil
}

// GetCommitHash returns the provided commit hash or the current commit hash if none is provided.
func (c *Client) GetCommitHash(commitHash string) (string, error) {
	if commitHash == "" {
		fmt.Println("No commit hash provided, fetching current commit hash...")
		ch, err := c.GetCurrentCommitHash()

		if err != nil {
			return "", err
		}

		commitHash = ch
	}

	return commitHash, nil
}

func (c *Client) getDiff(commitRange string, fileOnly bool) (string, error) {
	params := []string{
		"diff",
		"--no-color",
		"--no-ext-diff",
		"--diff-algorithm=" + DefaultDiffAlgorithm,
		"--find-renames=" + DefaultRenameThreshold,
		"-U0",
		commitRange,
	}
	if fileOnly {
		params = append(params, "--name-only")
	}
	return c.runner.Run("git", params...)
}

// GetDiffWithParent returns the diff between the current commit and its parent
func (c *Client) GetDiffWithParent(commitHash string, fileOnly bool) (string, error) {
	if commitHash == "" {
		return "", errors.New("commit hash cannot be empty")
	}

	return c.getDiff(fmt.Sprintf("%s^..%s", commitHash, commitHash), fileOnly)
}

// GetDiffWithMergeBase returns the diff between the current commit and the merge base with the provided branch
func (c *Client) GetDiffWithMergeBase(commitHash, branchName string, fileOnly bool) (string, error) {
	if commitHash == "" || branchName == "" {
		return "", errors.New("commit hash and branch name cannot be empty")
	}

	// Find the merge base
	mergeBase, err := c.runner.Run("git", "merge-base", commitHash, branchName)
	if err != nil {
		return "", fmt.Errorf("error finding merge base: %w", err)
	}

	return c.getDiff(fmt.Sprintf("%s..%s", mergeBase, commitHash), fileOnly)
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

	output, err := c.getDiff(fmt.Sprintf("%s..%s", from, to), true)
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

func (c *Client) getFileContent(commitHash, filePath string) (string, error) {
	if commitHash == "" || filePath == "" {
		return "", errors.New("commit hash and file path cannot be empty")
	}

	// check if the file exists in the commit
	output, err := c.runner.Run("git", "show", fmt.Sprintf("%s:%s", commitHash, filePath))
	if err != nil {
		return "", nil
	}

	return output, nil
}

// GetChangedFilesForCommit returns a list of files changed in the given commit or compared to the merge base
func (c *Client) getChangedFilesForCommit(commitHash, targetBranch string) ([]string, error) {
	var diff string
	var err error

	if targetBranch != "" {
		fmt.Println("Using target branch for merge-base comparison:", targetBranch)
		diff, err = c.GetDiffWithMergeBase(commitHash, targetBranch, true)
	} else {
		diff, err = c.GetDiffWithParent(commitHash, true)
	}

	if err != nil {
		return []string{}, fmt.Errorf("error getting changed files: %w", err)
	}

	return strings.Split(diff, "\n"), nil
}
