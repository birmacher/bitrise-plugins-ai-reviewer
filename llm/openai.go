package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/ci"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"
	"github.com/sashabaranov/go-openai"
)

// Custom type for context keys to avoid string collisions
type contextKey string

const (
	emptyContentKey  string     = "emptyContent"
	toolCallDepthKey contextKey = "toolCallDepth"
	messagesKey      contextKey = "messages"
	maxToolCallDepth int        = 10
)

// OpenAIModel implements the LLM interface using OpenAI's API
type OpenAIModel struct {
	client       *openai.Client
	modelName    string
	maxTokens    int
	apiTimeout   int // in seconds
	GitProvider  *review.Reviewer
	Settings     *common.Settings
	LineFeedback []common.LineLevel
	EnabledTools EnabledTools
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
		case EnabledToolsOption:
			if enabledTools, ok := opt.Value.(EnabledTools); ok {
				model.EnabledTools = enabledTools
			}
		}
	}

	return model, nil
}

func (o *OpenAIModel) GetEnabledTools() EnabledTools {
	return o.EnabledTools
}

func (o *OpenAIModel) SetGitProvider(gitProvider *review.Reviewer) {
	o.GitProvider = gitProvider
}

func (o *OpenAIModel) SetSettings(settings *common.Settings) {
	o.Settings = settings
}

func (o *OpenAIModel) promptWithContext(ctx context.Context, req Request, toolMessages []openai.ChatCompletionMessage) Response {
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
	depth, ok := ctx.Value(toolCallDepthKey).(int)
	if !ok {
		depth = 1
	}
	if depth == 1 {
		// todo initializer tool calls
	}

	if depth == maxToolCallDepth {
		logger.Warn("Reaching maximum tool call recursion depth, forcing summary")
	}
	if depth > maxToolCallDepth {
		logger.Warn("Maximum tool call recursion depth reached, stopping further tool calls")
		// todo finalizer
	}

	chatReq := o.createChatCompletionRequest(messages, depth)

	logger.Infof("Sending request to OpenAI with model %s, max tokens %d, tools enabled: %v",
		o.modelName, o.maxTokens, len(chatReq.Tools) > 0)

	// Debug log the messages being sent
	for _, message := range messages {
		logger.Debug("[" + message.Role + " prompt]:\n" + message.Content)
	}

	resp, err := o.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return o.handleAPIError(fmt.Sprintf("failed to create chat completion: %v", err), nil)
	}

	if len(resp.Choices) == 0 {
		return o.handleAPIError("OpenAI response contained no choices", nil)
	}

	// Check for tool calls in the response and handle them if present
	if len(resp.Choices[0].Message.ToolCalls) > 0 {
		// todo: if tool called is finalizer, finish the process
		logger.Debug("Tool call detected in response")
		return o.handleToolCalls(ctx, resp, req)
	}

	// Return the standard response
	responseContent := resp.Choices[0].Message.Content
	if responseContent == "" {
		responseContent = emptyContentKey
	}

	return Response{
		Content: responseContent,
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

	o.LineFeedback = []common.LineLevel{}

	return o.promptWithContext(ctx, req, nil)
}

