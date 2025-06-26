package model

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
