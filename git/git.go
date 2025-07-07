package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
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
	logger.Debugf("Creating new Git runner with repo path: %s", repoPath)
	return &DefaultRunner{
		RepoPath: repoPath,
	}
}

// Run executes a git command and returns its output
func (r *DefaultRunner) Run(name string, args ...string) (string, error) {
	logger.Debugf("Running git command: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	if r.RepoPath != "" {
		cmd.Dir = r.RepoPath
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := fmt.Sprintf("error running command: %s\nstderr: %s", err, stderr.String())
		logger.Errorf("Git command failed: %s", errMsg)
		return "", errors.New(errMsg)
	}

	result := strings.TrimSpace(stdout.String())
	return result, nil
}

// Client provides Git operations for AI code review
type Client struct {
	runner Runner
}

// NewClient creates a new Git client
func NewClient(runner Runner) *Client {
	logger.Debug("Creating new Git client")
	return &Client{
		runner: runner,
	}
}

func (c *Client) GetDiff(commitHash, targetBranch string) (string, error) {
	logger.Info("Using commit hash:", commitHash)

	if targetBranch != "" {
		logger.Info("Using target branch for merge-base:", targetBranch)
		return c.GetDiffWithMergeBase(commitHash, targetBranch, false)
	}

	return c.GetDiffWithParent(commitHash, false)
}

// GetFileContents retrieves the content of all files changed in the specified commit.
// If targetBranch is provided, it compares against the merge-base with that branch.
// Returns a formatted string containing the content of all changed files.
func (c *Client) GetFileContents(commitHash, targetBranch string) (string, error) {
	logger.Info("Generating file contents...")

	resolvedCommitHash, err := c.GetCommitHash(commitHash)
	if err != nil {
		return "", fmt.Errorf("error getting commit hash: %w", err)
	}
	logger.Info("Using commit hash:", resolvedCommitHash)

	files, err := c.getChangedFilesForCommit(resolvedCommitHash, targetBranch)
	if err != nil {
		return "", fmt.Errorf("error getting changed files: %w", err)
	}

	if len(files) == 0 {
		logger.Info("No files changed in commit")
		return "", nil
	}

	var fileOutput strings.Builder
	for i, filePath := range files {
		if filePath == "" {
			continue // Skip empty file paths
		}

		logger.Debug("Processing file:", filePath)
		content, err := c.getFileContent(resolvedCommitHash, filePath)
		if err != nil {
			return "", fmt.Errorf("error getting file content for %s: %w", filePath, err)
		}

		if content == "" {
			logger.Warn("File not found or empty:", filePath)
			continue
		}

		if i > 0 {
			fileOutput.WriteString("\n\n")
		}

		fileOutput.WriteString(fmt.Sprintf("===== FILE: %s =====\n%s\n===== END =====\n", filePath, content))
	}

	return fileOutput.String(), nil
}

// GetBlameForFileLine retrieves the commit hash that last modified the specified line in a file.
// Returns the commit hash responsible for the given line.
func (c *Client) GetBlameForFileLine(commitHash string, filePath string, lineNumber int) (string, error) {
	if commitHash == "" || filePath == "" || lineNumber <= 0 {
		errMsg := "commit hash, file path and line number cannot be empty"
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}

	output, err := c.runner.Run("git", "blame", "-L", fmt.Sprintf("%d,%d", lineNumber, lineNumber), commitHash, "--", filePath)
	if err != nil {
		errMsg := fmt.Sprintf("error getting blame for file line: %v", err)
		logger.Errorf(errMsg)
		return "", errors.New(errMsg)
	}

	parts := strings.Split(output, " ")
	if len(parts) < 2 {
		errMsg := "invalid blame output, expected at least 2 parts"
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}
	return parts[0], nil
}

// GetCommitHash returns the provided commit hash or the current commit hash if none is provided.
func (c *Client) GetCommitHash(commitHash string) (string, error) {
	if commitHash == "" {
		logger.Debug("No commit hash provided, fetching current commit hash...")
		return c.GetCurrentCommitHash()
	}

	// Validate that the commit hash exists and is valid
	resolvedHash, err := c.runner.Run("git", "rev-parse", "--verify", commitHash+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("invalid commit hash %s: %w", commitHash, err)
	}

	return resolvedHash, nil
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
		return "", fmt.Errorf("error finding merge base between %s and %s: %w", commitHash, branchName, err)
	}

	if strings.TrimSpace(mergeBase) == "" {
		return "", errors.New("empty merge base returned")
	}

	return c.getDiff(fmt.Sprintf("%s..%s", strings.TrimSpace(mergeBase), commitHash), fileOnly)
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
		return nil, fmt.Errorf("error getting changed files between %s and %s: %w", from, to, err)
	}

	if strings.TrimSpace(output) == "" {
		return []string{}, nil
	}

	// Filter out empty lines
	files := strings.Split(output, "\n")
	filteredFiles := make([]string, 0, len(files))
	for _, file := range files {
		if trimmed := strings.TrimSpace(file); trimmed != "" {
			filteredFiles = append(filteredFiles, trimmed)
		}
	}

	return filteredFiles, nil
}

func (c *Client) getFileContent(commitHash, filePath string) (string, error) {
	if commitHash == "" || filePath == "" {
		return "", errors.New("commit hash and file path cannot be empty")
	}

	// Check if the file exists in the commit
	output, err := c.runner.Run("git", "show", fmt.Sprintf("%s:%s", commitHash, filePath))
	if err != nil {
		// File might not exist in this commit, which is valid for deleted files
		logger.Debugf("File %s not found in commit %s: %v", filePath, commitHash, err)
		return "", nil
	}

	return output, nil
}

// GetChangedFilesForCommit returns a list of files changed in the given commit or compared to the merge base
func (c *Client) getChangedFilesForCommit(commitHash, targetBranch string) ([]string, error) {
	var diff string
	var err error

	if targetBranch != "" {
		logger.Info("Using target branch for merge-base comparison:", targetBranch)
		diff, err = c.GetDiffWithMergeBase(commitHash, targetBranch, true)
	} else {
		diff, err = c.GetDiffWithParent(commitHash, true)
	}

	if err != nil {
		return nil, fmt.Errorf("error getting changed files: %w", err)
	}

	if strings.TrimSpace(diff) == "" {
		return []string{}, nil
	}

	// Filter out empty lines
	files := strings.Split(diff, "\n")
	filteredFiles := make([]string, 0, len(files))
	for _, file := range files {
		if trimmed := strings.TrimSpace(file); trimmed != "" {
			filteredFiles = append(filteredFiles, trimmed)
		}
	}

	return filteredFiles, nil
}