func (o *OpenAIModel) GetLineFeedback() []common.LineLevel {
	return o.LineFeedback
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

	newMessages := []openai.ChatCompletionMessage{
		{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   resp.Choices[0].Message.Content,
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		},
	}

	finalCall := false

	logger.Debugf("Processing %d tool calls at depth %d", len(toolCalls), depth)

	// Process each tool call and add the results
	for _, tool := range toolCalls {
		var result string
		var err error

		// Dispatch to appropriate tool handler
		switch tool.Function.Name {
		case ToolTypeListDirectory:
			result, err = o.processListDirToolCall(tool.Function.Arguments)
		case ToolTypeGitDiff:
			result, err = o.processGitDiffToolCall(tool.Function.Arguments)
		case ToolTypeReadFile:
			result, err = o.processReadFileToolCall(tool.Function.Arguments)
		case ToolTypeSearchCodebase:
			result, err = o.processSearchCodebaseToolCall(tool.Function.Arguments)
		case ToolTypeGitBlame:
			result, err = o.processGitBlameToolCall(tool.Function.Arguments)
		case ToolTypeGetPullRequestDetails:
			result, err = o.processGetPullRequestDetailsToolCall(tool.Function.Arguments)
		case ToolTypePostPRSummary:
			result, err = o.processPostSummaryToolCall(tool.Function.Arguments)
		case ToolTypePostLineFeedback:
			result, err = o.processPostLineFeedbackToolCall(tool.Function.Arguments)
		case ToolTypeGetCIBuildLog:
			result, err = o.processGetBuildLogsToolCall(tool.Function.Arguments)
		case ToolTypePostCISummary:
			result, err = o.processPostBuildSummaryToolCall(tool.Function.Arguments)
		default:
			err = fmt.Errorf("unknown tool: %s", tool.Function.Name)
		}

		// Add the tool response message
		newMessages = append(newMessages, createToolResponse(tool.ID, result, err))

		if o.EnabledTools.IsFinal(tool.Function.Name) {
			logger.Debugf("Final tool call detected: %s", tool.Function.Name)
			finalCall = true
		}
	}

	responseContent := ""
	if !finalCall {
		// Combine existing messages with new ones to maintain full conversation history
		allMessages := append(existingMessages, newMessages...)

		// Create new context with incremented depth and message history
		newCtx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
		defer cancel()
		newCtx = context.WithValue(newCtx, toolCallDepthKey, depth+1)
		newCtx = context.WithValue(newCtx, messagesKey, allMessages)

		// Log conversation state for debugging
		logger.Debugf("Sending next prompt with %d total accumulated messages at depth %d", len(allMessages), depth+1)

		// Get response from the next recursive prompt with full conversation history
		nextResponse := o.promptWithContext(newCtx, originalReq, allMessages)

		// If there was an error in the next call, return it directly
		if nextResponse.Error != nil {
			return nextResponse
		}

		// Return the combined response including all tool calls and history
		responseContent := nextResponse.Content
		if responseContent == "" {
			responseContent = emptyContentKey
		}
	} else {
		responseContent = newMessages[len(newMessages)-1].Content
	}

	return Response{
		Content:   responseContent,
		ToolCalls: resp.Choices[0].Message.ToolCalls,
	}
}

// getTools returns the list of available tools
func (o *OpenAIModel) getTools(depth int) []openai.Tool {
	enabledToolsType := []string{}
	if depth == 1 {
		enabledToolsType = []string{ToolTypeInitalizer}
	}
	if depth >= maxToolCallDepth {
		enabledToolsType = []string{ToolTypeFinalizer}
	}
	if depth > 1 && depth < maxToolCallDepth {
		enabledToolsType = []string{ToolTypeHelper, ToolTypeFinalizer}
	}

	toolParams := o.EnabledTools.ToolParams(enabledToolsType)

	OpenAITools := []openai.Tool{}
	for _, tool := range toolParams {
		OpenAITools = append(OpenAITools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Properties,
			},
		})
	}

	return OpenAITools
}

func (o *OpenAIModel) processListDirToolCall(argumentsJSON string) (string, error) {
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
		return "", fmt.Errorf("failed to list directory contents: %v", err)
	}

	if output == "" {
		return "No tracked files found.", nil
	}

	return output, nil
}

