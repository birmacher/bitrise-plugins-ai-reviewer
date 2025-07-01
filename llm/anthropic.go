package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicModel implements the LLM interface using Anthropic's API
type AnthropicModel struct {
	client     anthropic.Client
	modelName  string
	maxTokens  int
	apiKey     string
	apiTimeout int // in seconds
}

// NewAnthropic creates a new Anthropic client
func NewAnthropic(apiKey string, opts ...Option) (*AnthropicModel, error) {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	model := &AnthropicModel{
		apiKey:     apiKey,
		client:     client,
		modelName:  "claude-3.7-sonnet", // Default model
		maxTokens:  4000,                // Default max tokens
		apiTimeout: 30,                  // Default timeout in seconds
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

// Prompt sends a request to Anthropic and returns the response
func (a *AnthropicModel) Prompt(req Request) Response {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(a.apiTimeout)*time.Second)
	defer cancel()

	// Combine user content (prompt, diff, file contents)
	messages := []anthropic.MessageParam{
		{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(req.UserPrompt),
			},
		},
	}

	if req.Diff != "" {
		messages = append(messages, anthropic.MessageParam{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(req.Diff),
			},
		})
	}

	if req.FileContents != "" {
		messages = append(messages, anthropic.MessageParam{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(req.FileContents),
			},
		})
	}

	if req.LineLevelFeedback != "" {
		messages = append(messages, anthropic.MessageParam{
			Role: anthropic.MessageParamRoleAssistant,
			Content: []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(req.LineLevelFeedback),
			},
		})
	}

	// Convert model name string to anthropic.Model
	var model anthropic.Model
	switch a.modelName {
	case "claude-3.7-sonnet":
		model = anthropic.ModelClaude3_7SonnetLatest
	case "claude-3.5-sonnet":
		model = anthropic.ModelClaude3_5SonnetLatest
	case "claude-3.5-haiku":
		model = anthropic.ModelClaude3_5HaikuLatest
	default:
		model = anthropic.ModelClaude3_7SonnetLatest // Default fallback
	}

	// Create the message request
	messageParams := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: int64(a.maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		},
		Messages: messages,
	}

	// Make the API call
	message, err := a.client.Messages.New(ctx, messageParams)
	if err != nil {
		return Response{
			Error: fmt.Errorf("failed to create message: %w", err),
		}
	}

	// Extract text content from the response
	var content string
	for _, block := range message.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			content += b.Text
		}
	}

	return Response{
		Content: content,
	}
}
