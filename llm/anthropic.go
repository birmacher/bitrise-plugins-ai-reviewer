package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"
)

// AnthropicModel implements the LLM interface using Anthropic's API
type AnthropicModel struct {
	client      anthropic.Client
	modelName   string
	maxTokens   int
	apiTimeout  int // in seconds
	GitProvider *review.Reviewer
	Settings    *common.Settings
}

// NewAnthropic creates a new Anthropic client
func NewAnthropic(apiKey string, opts ...Option) (*AnthropicModel, error) {
	if apiKey == "" {
		errMsg := "anthropic API key cannot be empty"
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	// Create retryable HTTP client with exponential backoff using common configuration
	retryClient := common.NewRetryableClient(common.DefaultRetryConfig())

	// Get standard HTTP client from retryable client
	standardClient := retryClient.StandardClient()

	// Create Anthropic client with retry capabilities
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(standardClient),
	)

	model := &AnthropicModel{
		client:     client,
		modelName:  "claude-3-sonnet", // Default model
		maxTokens:  4000,              // Default max tokens
		apiTimeout: 30,                // Default timeout in seconds
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

func (a *AnthropicModel) SetGitProvider(gitProvider *review.Reviewer) {
	a.GitProvider = gitProvider
}

func (a *AnthropicModel) SetSettings(settings *common.Settings) {
	a.Settings = settings
}

// Prompt sends a request to Anthropic and returns the response
func (a *AnthropicModel) Prompt(req Request) Response {
	logger.Debugf("Sending prompt to Anthropic model: %s", a.modelName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(a.apiTimeout)*time.Second)
	defer cancel()

	userContent := []string{req.UserPrompt}
	logger.Debug("Including message in Anthropic prompt")
	logger.Debug(req.UserPrompt)

	if req.Diff != "" {
		logger.Debug("Including diff in Anthropic prompt")
		logger.Debug(req.Diff)
		userContent = append(userContent, req.Diff)
	}

	if req.FileContents != "" {
		logger.Debug("Including file contents in Anthropic prompt")
		logger.Debug(req.FileContents)
		userContent = append(userContent, req.FileContents)
	}

	// if req.LineLevelFeedback != "" {
	// 	userContent = append(userContent, req.LineLevelFeedback)
	// }

	// Convert model name string to anthropic.Model
	var model anthropic.Model
	switch a.modelName {
	case "claude-3-sonnet":
		model = anthropic.ModelClaude3_7SonnetLatest
	case "claude-3-haiku":
		model = anthropic.ModelClaude3_5HaikuLatest
	case "claude-4-sonnet":
		model = anthropic.ModelClaudeSonnet4_0
	default:
		model = anthropic.ModelClaude3_7SonnetLatest // Default fallback
	}

	logger.Debug("Including system message in Anthropic prompt")
	logger.Debug(req.SystemPrompt)

	logger.Debugf("Using Anthropic model: %s with max tokens: %d", a.modelName, a.maxTokens)

	// Create the message request
	messageParams := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: int64(a.maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					anthropic.NewTextBlock(strings.Join(userContent, "\n\n")),
				},
			},
		},
	}

	// Make the API call
	message, err := a.client.Messages.New(ctx, messageParams)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create message with Anthropic: %v", err)
		logger.Errorf(errMsg)
		return Response{
			Error: errors.New(errMsg),
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

	logger.Debugf("Received response from Anthropic: %s", content)

	return Response{
		Content: content,
	}
}
