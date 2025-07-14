package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"
	"github.com/sashabaranov/go-openai"
)

// Custom type for context keys to avoid string collisions
type contextKey string

const (
	toolCallDepthKey contextKey = "toolCallDepth"
	messagesKey      contextKey = "messages"
	maxToolCallDepth int        = 5
	ToolUseRequired  string     = "required"
	ToolUseDAuto     string     = "auto"
	ToolUseDisabled  string     = "none"
)

// OpenAIModel implements the LLM interface using OpenAI's API
type OpenAIModel struct {
	client      *openai.Client
	modelName   string
	maxTokens   int
	apiTimeout  int // in seconds
	GitProvider *review.Reviewer
}

// NewOpenAI creates a new OpenAI client
func NewOpenAI(apiKey string, opts ...Option) (*OpenAIModel, error) {
	if apiKey == "" {
		errMsg := "OpenAI API key cannot be empty"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// Create retryable HTTP client with exponential backoff using common configuration
	retryClient := common.NewRetryableClient(common.DefaultRetryConfig())

	// Use the retryable client for OpenAI
	config := openai.DefaultConfig(apiKey)
	config.HTTPClient = retryClient.StandardClient()

	model := &OpenAIModel{
		client:     openai.NewClientWithConfig(config),
		modelName:  "gpt-4.1", // Default model
		maxTokens:  4000,      // Default max tokens
		apiTimeout: 30,        // Default timeout in seconds
	}

	logger.Debugf("OpenAI client initialized with model: %s, max tokens: %d, timeout: %d seconds",
		model.modelName, model.maxTokens, model.apiTimeout)

	// Apply options
	for _, opt := range opts {
		switch opt.Type {
		case ModelNameOption:
			if modelName, ok := opt.Value.(string); ok {
				model.modelName = modelName
			}
		case MaxTokensOption:
			if maxTokens, ok := opt.Value.(int); ok {
				model.maxTokens = maxTokens
			}
		case APITimeoutOption:
			if timeout, ok := opt.Value.(int); ok {
				model.apiTimeout = timeout
			}
		case ToolOption:
			if tools, ok := opt.Value.(Tools); ok {
				model.GitProvider = tools.GitProvider
				if model.GitProvider != nil {
					logger.Debugf("OpenAI client configured with Git provider: %s", (*model.GitProvider).GetProvider())
				}
			} else {
				errMsg := "tool option must be of type Tools"
				logger.Error(errMsg)
				return nil, errors.New(errMsg)
			}
		}
	}

	return model, nil
}

func (o *OpenAIModel) promptWithContext(ctx context.Context, req Request, toolMessages []openai.ChatCompletionMessage, toolChoice string) Response {
	// Create base messages with system and user prompts
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: req.UserPrompt,
		},
	}

	// Add any tool messages from previous calls to maintain context
	if len(toolMessages) > 0 {
		messages = append(messages, toolMessages...)
	}

	// Create and send the completion request
	chatReq := o.createChatCompletionRequest(messages, toolChoice)

	logger.Infof("Sending request to OpenAI with model %s, max tokens %d, tools enabled: %v",
		o.modelName, o.maxTokens, len(chatReq.Tools) > 0)
	logger.Debug("System prompt: " + req.SystemPrompt)
	logger.Debug("User prompt: " + req.UserPrompt)

	resp, err := o.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return o.handleAPIError(fmt.Sprintf("failed to create chat completion: %v", err), nil)
	}

	if len(resp.Choices) == 0 {
		return o.handleAPIError("OpenAI response contained no choices", nil)
	}

	// Check for tool calls in the response and handle them if present
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		logger.Debug("Tool call detected in response")
		return o.handleToolCalls(ctx, resp, req)
	}

	// Return the standard response
	return Response{
		Content: resp.Choices[0].Message.Content,
	}
}

// Prompt sends a request to OpenAI and returns the response
func (o *OpenAIModel) Prompt(req Request) Response {
	// Create context with timeout and initialize it with empty message history
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
	defer cancel()

	// Initialize with empty message history and depth 1
	ctx = context.WithValue(ctx, messagesKey, []openai.ChatCompletionMessage{})
	ctx = context.WithValue(ctx, toolCallDepthKey, 1)

	return o.promptWithContext(ctx, req, nil, ToolUseRequired)
}

