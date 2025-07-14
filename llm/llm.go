package llm

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/review"
)

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
)

// OptionType defines the type of option
type OptionType string

// Available option types
const (
	ModelNameOption  OptionType = "model"
	MaxTokensOption  OptionType = "max_tokens"
	APITimeoutOption OptionType = "api_timeout"
)

// Option represents a generic configuration option for any LLM provider
type Option struct {
	Type  OptionType
	Value any
}

// WithModel creates an option to set the model name
func WithModel(model string) Option {
	return Option{
		Type:  ModelNameOption,
		Value: model,
	}
}

// WithMaxTokens creates an option to set the max tokens
func WithMaxTokens(maxTokens int) Option {
	return Option{
		Type:  MaxTokensOption,
		Value: maxTokens,
	}
}

// WithAPITimeout creates an option to set the API timeout in seconds
func WithAPITimeout(timeout int) Option {
	return Option{
		Type:  APITimeoutOption,
		Value: timeout,
	}
}

// Request represents the data needed to generate a prompt for the LLM
type Request struct {
	SystemPrompt      string
	UserPrompt        string
	Diff              string
	FileContents      string
	LineLevelFeedback string
}

// Response represents the response from the LLM
type Response struct {
	Content   string
	Error     error
	ToolCalls interface{} // Generic interface to handle different tool call structures
}

type Tools struct {
	GitProvider review.Reviewer
}

// LLM defines the interface for language model prompting
type LLM interface {
	// Prompt sends a request to the language model and returns its response
	Prompt(req Request) Response
	SetGitProvider(gitProvider *review.Reviewer)
}

func getAPIKey() (string, error) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		errMsg := "LLM_API_KEY environment variable is not set"
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}
	logger.Debug("Successfully retrieved LLM API key")
	return apiKey, nil
}

func NewLLM(providerName, modelName string, opts ...Option) (LLM, error) {
	logger.Infof("Creating new LLM client with provider: %s, model: %s", providerName, modelName)

	var llmClient LLM
	var err error

	apiKey, err := getAPIKey()
	if err != nil {
		logger.Errorf("Failed to get API key: %v", err)
		return nil, err
	}

	options := []Option{
		WithModel(modelName),
		WithMaxTokens(4000),
		WithAPITimeout(60),
	}
	options = append(options, opts...)
	switch providerName {
	case ProviderOpenAI:
		logger.Debug("Initializing OpenAI client")
		llmClient, err = NewOpenAI(apiKey, options...)
	case ProviderAnthropic:
		logger.Debug("Initializing Anthropic client")
		llmClient, err = NewAnthropic(apiKey, options...)
	default:
		errMsg := fmt.Sprintf("unsupported provider: %s", providerName)
		logger.Error(errMsg)
		err = errors.New(errMsg)
	}

	if err == nil {
		logger.Infof("Successfully created LLM client with provider: %s, model: %s", providerName, modelName)
	} else {
		logger.Errorf("Failed to create LLM client: %v", err)
	}

	return llmClient, err
}
