package llm

import (
	"slices"
	"strings"
)

const (
	ToolTypeInitalizer = "initializer"
	ToolTypeHelper     = "helper"
	ToolTypeFinalizer  = "finalizer"
	ToolTypeDisabled   = "disabled"
)

type EnabledTools struct {
	ListDirectory         string
	GetGitDiff            string
	ReadFile              string
	SearchCodebase        string
	GetGitBlame           string
	GetPullRequestDetails string
	GetBuildLog           string
	PostSummary           string
	PostLineFeedback      string
	PostBuildSummary      string
}

func (et EnabledTools) UseString() string {
	enabled := []string{ToolTypeInitalizer, ToolTypeHelper, ToolTypeFinalizer}
	tools := []string{}
	if slices.Contains(enabled, et.ListDirectory) {
		tools = append(tools, "- list_directory: Use to understand the project structure or locate files.")
	}
	if slices.Contains(enabled, et.GetGitDiff) {
		tools = append(tools, "- get_git_diff: See what changed between branches or commits.")
	}
	if slices.Contains(enabled, et.ReadFile) {
		tools = append(tools, "- read_file: Use to read any file if additional context is needed.")
	}
	if slices.Contains(enabled, et.SearchCodebase) {
		tools = append(tools, "- search_codebase: Use if a function, class, or symbol appears in the diff and you want to know where else it is used or defined.")
	}
	if slices.Contains(enabled, et.GetGitBlame) {
		tools = append(tools, "- get_git_blame: Use to see who last modified a line or to understand why a change was made.")
	}
	if slices.Contains(enabled, et.GetPullRequestDetails) {
		tools = append(tools, "- get_pull_request_details: Use to get details about a pull request. Such as title, description, and author.")
	}
	if slices.Contains(enabled, et.GetBuildLog) {
		tools = append(tools, "- get_build_log: Use to get the build logs and understand the context and the error.")
	}
	if slices.Contains(enabled, et.PostSummary) {
		tools = append(tools, "- post_summary: Use to post a summary of the changes made.")
	}
	if slices.Contains(enabled, et.PostLineFeedback) {
		tools = append(tools, "- post_line_feedback: Use to provide feedback on specific lines of code.")
	}
	if slices.Contains(enabled, et.PostBuildSummary) {
		tools = append(tools, "- post_build_summary: Use to post the summary of the CI build errors and optional suggestion to fix the issue.")
	}

	return strings.Join(tools, "\n")
}

func (et EnabledTools) IsFinal(tool string) bool {
	switch tool {
	case "list_directory":
		return et.ListDirectory == ToolTypeFinalizer
	case "get_git_diff":
		return et.GetGitDiff == ToolTypeFinalizer
	case "read_file":
		return et.ReadFile == ToolTypeFinalizer
	case "search_codebase":
		return et.SearchCodebase == ToolTypeFinalizer
	case "get_git_blame":
		return et.GetGitBlame == ToolTypeFinalizer
	case "get_pull_request_details":
		return et.GetPullRequestDetails == ToolTypeFinalizer
	case "get_build_log":
		return et.GetBuildLog == ToolTypeFinalizer
	case "post_summary":
		return et.PostSummary == ToolTypeFinalizer
	case "post_line_feedback":
		return et.PostLineFeedback == ToolTypeFinalizer
	case "post_build_summary":
		return et.PostBuildSummary == ToolTypeFinalizer
	default:
		return false
	}
}
