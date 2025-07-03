package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-retryablehttp"
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
		return nil, fmt.Errorf("API key cannot be empty")
	}

	// Create retryable HTTP client with exponential backoff
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.CheckRetry = retryablehttp.DefaultRetryPolicy

	// Use the retryable client for OpenAI
	config := openai.DefaultConfig(apiKey)
	config.HTTPClient = retryClient.StandardClient()

	model := &OpenAIModel{
		client:     openai.NewClientWithConfig(config),
		modelName:  "gpt-4.1", // Default model
		maxTokens:  4000,      // Default max tokens
		apiTimeout: 30,        // Default timeout in seconds
	}

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.apiTimeout)*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
	}

	// Add user prompt
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.UserPrompt,
	})

	// Add diff if available
	if req.Diff != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Diff,
		})
	}

	// Add file contents if available
	if req.FileContents != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.FileContents,
		})
	}

	// Add line-level feedback if available
	if req.LineLevelFeedback != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: req.LineLevelFeedback,
		})
	}

	// Create the completion request
	chatReq := openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2, // Lower temperature for more deterministic results
	}

	resp, err := o.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return Response{
			Error: fmt.Errorf("failed to create chat completion: %w", err),
		}
	}

	if len(resp.Choices) == 0 {
		return Response{
			Error: fmt.Errorf("no response choices returned"),
		}
	}

	return Response{
		Content: resp.Choices[0].Message.Content,
	}
}
