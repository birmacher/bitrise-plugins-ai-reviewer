package review

import (
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
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
	BaseURLOption  OptionType = "base_url"
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

// WithBaseURL creates an option to set the base URL for GitHub Enterprise
func WithBaseURL(baseURL string) Option {
	return Option{
		Type:  BaseURLOption,
		Value: baseURL,
	}
}

// Reviewer defines the interface for code review interactions
type Reviewer interface {
	PostSummary(repoOwner, repoName string, pr int, summary common.Summary, settings common.Settings) error
	PostLineFeedback(client *git.Client, repoOwner, repoName string, pr int, commitHash string, lineFeedback common.LineLevelFeedback) error
	GetReviewRequestComments(repoOwner, repoName string, pr int) ([]common.LineLevel, error)
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

	// Check for GitHub Enterprise URL
	if githubURL := os.Getenv("GITHUB_API_URL"); githubURL != "" && providerName == ProviderGitHub {
		options = append(options, WithBaseURL(githubURL))
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
