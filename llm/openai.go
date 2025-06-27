package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIModel implements the LLM interface using OpenAI's API
type OpenAIModel struct {
	client     *openai.Client
	modelName  string
	maxTokens  int
	apiKey     string
	apiTimeout int // in seconds
}

// OpenAIOption is a function that configures the OpenAIModel
type OpenAIOption func(*OpenAIModel)

// WithModel sets the model name
func WithModel(model string) OpenAIOption {
	return func(o *OpenAIModel) {
		o.modelName = model
	}
}

// WithMaxTokens sets the max tokens
func WithMaxTokens(maxTokens int) OpenAIOption {
	return func(o *OpenAIModel) {
		o.maxTokens = maxTokens
	}
}

// WithAPITimeout sets the API timeout in seconds
func WithAPITimeout(timeout int) OpenAIOption {
	return func(o *OpenAIModel) {
		o.apiTimeout = timeout
	}
}

// NewOpenAI creates a new OpenAI client
// It requires an API key either from the OPENAI_API_KEY environment variable
// or passed explicitly
func NewOpenAI(apiKey string, opts ...OpenAIOption) (*OpenAIModel, error) {
	model := &OpenAIModel{
		apiKey:     apiKey,
		client:     openai.NewClient(apiKey),
		modelName:  "gpt-4o-mini", // Default model
		maxTokens:  4000,          // Default max tokens
		apiTimeout: 30,            // Default timeout in seconds
	}

	// Apply options
	for _, opt := range opts {
		opt(model)
	}

	return model, nil
}

// Prompt sends a request to OpenAI and returns the response
func (o *OpenAIModel) Prompt(req Request) Response {
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(o.apiTimeout)*time.Second)
	defer cancel()

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
	}

	// Add diff if available
	if req.Diff != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Diff,
		})
	}

	// Add user prompt
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.UserPrompt,
	})

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
