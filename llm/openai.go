package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
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
	client     *openai.Client
	modelName  string
	maxTokens  int
	apiTimeout int // in seconds
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
					"base": map[string]interface{}{
						"type":        "string",
						"description": "The base commit or branch to compare from (e.g., 'main', 'HEAD~1', '2a7ebf')",
					},
					"head": map[string]interface{}{
						"type":        "string",
						"description": "The head commit or branch to compare to (e.g., 'feature-branch', 'HEAD', 'main')",
					},
				},
				"required": []string{"base", "head"},
				"examples": []map[string]interface{}{
					{
						"base": "main",
						"head": "feature-branch",
					},
					{
						"base": "HEAD~3",
						"head": "HEAD",
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

	return []openai.Tool{ListDirTool, gitDiffTool, readFileTool}
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

	if args.Ref != "" {
		args.Ref = "HEAD"
	}

	logger.Infof("Listing directory contents at ref: %s", args.Ref)

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
		Base string `json:"base"`
		Head string `json:"head"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	// Validate required fields
	if args.Base == "" || args.Head == "" {
		return "", fmt.Errorf("both base and head must be provided")
	}

	logger.Infof("Getting git diff between `%s` and `%s`", args.Head, args.Base)

	git := git.NewClient(git.NewDefaultRunner("."))
	output, err := git.GetDiff(args.Head, args.Base)

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

	if args.Ref != "" {
		logger.Infof("Reading file: `%s`, lines %d to %d, commitHash: %s", args.Path, args.StartLine, args.EndLine, args.Ref)

		git := git.NewClient(git.NewDefaultRunner("."))
		content, err = git.GetFileContent(args.Ref, cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to get file content from git: %v", err)
		}
	} else {
		logger.Infof("Reading file: `%s`, lines %d to %d", args.Path, args.StartLine, args.EndLine)

		contentBytes, err := os.ReadFile(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %v", err)
		}
		content = string(contentBytes)
	}

	if err != nil {
		return "", fmt.Errorf("failed to read file content: %v", err)
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
