package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/sashabaranov/go-openai"
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

// Prompt sends a request to OpenAI and returns the response
func (o *OpenAIModel) Prompt(req Request) Response {
	logger.Debugf("Sending prompt to OpenAI model: %s", o.modelName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
	defer cancel()

	logger.Debug("Adding system prompt to OpenAI request")
	logger.Debug(req.SystemPrompt)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
	}

	// Add user prompt
	logger.Debug("Adding user prompt to OpenAI request")
	logger.Debug(req.UserPrompt)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.UserPrompt,
	})

	// Add diff if available
	// if req.Diff != "" {
	// 	logger.Debug("Including diff in OpenAI prompt")
	// 	logger.Debug(req.Diff)
	// 	messages = append(messages, openai.ChatCompletionMessage{
	// 		Role:    openai.ChatMessageRoleUser,
	// 		Content: req.Diff,
	// 	})
	// }

	// // Add file contents if available
	// if req.FileContents != "" {
	// 	logger.Debug("Including file contents in OpenAI prompt")
	// 	logger.Debug(req.FileContents)
	// 	messages = append(messages, openai.ChatCompletionMessage{
	// 		Role:    openai.ChatMessageRoleUser,
	// 		Content: req.FileContents,
	// 	})
	// }

	// // Add line-level feedback if available
	// if req.LineLevelFeedback != "" {
	// 	messages = append(messages, openai.ChatCompletionMessage{
	// 		Role:    openai.ChatMessageRoleAssistant,
	// 		Content: req.LineLevelFeedback,
	// 	})
	// }

	// Define the git diff tool
	gitDiffTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_git_diff",
			Description: "Gets the diff between two commits, branches, or any git references",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"base": map[string]interface{}{
						"type":        "string",
						"description": "The base commit or branch to compare from",
					},
					"head": map[string]interface{}{
						"type":        "string",
						"description": "The head commit or branch to compare to",
					},
				},
				"required": []string{"base", "head"},
			},
		},
	}

	// Create the completion request with the git diff tool
	chatReq := openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2, // Lower temperature for more deterministic results
		Tools:       []openai.Tool{gitDiffTool},
		ToolChoice:  "auto", // Let the model decide when to use tools
	}

	logger.Infof("Sending request to OpenAI with model %s, max tokens %d, tools enabled: %v",
		o.modelName, o.maxTokens, len(chatReq.Tools) > 0)

	resp, err := o.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create chat completion: %v", err)
		logger.Error(errMsg)
		return Response{
			Error: errors.New(errMsg),
		}
	}

	if len(resp.Choices) == 0 {
		errMsg := "OpenAI response contained no choices"
		logger.Error(errMsg)
		return Response{
			Error: errors.New(errMsg),
		}
	}
	// Check if there's a tool call in the response
	if len(resp.Choices) > 0 && len(resp.Choices[0].Message.ToolCalls) > 0 {
		logger.Debug("Tool call detected in response")
		// Process tool calls and get a follow-up response if needed
		return o.handleToolCalls(ctx, resp, req)
	}

	return Response{
		Content: resp.Choices[0].Message.Content,
	}
}

// handleToolCalls processes any tool calls in the response and sends follow-up requests if needed
func (o *OpenAIModel) handleToolCalls(ctx context.Context, resp openai.ChatCompletionResponse, originalReq Request) Response {
	toolCalls := resp.Choices[0].Message.ToolCalls

	// Log all tool calls
	for _, tool := range toolCalls {
		logger.Debugf("Tool call: %s, arguments: %s", tool.Function.Name, tool.Function.Arguments)
	}

	// Create a new message list starting with the original messages
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
		if tool.Function.Name == "get_git_diff" {
			// Handle git diff tool call
			diffResult, err := o.processGitDiffToolCall(tool.Function.Arguments)

			// Create a tool result message
			content := diffResult
			if err != nil {
				content = fmt.Sprintf("Error executing git diff: %v", err)
				logger.Error(content)
			}

			// Add the tool result as a message
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: tool.ID,
			})
		}
	}

	// Create a follow-up request with the tool results
	followUpReq := openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2,
	}

	logger.Debug("Sending follow-up request with tool results")
	followUpResp, err := o.client.CreateChatCompletion(ctx, followUpReq)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create follow-up chat completion: %v", err)
		logger.Error(errMsg)
		return Response{
			Error:     errors.New(errMsg),
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		}
	}

	if len(followUpResp.Choices) == 0 {
		errMsg := "follow-up OpenAI response contained no choices"
		logger.Error(errMsg)
		return Response{
			Error:     errors.New(errMsg),
			ToolCalls: resp.Choices[0].Message.ToolCalls,
		}
	}

	// Return the final response with tool calls info preserved
	return Response{
		Content:   followUpResp.Choices[0].Message.Content,
		ToolCalls: resp.Choices[0].Message.ToolCalls,
	}
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

	// Check that both base and head are provided
	if args.Base == "" || args.Head == "" {
		return "", fmt.Errorf("both base and head must be provided")
	}

	logger.Debugf("Executing git diff between %s and %s", args.Base, args.Head)

	// Create a git runner (assuming working in current directory)
	runner := git.NewDefaultRunner(".")

	// Execute the diff command between the specified references
	// We'll use a custom range
	diffRange := fmt.Sprintf("%s..%s", args.Base, args.Head)

	// Since the getDiff method is not directly exported, we'll use a similar approach
	diffArgs := []string{"diff", "--find-renames=" + git.DefaultRenameThreshold,
		"--diff-algorithm=" + git.DefaultDiffAlgorithm, diffRange}

	// Execute the git command using the runner
	output, err := runner.Run("git", diffArgs...)
	if err != nil {
		return "", fmt.Errorf("git diff command failed: %v", err)
	}

	return output, nil
}
