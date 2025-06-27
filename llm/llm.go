package llm

import (
	"fmt"
	"os"
)

const ProviderOpenAI = "openai"

// Request represents the data needed to generate a prompt for the LLM
type Request struct {
	SystemPrompt string
	UserPrompt   string
	Diff         string
}

// Response represents the response from the LLM
type Response struct {
	Content string // Markdown formatted content
	Error   error
}

// LLM defines the interface for language model prompting
type LLM interface {
	// Prompt sends a request to the language model and returns its response
	Prompt(req Request) Response
}

func getAPIKey() (string, error) {
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	return apiKey, nil
}

func NewLLM(providerName, modelName string, opts ...Option) (LLM, error) {
	var llmClient LLM
	var err error

	apiKey, err := getAPIKey()
	if err != nil {
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
		llmClient, err = NewOpenAI(apiKey, options...)
	default:
		err = fmt.Errorf("unsupported provider: %s", providerName)
	}

	if err != nil {
		fmt.Println("")
		fmt.Println("Using LLM Provider:", providerName)
		fmt.Println("With Model:", modelName)
	}

	return llmClient, err
}
