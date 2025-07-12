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
)

// OpenAIModel implements the LLM interface using OpenAI's API
type OpenAIModel struct {
	client     *openai.Client
	modelName  string
	maxTokens  int
	apiTimeout int // in seconds
	tools      Tools
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

// Prompt sends a request to OpenAI and returns the response
func (o *OpenAIModel) Prompt(req Request) Response {
	logger.Debugf("Sending prompt to OpenAI model: %s", o.modelName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
	defer cancel()

	// Prepare base messages with system and user prompts
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

	// Log what we're sending
	logger.Debug("System prompt: " + req.SystemPrompt)
	logger.Debug("User prompt: " + req.UserPrompt)

	// Create and send the completion request
	chatReq := o.createChatCompletionRequest(messages)
	logger.Infof("Sending request to OpenAI with model %s, max tokens %d, tools enabled: %v",
		o.modelName, o.maxTokens, len(chatReq.Tools) > 0)

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

// handleToolCalls processes any tool calls in the response and sends follow-up requests if needed
func (o *OpenAIModel) handleToolCalls(ctx context.Context, resp openai.ChatCompletionResponse, originalReq Request) Response {
	// Define maximum recursion depth to prevent infinite tool call loops
	const maxToolCallDepth = 5

	// Get current recursion depth from context or start at 1
	depth, ok := ctx.Value(toolCallDepthKey).(int)
	if !ok {
		depth = 1
	}

	// Check if we've exceeded maximum depth
	if depth > maxToolCallDepth {
		logger.Warn("Maximum tool call recursion depth reached, stopping further tool calls")
		return Response{
			Content:   resp.Choices[0].Message.Content + "\n\n[Note: Maximum tool call depth reached]",
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		}
	}

	// Log and process all tool calls
	toolCalls := resp.Choices[0].Message.ToolCalls
	for _, tool := range toolCalls {
		logger.Debugf("Tool call: %s, arguments: %s", tool.Function.Name, tool.Function.Arguments)
	}

	// Create initial messages with original system and user prompts
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: originalReq.SystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: originalReq.UserPrompt,
		},
	}

	// Add the assistant's message with tool calls
	messages = append(messages, openai.ChatCompletionMessage{
		Role:      openai.ChatMessageRoleAssistant,
		Content:   resp.Choices[0].Message.Content,
		ToolCalls: resp.Choices[0].Message.ToolCalls,
	})

	// Process each tool call and add the results
	for _, tool := range toolCalls {
		var result string
		var err error

		// Dispatch to appropriate tool handler
		switch tool.Function.Name {
		case "get_git_diff":
			result, err = o.processGitDiffToolCall(tool.Function.Arguments)
		case "read_file":
			result, err = o.processReadFileToolCall(tool.Function.Arguments)
		// case "get_github_comments":
		// result, err = o.processGitHubCommentsToolCall(tool.Function.Arguments)
		default:
			err = fmt.Errorf("unknown tool: %s", tool.Function.Name)
		}

		// Add the tool response message
		messages = append(messages, createToolResponse(tool.ID, result, err))
	}

	// Create and send follow-up request with tool results
	followUpReq := o.createChatCompletionRequest(messages)
	logger.Debug("Sending follow-up request with tool results")

	followUpResp, err := o.client.CreateChatCompletion(ctx, followUpReq)
	if err != nil {
		return o.handleAPIError(fmt.Sprintf("failed to create follow-up chat completion: %v", err), resp.Choices[0].Message.ToolCalls)
	}

	if len(followUpResp.Choices) == 0 {
		return o.handleAPIError("follow-up OpenAI response contained no choices", resp.Choices[0].Message.ToolCalls)
	}

	// Check for additional tool calls in the follow-up response
	if len(followUpResp.Choices[0].Message.ToolCalls) > 0 {
		toolCount := len(followUpResp.Choices[0].Message.ToolCalls)
		logger.Debugf("Additional %d tool call(s) detected in follow-up response (depth: %d)", toolCount, depth)

		// Create new context with incremented depth for recursion
		newCtx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
		defer cancel()
		newCtx = context.WithValue(newCtx, toolCallDepthKey, depth+1)

		// Create modified request for recursion with context hint
		modifiedReq := originalReq
		modifiedReq.SystemPrompt += "\n\nThis is a continuation of a conversation with previous tool calls."

		// Handle the additional tool calls recursively
		return o.handleToolCalls(newCtx, followUpResp, modifiedReq)
	}

	// Return the final response with original tool calls info preserved
	return Response{
		Content:   followUpResp.Choices[0].Message.Content,
		ToolCalls: resp.Choices[0].Message.ToolCalls,
	}
}

// getTools returns the list of available tools
func (o *OpenAIModel) getTools() []openai.Tool {
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

	// Define GitHub comments tool
	// githubCommentsTool := openai.Tool{
	// 	Type: openai.ToolTypeFunction,
	// 	Function: &openai.FunctionDefinition{
	// 		Name:        "get_github_comments",
	// 		Description: "Gets comments from a GitHub pull request",
	// 		Parameters: map[string]interface{}{
	// 			"type": "object",
	// 			"properties": map[string]interface{}{
	// 				"owner": map[string]interface{}{
	// 					"type":        "string",
	// 					"description": "The GitHub repository owner (user or organization)",
	// 				},
	// 				"repo": map[string]interface{}{
	// 					"type":        "string",
	// 					"description": "The GitHub repository name",
	// 				},
	// 				"pr": map[string]interface{}{
	// 					"type":        "integer",
	// 					"description": "The pull request number",
	// 				},
	// 			},
	// 			"required": []string{"owner", "repo", "pr"},
	// 			"examples": []map[string]interface{}{
	// 				{
	// 					"owner": "birmacher",
	// 					"repo":  "bitrise-plugins-ai-reviewer",
	// 					"pr":    42,
	// 				},
	// 				{
	// 					"owner": "birmacher",
	// 					"repo":  "bitrise-plugins-ai-reviewer",
	// 					"pr":    42,
	// 				},
	// 			},
	// 		},
	// 	},
	// }

	return []openai.Tool{gitDiffTool, readFileTool} //, githubCommentsTool}
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

	logger.Debugf("Executing git diff between %s and %s", args.Base, args.Head)

	// Create the diff range and arguments
	diffRange := fmt.Sprintf("%s..%s", args.Base, args.Head)
	diffArgs := []string{
		"diff",
		"--find-renames=" + git.DefaultRenameThreshold,
		"--diff-algorithm=" + git.DefaultDiffAlgorithm,
		diffRange,
	}

	// Execute the git command using the runner
	output, err := git.NewDefaultRunner(".").Run("git", diffArgs...)
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

	logger.Debugf("Reading file: %s, ref: %s, lines: %d-%d", cleanPath, args.Ref, args.StartLine, args.EndLine)

	// Get file content either from git or filesystem
	var content string
	var err error

	if args.Ref != "" {
		// Read from git ref
		objectPath := fmt.Sprintf("%s:%s", args.Ref, cleanPath)
		content, err = git.NewDefaultRunner(".").Run("git", "show", objectPath)
	} else {
		// Read from filesystem
		contentBytes, err := os.ReadFile(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %v", err)
		}
		content = string(contentBytes)
	}

	if err != nil {
		return "", fmt.Errorf("failed to read file content: %v", err)
	}

	// Extract line range if specified
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

// processGitHubCommentsToolCall extracts parameters and fetches GitHub comments
// func (o *OpenAIModel) processGitHubCommentsToolCall(argumentsJSON string) (string, error) {
// 	logger.Debug("Processing GitHub comments tool call")

// 	// Parse the arguments JSON
// 	var args struct {
// 		Owner        string `json:"owner"`
// 		Repo         string `json:"repo"`
// 		PR           int    `json:"pr"`
// 		FilterHeader string `json:"filterHeader"`
// 	}

// 	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
// 		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
// 	}

// 	// Validate required fields
// 	if args.Owner == "" || args.Repo == "" || args.PR <= 0 {
// 		return "", fmt.Errorf("owner, repo, and PR number must be provided")
// 	}

// 	logger.Debugf("Fetching GitHub comments for %s/%s PR #%d", args.Owner, args.Repo, args.PR)

// 	// Note: This is a simple implementation, in a real-world scenario we would need to
// 	// set up a more advanced implementation that could directly call the GitHub API
// 	// or integrate with the review package without creating import cycles

// 	// This tool provides instructions on how to use the GitHub comments feature
// 	if o.tools.GitProvider == nil {
// 		errMsg := "GitProvider tool is not set, cannot fetch GitHub comments"
// 		logger.Error(errMsg)
// 		return "", errors.New(errMsg)
// 	}

// 	tmpCtx, cancel := o.tools.GitProvider.CreateTimeoutContext()
// 	defer cancel()
// 	o.tools.GitProvider.GetComments(args.Owner, args.Repo, args.PR)

// 	instructions := fmt.Sprintf("To get GitHub comments, use the following code:\n\n"+
// 		"```go\n"+
// 		"import (\n"+
// 		"\t\"context\"\n"+
// 		"\t\"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review\"\n"+
// 		")\n\n"+
// 		"// Create a GitHub reviewer client\n"+
// 		"githubReviewer, err := review.NewGitHub(\n"+
// 		"\treview.WithAPIToken(\"your-github-token\"),\n"+
// 		"\t// Add any other options you need\n"+
// 		")\n"+
// 		"if err != nil {\n"+
// 		"\t// Handle error\n"+
// 		"}\n\n"+
// 		"// Get the GitHub client from the reviewer\n"+
// 		"gh := githubReviewer.(*review.GitHub)\n\n"+
// 		"// Create a context\n"+
// 		"ctx, cancel := gh.CreateTimeoutContext() // Or use context.Background()\n"+
// 		"defer cancel()\n\n"+
// 		"// Get comments for the PR\n"+
// 		"comments, err := gh.getComments(ctx, \"%s\", \"%s\", %d)\n"+
// 		"if err != nil {\n"+
// 		"\t// Handle error\n"+
// 		"}\n\n"+
// 		"// If you want to filter by header\n"+
// 		"if filterHeader := \"%s\"; filterHeader != \"\" {\n"+
// 		"\tcommentID, err := gh.getComment(comments, filterHeader)\n"+
// 		"\tif err != nil {\n"+
// 		"\t\t// Handle error\n"+
// 		"\t}\n"+
// 		"\t// Process the comment with ID: commentID\n"+
// 		"}\n"+
// 		"```\n\n"+
// 		"This code demonstrates how to retrieve comments for the requested PR: %s/%s #%d.",
// 		args.Owner, args.Repo, args.PR, args.FilterHeader, args.Owner, args.Repo, args.PR)

// 	// Return the instructions without error since this is an instruction-only tool
// 	return instructions, nil
// }

// createChatCompletionRequest creates a standard chat completion request with common settings
func (o *OpenAIModel) createChatCompletionRequest(messages []openai.ChatCompletionMessage) openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2, // Lower temperature for more deterministic results
		Tools:       o.getTools(),
		ToolChoice:  "auto", // Let the model decide when to use tools
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