// processGitDiffToolCall extracts parameters and executes the git diff command
func (o *OpenAIModel) processGitDiffToolCall(argumentsJSON string) (string, error) {
	var args struct {
		Target string `json:"target"`
		Source string `json:"source"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	if args.Source == "" {
		args.Source = "HEAD"
	}

	// Validate required fields
	if args.Target == "" {
		return "", fmt.Errorf("both source and target must be provided")
	}

	logger.Infof("🤖 Getting git diff between `%s` and `%s`", args.Source, args.Target)

	git := git.NewClient(git.NewDefaultRunner("."))
	output, err := git.GetDiff(args.Source, args.Target)

	if err != nil {
		return "", fmt.Errorf("git diff command failed: %v", err)
	}

	if output == "" {
		return "No changes found in diff.", nil
	}

	return output, nil
}

// processReadFileToolCall extracts parameters and reads the specified file
func (o *OpenAIModel) processReadFileToolCall(argumentsJSON string) (string, error) {
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

	logger.Infof("🤖 Reading file: `%s`, lines %d to %d, commitHash: %s", args.Path, args.StartLine, args.EndLine, args.Ref)

	// Sanitize the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(args.Path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "/") {
		return "", fmt.Errorf("invalid path: %s", args.Path)
	}

	var content string
	var err error

	if args.Ref == "" {
		args.Ref = "HEAD"
	}

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
	var args struct {
		Query    string `json:"query"`
		Ref      string `json:"ref"`
		UseRegex bool   `json:"use_regex"`
		Path     string `json:"path"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	logger.Infof("🤖 Searching codebase for query: `%s`, ref: %s", args.Query, args.Ref)

	if args.Query == "" {
		return "", fmt.Errorf("search query must be provided")
	}

	if args.Ref == "" {
		args.Ref = "HEAD"
	}

	git := git.NewClient(git.NewDefaultRunner("."))
	content, err := git.Grep(args.Ref, args.Query, args.UseRegex, args.Path)

	if err != nil {
		return "", fmt.Errorf("git grep command failed: %v", err)
	}

	if content == "" {
		return "No matches found for the search query.", nil
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

	logger.Infof("🤖 Getting git blame for file: `%s`, lines %d to %d, ref: %s",
		args.Path, args.StartLine, args.EndLine, args.Ref)

	if args.Path == "" {
		return "", fmt.Errorf("file path must be provided")
	}

	cleanPath := filepath.Clean(args.Path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) && strings.HasPrefix(cleanPath, "/") {
		return "", fmt.Errorf("invalid path: %s", args.Path)
	}

	git := git.NewClient(git.NewDefaultRunner("."))
	output, err := git.GetBlame(args.Ref, cleanPath, args.StartLine, args.EndLine)

	if err != nil {
		return "", fmt.Errorf("git blame command failed: %v", err)
	}

	if output == "" {
		return "No blame information found for the specified file and lines.", nil
	}

	return output, nil
}

func (o *OpenAIModel) processGetPullRequestDetailsToolCall(argumentsJSON string) (string, error) {
	var args struct {
		RepoOwner string `json:"repo_owner"`
		RepoName  string `json:"repo_name"`
		PRNumber  int    `json:"pr_number"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	logger.Infof("🤖 Getting pull request details for %s/%s PR #%d", args.RepoOwner, args.RepoName, args.PRNumber)

	if args.RepoOwner == "" || args.RepoName == "" || args.PRNumber <= 0 {
		return "", fmt.Errorf("repo_owner, repo_name, and pr_number must be provided")
	}

	if o.GitProvider == nil {
		return "", fmt.Errorf("git provider is not initialized, cannot fetch PR details")
	}

	pullRequestDetails, err := (*o.GitProvider).GetPullRequestDetails(args.RepoOwner, args.RepoName, args.PRNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR details: %v", err)
	}

	return pullRequestDetails.String(), nil
}

func (o *OpenAIModel) processPostSummaryToolCall(argumentsJSON string) (string, error) {
	var args struct {
		RepoOwner   string `json:"repo_owner"`
		RepoName    string `json:"repo_name"`
		PRNumber    int    `json:"pr_number"`
		Summary     string `json:"summary"`
		Walkthrough string `json:"walkthrough"`
		Haiku       string `json:"haiku"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	logger.Infof("🤖 Posting summary")

	if args.RepoOwner == "" || args.RepoName == "" || args.PRNumber <= 0 {
		return "", fmt.Errorf("repo_owner, repo_name, and pr_number must be provided")
	}

	if args.Summary == "" {
		return "", fmt.Errorf("summary must be provided")
	}

	if o.GitProvider == nil {
		return "", fmt.Errorf("git provider is not initialized, cannot fetch PR details")
	}

	walkthrough := make([]common.Walkthrough, 0)
	for line := range strings.SplitSeq(args.Walkthrough, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid walkthrough format, expected 'file: change summary', got: %s", line)
		}
		filePath := strings.TrimSpace(parts[0])
		changeSummary := strings.TrimSpace(parts[1])

		walkthrough = append(walkthrough, common.Walkthrough{
			Files:   filePath,
			Summary: changeSummary,
		})
	}

	summary := common.Summary{
		Summary:     args.Summary,
		Walkthrough: walkthrough,
		Haiku:       args.Haiku,
	}

	headerStr := summary.Header()
	summaryStr := summary.String((*o.GitProvider).GetProvider(), *o.Settings)

	err := (*o.GitProvider).PostSummary(args.RepoOwner, args.RepoName, args.PRNumber, headerStr, summaryStr)
	if err != nil {
		return "", fmt.Errorf("failed to post summary: %v", err)
	}

	return "Summary posted successfully", nil
}

func (o *OpenAIModel) processPostLineFeedbackToolCall(argumentsJSON string) (string, error) {
	var args struct {
		RepoOwner  string `json:"repo_owner"`
		RepoName   string `json:"repo_name"`
		PRNumber   int    `json:"pr_number"`
		File       string `json:"file"`
		Issue      string `json:"issue"`
		Category   string `json:"category"`
		Line       string `json:"line"`
		Prompt     string `json:"prompt"`
		Suggestion string `json:"suggestion,omitempty"`
	}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	logger.Infof("🤖 Posting line fedback for %s", args.File)

	if args.RepoOwner == "" || args.RepoName == "" || args.PRNumber <= 0 {
		return "", fmt.Errorf("repo_owner, repo_name, and pr_number must be provided")
	}

	if args.File == "" || args.Line == "" || args.Issue == "" {
		return "", fmt.Errorf("file, line and issue must be provided")
	}

	lineFeedback := common.LineLevel{
		File:       args.File,
		Body:       args.Issue,
		Category:   args.Category,
		Line:       args.Line,
		Prompt:     args.Prompt,
		Suggestion: args.Suggestion,
	}

	o.LineFeedback = append(o.LineFeedback, lineFeedback)

	return fmt.Sprintf("Line feedback processed successfully for file %s", lineFeedback.File), nil
}

func (o *OpenAIModel) processGetBuildLogsToolCall(argumentsJSON string) (string, error) {
	logger.Infof("🤖 Getting build logs")

	logs, err := ci.GetBuildLog()
	if err != nil {
		return "", fmt.Errorf("failed to get build logs: %v", err)
	}

	return logs, nil
}

func (o *OpenAIModel) processPostBuildSummaryToolCall(argumentsJSON string) (string, error) {
	var args struct {
		Build      string `json:"build"`
		Summary    string `json:"summary"`
		Suggestion string `json:"suggestion,omitempty"`
	}

	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %v", err)
	}

	logger.Infof("🤖 Posting build summary for build: %s", args.Build)

	if args.Summary == "" {
		return "", fmt.Errorf("summary must be provided")
	}

	err := ci.PostBuildSummary(args.Summary, args.Suggestion)
	if err != nil {
		return "", fmt.Errorf("failed to post build summary: %v", err)
	}

	return "Build summary posted successfully", nil
}

// createChatCompletionRequest creates a standard chat completion request with common settings
func (o *OpenAIModel) createChatCompletionRequest(messages []openai.ChatCompletionMessage, depth int) openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2,
		Tools:       o.getTools(depth),
		ToolChoice:  "required",
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
		logger.Warnf("Tool call %s failed: %v", toolID, err)
		content = fmt.Sprintf("Error: %v", err)
	}

	if content == "" {
		logger.Warnf("Tool call %s returned empty content, using placeholder", toolID)
		content = emptyContentKey
	}

	return openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    content,
		ToolCallID: toolID,
	}
}