// handleToolCalls processes any tool calls in the response and sends follow-up requests if needed
func (o *OpenAIModel) handleToolCalls(ctx context.Context, resp openai.ChatCompletionResponse, originalReq Request) Response {
	// Get current recursion depth from context or start at 1
	depth, ok := ctx.Value(toolCallDepthKey).(int)
	if !ok {
		depth = 1
	}

	// Get existing message history from context if available
	var existingMessages []openai.ChatCompletionMessage
	if ctxMessages, ok := ctx.Value(messagesKey).([]openai.ChatCompletionMessage); ok && len(ctxMessages) > 0 {
		existingMessages = ctxMessages
		logger.Debugf("Retrieved %d existing messages from context", len(existingMessages))
	}

	// Log and process all tool calls
	toolCalls := resp.Choices[0].Message.ToolCalls

	// First, collect all new messages from this tool call sequence
	newMessages := []openai.ChatCompletionMessage{
		{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   resp.Choices[0].Message.Content,
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		},
	}

	// Process each tool call and add the results
	for _, tool := range toolCalls {
		var result string
		var err error

		// Dispatch to appropriate tool handler
		switch tool.Function.Name {
		case "list_directory":
			result, err = o.processListDirToolCall(tool.Function.Arguments)
		case "get_git_diff":
			result, err = o.processGitDiffToolCall(tool.Function.Arguments)
		case "read_file":
			result, err = o.processReadFileToolCall(tool.Function.Arguments)
		case "search_codebase":
			result, err = o.processSearchCodebaseToolCall(tool.Function.Arguments)
		case "get_git_blame":
			result, err = o.processGitBlameToolCall(tool.Function.Arguments)
		case "get_pull_request_details":
			result, err = o.processGetPullRequestDetailsToolCall(tool.Function.Arguments)
		default:
			err = fmt.Errorf("unknown tool: %s", tool.Function.Name)
		}

		// Add the tool response message
		newMessages = append(newMessages, createToolResponse(tool.ID, result, err))
	}

	// Combine existing messages with new ones to maintain full conversation history
	allMessages := append(existingMessages, newMessages...)

	// Create and send follow-up request with tool results
	toolChoice := ToolUseDAuto
	if depth > maxToolCallDepth {
		logger.Warn("Maximum tool call recursion depth reached, stopping further tool calls")
		toolChoice = ToolUseDisabled
	}

	// Create new context with incremented depth and message history
	newCtx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
	defer cancel()
	newCtx = context.WithValue(newCtx, toolCallDepthKey, depth+1)
	newCtx = context.WithValue(newCtx, messagesKey, allMessages)

	// Log conversation state for debugging
	logger.Debugf("Sending next prompt with %d total accumulated messages at depth %d", len(allMessages), depth+1)

	// Get response from the next recursive prompt with full conversation history
	nextResponse := o.promptWithContext(newCtx, originalReq, allMessages, toolChoice)

	// If there was an error in the next call, return it directly
	if nextResponse.Error != nil {
		return nextResponse
	}

	// Return the combined response including all tool calls and history
	return Response{
		Content:   nextResponse.Content,
		ToolCalls: resp.Choices[0].Message.ToolCalls,
	}
}

// getTools returns the list of available tools
func (o *OpenAIModel) getTools() []openai.Tool {
	// List directory
	ListDirTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "list_directory",
			Description: "Lists all the files inside the git repository",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "Optional git reference (branch, tag, or commit hash) to list the directory from. If not provided, reads from the current working directory.",
					},
				},
				"required": []string{},
				"examples": []map[string]interface{}{
					{},
					{
						"ref": "HEAD",
					},
				},
			},
		},
	}

	// Define the git diff tool
	gitDiffTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_git_diff",
			Description: "Gets the diff between two git references (commits, branches, or tags) showing code changes",
			Parameters: map[string]interface{}{
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
		},
	}

	// Define the file reading tool
	readFileTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "read_file",
			Description: "Reads the content of a file from the repository or filesystem",
			Parameters: map[string]interface{}{
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
						"description": "Optional starting line number (1-indexed). If provided with endLine, only returns the specified range of lines.",
					},
					"endLine": map[string]interface{}{
						"type":        "integer",
						"description": "Optional ending line number (1-indexed, inclusive). Must be used with startLine.",
					},
				},
				"required": []string{"path"},
				"examples": []map[string]interface{}{
					{
						"path": "main.go",
					},
					{
						"path":      "cmd/root.go",
						"startLine": 10,
						"endLine":   20,
					},
					{
						"path": "llm/openai.go",
						"ref":  "main",
					},
				},
			},
		},
	}

	searchCodebaseTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "search_codebase",
			Description: "Searches for a string or regex pattern in the codebase. Returns file paths and line numbers where the pattern matches.",
			Parameters: map[string]interface{}{
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
		},
	}

	// Define the git blame tool
	gitBlameTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_git_blame",
			Description: "Gets git blame information for a file or specific lines in a file, showing which commits last modified each line",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The relative path to the file within the repository",
					},
					"startLine": map[string]interface{}{
						"type":        "integer",
						"description": "Optional starting line number (1-indexed). If provided with endLine, only returns blame for the specified range of lines.",
					},
					"endLine": map[string]interface{}{
						"type":        "integer",
						"description": "Optional ending line number (1-indexed, inclusive). Must be used with startLine.",
					},
					"ref": map[string]interface{}{
						"type":        "string",
						"description": "Optional git reference (branch, tag, or commit hash) to get blame for. If not provided, uses HEAD.",
					},
				},
				"required": []string{"path"},
				"examples": []map[string]interface{}{
					{
						"path": "main.go",
					},
					{
						"path":      "cmd/root.go",
						"startLine": 10,
						"endLine":   20,
					},
				},
			},
		},
	}

	getPullRequestDetailsTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_pull_request_details",
			Description: "Retrieves details about a pull request, including its title, description, author, and status",
			Parameters: map[string]interface{}{
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
		},
	}

	return []openai.Tool{ListDirTool, gitDiffTool, readFileTool, searchCodebaseTool, gitBlameTool, getPullRequestDetailsTool}
}

