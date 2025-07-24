package llm

import (
	"fmt"
	"slices"
	"strings"
)

const (
	ToolTypeInitalizer = "initializer"
	ToolTypeHelper     = "helper"
	ToolTypeFinalizer  = "finalizer"
	ToolTypeDisabled   = "disabled"
)

const (
	ToolTypeListDirectory         = "list_directory"
	ToolTypeGitDiff               = "get_git_diff"
	ToolTypeReadFile              = "read_file"
	ToolTypeSearchCodebase        = "search_codebase"
	ToolTypeGitBlame              = "get_git_blame"
	ToolTypeGetPullRequestDetails = "get_pull_request_details"
	ToolTypeGetCIBuildLog         = "get_ci_build_log"
	ToolTypePostPRSummary         = "post_summary"
	ToolTypePostLineFeedback      = "post_line_feedback"
	ToolTypePostCISummary         = "post_build_summary"
)

type EnabledTools struct {
	ListDirectory         string
	GitDiff               string
	ReadFile              string
	SearchCodebase        string
	GitBlame              string
	GetPullRequestDetails string
	GetCIBuildLog         string
	PostLineFeedback      string
	PostPRSummary         string
	PostCISummary         string
}

type ToolParam struct {
	Name        string
	Description string
	Properties  map[string]interface{}
}

