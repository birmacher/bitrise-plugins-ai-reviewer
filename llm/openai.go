package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
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
	if req.Diff != "" {
		logger.Debug("Including diff in OpenAI prompt")
		logger.Debug(req.Diff)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Diff,
		})
	}

	// Add file contents if available
	if req.FileContents != "" {
		logger.Debug("Including file contents in OpenAI prompt")
		logger.Debug(req.FileContents)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.FileContents,
		})
	}

	// // Add line-level feedback if available
	// if req.LineLevelFeedback != "" {
	// 	messages = append(messages, openai.ChatCompletionMessage{
	// 		Role:    openai.ChatMessageRoleAssistant,
	// 		Content: req.LineLevelFeedback,
	// 	})
	// }

	// Create the completion request
	chatReq := openai.ChatCompletionRequest{
		Model:       o.modelName,
		Messages:    messages,
		MaxTokens:   o.maxTokens,
		Temperature: 0.2, // Lower temperature for more deterministic results
	}

	logger.Infof("Sending request to OpenAI with model %s, max tokens %d", o.modelName, o.maxTokens)

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

	return Response{
		Content: resp.Choices[0].Message.Content,
	}
}
