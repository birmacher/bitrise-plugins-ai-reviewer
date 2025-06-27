package review

import (
	"fmt"
	"os"
	"strings"
)

const (
	// ProviderGitHub represents the GitHub provider
	ProviderGitHub = "github"
	// Add more providers as needed, such as GitLab, Bitbucket, etc.
)

// OptionType defines the type of option for review providers
type OptionType string

// Available option types
const (
	APITokenOption OptionType = "api_token"
	TimeoutOption  OptionType = "timeout"
	// Add more option types as needed
)

// Option represents a generic configuration option for any review provider
type Option struct {
	Type  OptionType
	Value any
}

// WithAPIToken creates an option to set the API token
func WithAPIToken(token string) Option {
	return Option{
		Type:  APITokenOption,
		Value: token,
	}
}

// WithTimeout creates an option to set the API timeout in seconds
func WithTimeout(timeout int) Option {
	return Option{
		Type:  TimeoutOption,
		Value: timeout,
	}
}

// Comment represents a review comment to be posted
type Comment struct {
	FilePath string
	Line     int
	Body     string
}

// ReviewRequest represents the data needed for a code review
type ReviewRequest struct {
	Repository string
	PRNumber   int
	Comments   []Comment
	Summary    string
}

func (r ReviewRequest) RepoOwner() string {
	parts := strings.Split(r.Repository, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (r ReviewRequest) RepoName() string {
	parts := strings.Split(r.Repository, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// ReviewResponse represents the response from the review provider
type ReviewResponse struct {
	Success bool
	URL     string // URL to the review, if applicable
	Error   error
}

// Reviewer defines the interface for code review interactions
type Reviewer interface {
	// PostReview submits review comments to the specified PR
	PostReview(req ReviewRequest) ReviewResponse
}

// getAPIToken retrieves the API token from environment variables
func getAPIToken() (string, error) {
	apiToken := os.Getenv("GITHUB_TOKEN")
	if apiToken == "" {
		return "", fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	return apiToken, nil
}

// NewReviewer creates a new review provider client
func NewReviewer(providerName string, opts ...Option) (Reviewer, error) {
	var reviewer Reviewer
	var err error

	apiToken, err := getAPIToken()
	if err != nil {
		return nil, err
	}

	options := []Option{
		WithAPIToken(apiToken),
		WithTimeout(60),
	}
	options = append(options, opts...)

	switch providerName {
	case ProviderGitHub:
		reviewer, err = NewGitHub(options...)
	default:
		err = fmt.Errorf("unsupported review provider: %s", providerName)
	}

	if err == nil {
		fmt.Println("")
		fmt.Println("Using Review Provider:", providerName)
	}

	return reviewer, err
}