func (o *OpenAIModel) processListDirToolCall(argumentsJSON string) (string, error) {
	logger.Debug("Processing list directory tool call")

	// Parse the arguments JSON
	var args struct {
		Ref string `json:"ref,omitempty"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	if args.Ref == "" {
		args.Ref = "HEAD"
	}

	logger.Infof("🤖 Listing git directory contents at ref: %s", args.Ref)

	git := git.NewClient(git.NewDefaultRunner("."))
	output, err := git.ListFiles(args.Ref)

	if err != nil {
		return "", fmt.Errorf("git ls-tree command failed: %v", err)
	}

	if output == "" {
		return "No tracked files found.", nil
	}

	return output, nil
}

// processGitDiffToolCall extracts parameters and executes the git diff command
func (o *OpenAIModel) processGitDiffToolCall(argumentsJSON string) (string, error) {
	logger.Debug("Processing git diff tool call")

	// Parse the arguments JSON
	var args struct {
		Target string `json:"target"`
		Source string `json:"source"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	// Validate required fields
	if args.Target == "" || args.Source == "" {
		return "", fmt.Errorf("both source and target must be provided")
	}

	logger.Infof("🤖 Getting git diff between `%s` and `%s`", args.Source, args.Target)

	git := git.NewClient(git.NewDefaultRunner("."))
	output, err := git.GetDiff(args.Source, args.Target)

	if err != nil {
		return "", fmt.Errorf("git diff command failed: %v", err)
	}

	if output == "" {
		return "No changes found between the specified references.", nil
	}

	return output, nil
}

// processReadFileToolCall extracts parameters and reads the specified file
func (o *OpenAIModel) processReadFileToolCall(argumentsJSON string) (string, error) {
	logger.Debug("Processing read file tool call")

	// Parse the arguments JSON
	var args struct {
		Path      string `json:"path"`
		Ref       string `json:"ref"`
		StartLine int    `json:"startLine"`
		EndLine   int    `json:"endLine"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	// Validate required fields
	if args.Path == "" {
		return "", fmt.Errorf("file path must be provided")
	}

	// Sanitize the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(args.Path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "/") {
		return "", fmt.Errorf("invalid path: %s", args.Path)
	}

	// Get file content either from git or filesystem
	var content string
	var err error

	if args.Ref == "" {
		args.Ref = "HEAD"
	}

	logger.Infof("🤖 Reading file: `%s`, lines %d to %d, commitHash: %s", args.Path, args.StartLine, args.EndLine, args.Ref)

	git := git.NewClient(git.NewDefaultRunner("."))
	content, err = git.GetFileContent(args.Ref, cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file content from git: %v", err)
	}

	if args.StartLine > 0 && args.EndLine >= args.StartLine {
		lines := strings.Split(content, "\n")
		totalLines := len(lines)

		// Convert to 0-based index
		startIdx := args.StartLine - 1
		endIdx := args.EndLine - 1

		// Validate range
		if startIdx >= totalLines {
			return "", fmt.Errorf("start line %d exceeds file length of %d lines", args.StartLine, totalLines)
		}

		// Adjust end line if it exceeds file length
		if endIdx >= totalLines {
			endIdx = totalLines - 1
			logger.Debugf("Adjusting end line to file length: %d", endIdx+1)
		}

		// Extract specified lines
		content = strings.Join(lines[startIdx:endIdx+1], "\n")
		logger.Debugf("Extracted lines %d-%d (%d lines) from file", args.StartLine, endIdx+1, endIdx-startIdx+1)
	} else {
		logger.Debugf("Reading entire file content (%d characters)", len(content))
	}

	return content, nil
}

func (o *OpenAIModel) processSearchCodebaseToolCall(argumentsJSON string) (string, error) {
	logger.Debug("Processing search codebase tool call")

	// Parse the arguments JSON
	var args struct {
		Query    string `json:"query"`
		Ref      string `json:"ref"`
		UseRegex bool   `json:"use_regex"`
		Path     string `json:"path"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	// Validate required fields
	if args.Query == "" {
		return "", fmt.Errorf("search query must be provided")
	}

	if args.Ref == "" {
		args.Ref = "HEAD"
	}

	logger.Infof("🤖 Searching codebase for query: `%s`, ref: %s, use regex: %t, path: %s", args.Query, args.Ref, args.Path)

	git := git.NewClient(git.NewDefaultRunner("."))
	content, err := git.Grep(args.Ref, args.Query, args.UseRegex, args.Path)
	if err != nil {
		return "", fmt.Errorf("git grep command failed: %v", err)
	}
	return content, nil
}

// processGitBlameToolCall extracts parameters and executes the git blame command
func (o *OpenAIModel) processGitBlameToolCall(argumentsJSON string) (string, error) {
	logger.Debug("Processing git blame tool call")

	// Parse the arguments JSON
	var args struct {
		Path      string `json:"path"`
		Ref       string `json:"ref"`
		StartLine int    `json:"startLine"`
		EndLine   int    `json:"endLine"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	// Validate required fields
	if args.Path == "" {
		return "", fmt.Errorf("file path must be provided")
	}

	// Sanitize the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(args.Path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "/") {
		return "", fmt.Errorf("invalid path: %s", args.Path)
	}

	logger.Infof("🤖 Getting git blame for file: `%s`, lines %d to %d, ref: %s",
		args.Path, args.StartLine, args.EndLine, args.Ref)

	git := git.NewClient(git.NewDefaultRunner("."))
	output, err := git.GetBlame(args.Ref, cleanPath, args.StartLine, args.EndLine)

	if err != nil {
		return "", fmt.Errorf("git blame command failed: %v", err)
	}

	return output, nil
}

func (o *OpenAIModel) processGetPullRequestDetailsToolCall(argumentsJSON string) (string, error) {
	logger.Debug("Processing get pull request details tool call")
	// Parse the arguments JSON
	var args struct {
		RepoOwner string `json:"repo_owner"`
		RepoName  string `json:"repo_name"`
		PRNumber  int    `json:"pr_number"`
	}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}
	// Validate required fields
	if args.RepoOwner == "" || args.RepoName == "" || args.PRNumber <= 0 {
		return "", fmt.Errorf("repo_owner, repo_name, and pr_number must be provided")
	}

	logger.Infof("🤖 Getting pull request details for %s/%s PR #%d", args.RepoOwner, args.RepoName, args.PRNumber)

	// Check if GitProvider is properly initialized
	if o.GitProvider == nil {
		return "", fmt.Errorf("git provider is not initialized, cannot fetch PR details")
	}

	// Create a new GitHub client
	details, err := (*o.GitProvider).GetPullRequestDetails(args.RepoOwner, args.RepoName, args.PRNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR details: %v", err)
	}

	logger.Debug(details.String())

	return details.String(), nil
}

// createChatCompletionRequest creates a standard chat completion request with common settings
func (o *OpenAIModel) createChatCompletionRequest(messages []openai.ChatCompletionMessage, toolChoice string) openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2,
		Tools:       o.getTools(),
		ToolChoice:  toolChoice,
	}
}

// handleAPIError creates a standard error response
func (o *OpenAIModel) handleAPIError(errMsg string, toolCalls interface{}) Response {
	logger.Error(errMsg)
	return Response{
		Error:     errors.New(errMsg),
		ToolCalls: toolCalls,
	}
}

// createToolResponse creates a message with the tool response, handling any errors
func createToolResponse(toolID string, content string, err error) openai.ChatCompletionMessage {
	if err != nil {
		// The error message is already logged in the tool-specific handler
		content = fmt.Sprintf("Error: %v", err)
	}

	return openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    content,
		ToolCallID: toolID,
	}
}