func (et EnabledTools) UseString(enabledToolTypes []string) string {
	toolStr := []string{}
	for _, tool := range et.ToolParams(enabledToolTypes) {
		toolStr = append(toolStr, fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
	}

	return strings.Join(toolStr, "\n")
}

func (et EnabledTools) ToolParams(enabledToolTypes []string) []ToolParam {
	toolParams := []ToolParam{}
	if slices.Contains(enabledToolTypes, et.ListDirectory) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeListDirectory,
			Description: "Lists recursively all the files inside the git repository",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "Optional git reference (branch, tag, or commit hash) to list the directory from. If not provided, reads from the current working directory.",
					},
				},
				"required": []string{},
				"examples": []map[string]interface{}{
					{
						"ref": "HEAD",
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.GitDiff) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeGitDiff,
			Description: "Gets the diff between two git references (commits, branches, or tags) showing code changes",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source": map[string]interface{}{
						"type":        "string",
						"description": "The source branch or commit with changes (e.g., feature branch, PR commit)",
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "The target branch the changes will be merged into (e.g., 'main', 'develop')",
					},
				},
				"required": []string{"source", "target"},
				"examples": []map[string]interface{}{
					{
						"source": "5d7f7ce9c705d2f6bfcac3ae35f5bbc9ba736b5a",
						"target": "master",
					},
					{
						"source": "feature/branch",
						"target": "master",
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.ReadFile) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeReadFile,
			Description: "Reads the content of a file from the repository or filesystem",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The relative path to the file within the repository",
					},
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "Optional git reference (branch, tag, or commit hash) to read the file from. If not provided, reads from the current working directory.",
					},
					"startLine": map[string]interface{}{
						"type":        "integer",
						"description": "Optional starting line number. If provided with endLine, only returns the specified range of lines.",
					},
					"endLine": map[string]interface{}{
						"type":        "integer",
						"description": "Optional ending line number. Must be used with startLine.",
					},
				},
				"required": []string{"path"},
				"examples": []map[string]interface{}{
					{
						"path": "main.go",
					},
					{
						"path":      "cmd/root.go",
						"startLine": 100,
						"endLine":   200,
					},
					{
						"path": "llm/openai.go",
						"ref":  "main",
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.SearchCodebase) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeSearchCodebase,
			Description: "Searches for a string or regex pattern in the codebase. Returns file paths and line numbers where the pattern matches.",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The string or regex pattern to search for.",
					},
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "Optional git reference (branch, tag, or commit hash) to search in. Defaults to current working directory.",
					},
					"use_regex": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, treats the query as a regex pattern.",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Restrict search to files under this directory path.",
					},
				},
				"required": []string{"query"},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.GitBlame) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeGitBlame,
			Description: "Gets git blame information for a file or specific lines in a file, showing which commits last modified each line",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The relative path to the file within the repository",
					},
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "Optional git reference (branch, tag, or commit hash) to get blame from. If not provided, reads from the current working directory.",
					},
				},
				"required": []string{"path"},
				"examples": []map[string]interface{}{
					{
						"path": "main.go",
					},
					{
						"path": "cmd/root.go",
						"ref":  "main",
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.GetPullRequestDetails) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeGetPullRequestDetails,
			Description: "Retrieves details about a pull request, including its title, description, author, and status.",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_owner": map[string]interface{}{
						"type":        "string",
						"description": "The owner of the repository (e.g., 'bitrise-io')",
					},
					"repo_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the repository (e.g., 'bitrise-plugins-ai-reviewer')",
					},
					"pr_number": map[string]interface{}{
						"type":        "integer",
						"description": "The pull request number to retrieve details for",
					},
				},
				"required": []string{"repo_owner", "repo_name", "pr_number"},
				"examples": []map[string]interface{}{
					{
						"repo_owner": "bitrise-io",
						"repo_name":  "bitrise-plugins-ai-reviewer",
						"pr_number":  42,
					},
					{
						"repo_owner": "bitrise-io",
						"repo_name":  "bitrise-plugins-ai-reviewer",
						"pr_number":  100,
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.GetCIBuildLog) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypeGetCIBuildLog,
			Description: "Retrieves the CI build logs for a specific build",
			Properties: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
				"examples":   []map[string]interface{}{},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.PostPRSummary) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypePostPRSummary,
			Description: "Posts a summary of the changes made in the pull request",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_owner": map[string]interface{}{
						"type":        "string",
						"description": "The owner of the repository (e.g., 'bitrise-io')",
					},
					"repo_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the repository (e.g., 'bitrise-plugins-ai-reviewer')",
					},
					"pr_number": map[string]interface{}{
						"type":        "integer",
						"description": "The pull request number to retrieve details for",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "A high-level, to-the-point, short summary of the overall change instead of specific files within 80 words.",
					},
					"walkthrough": map[string]interface{}{
						"type":        "integer",
						"description": "Files that changed and the change summary separated by ':'. Group files with similar changes together to save space. Separate lines by \n.",
					},
					"haiku": map[string]interface{}{
						"type":        "string",
						"description": "A whimsical, short haiku to celebrate the changes as 'Bit Bot'. Format the haiku as a quote using the '>' symbol and feel free to use emojis where relevant.",
					},
				},
				"required": []string{"repo_owner", "repo_name", "pr_number", "summary", "walkthrough", "haiku"},
				"examples": []map[string]interface{}{
					{
						"repo_owner":  "bitrise-io",
						"repo_name":   "bitrise-plugins-ai-reviewer",
						"pr_number":   42,
						"summary":     "This PR implements a new feature that allows users to filter search results by date.",
						"walkthrough": "main.go: Implemented search filtering by date\ncmd/root.go: Updated CLI commands to support new filter options",
						"haiku":       "> New tools in the breeze\n> Codebase whispers, search, blame, fetch—\n> Review magic grows 🌱🤖",
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.PostLineFeedback) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypePostLineFeedback,
			Description: "Posts feedback on issues found at specific lines of code in a pull request",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_owner": map[string]interface{}{
						"type":        "string",
						"description": "The owner of the repository (e.g., 'bitrise-io')",
					},
					"repo_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the repository (e.g., 'bitrise-plugins-ai-reviewer')",
					},
					"pr_number": map[string]interface{}{
						"type":        "integer",
						"description": "The pull request number to retrieve details for",
					},
					"file": map[string]interface{}{
						"type":        "string",
						"description": "The relative path to the file within the repository",
					},
					"issue": map[string]interface{}{
						"type":        "string",
						"description": "A description of the issue found in the code",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "The category of the feedback from: bug, refactor, improvement, documentation, nitpick, test coverage, security.",
					},
					"line": map[string]interface{}{
						"type":        "string",
						"description": "The exact line from the diff hunk that you are commenting on.",
					},
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "A short, clear instruction for an AI agent to fix the issue (imperative; do not include file or line number).",
					},
					"suggestion": map[string]interface{}{
						"type":        "string",
						"description": "An optional suggestion for how to fix the issue. If provided, it should be a complete code snippet that can be applied directly to the file.",
					},
				},
				"required": []string{"repo_owner", "repo_name", "pr_number", "file", "issue", "category", "line", "prompt"},
				"examples": []map[string]interface{}{
					{
						"repo_owner": "bitrise-io",
						"repo_name":  "bitrise-plugins-ai-reviewer",
						"pr_number":  42,
						"file":       "main.go",
						"issue":      "This line has a potential bug where the variable is not initialized before use.",
						"category":   "bug",
						"line":       "\t\tif x > 0 {",
						"prompt":     "Initialize the variable x before using it to avoid potential runtime errors",
						"suggestion": "\tx := 0 // Initialize x before use\n\t\tif x > 0 {",
					},
				},
			},
		})
	}

	if slices.Contains(enabledToolTypes, et.PostCISummary) {
		toolParams = append(toolParams, ToolParam{
			Name:        ToolTypePostCISummary,
			Description: "Posts a summary of the CI build errors and optional suggestions to fix the issue",
			Properties: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "A summary of the build results, including success/failure status and any relevant details",
					},
					"suggestion": map[string]interface{}{
						"type":        "string",
						"description": "An optional suggestion for how to improve the build process or fix any issues",
					},
				},
				"required": []string{"summary"},
				"examples": []map[string]interface{}{
					{
						"summary":    "Build completed successfully with no errors.",
						"suggestion": "Consider adding more unit tests to improve code coverage.",
					},
				},
			},
		})
	}

	return toolParams
}

func (et EnabledTools) IsFinal(tool string) bool {
	switch tool {
	case "list_directory":
		return et.ListDirectory == ToolTypeFinalizer
	case "get_git_diff":
		return et.GitDiff == ToolTypeFinalizer
	case "read_file":
		return et.ReadFile == ToolTypeFinalizer
	case "search_codebase":
		return et.SearchCodebase == ToolTypeFinalizer
	case "get_git_blame":
		return et.GitBlame == ToolTypeFinalizer
	case "get_pull_request_details":
		return et.GetPullRequestDetails == ToolTypeFinalizer
	case "get_ci_build_log":
		return et.GetCIBuildLog == ToolTypeFinalizer
	case "post_summary":
		return et.PostPRSummary == ToolTypeFinalizer
	case "post_line_feedback":
		return et.PostLineFeedback == ToolTypeFinalizer
	case "post_build_summary":
		return et.PostCISummary == ToolTypeFinalizer
	default:
		return false
	}
}
