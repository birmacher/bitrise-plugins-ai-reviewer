package review

import (
	"errors"
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/common"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/git"
	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
)

const (
	ProviderGitHub    = "github"
	ProviderBitbucket = "bitbucket"
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
	SupportCollapsibleMarkdown() bool
	PostSummary(repoOwner, repoName string, pr int, header, body string) error
	PostLineFeedback(client *git.Client, repoOwner, repoName string, pr int, commitHash string, lineFeedback common.LineLevelFeedback) error
	GetReviewRequestComments(repoOwner, repoName string, pr int) ([]common.LineLevel, error)
}

// getAPIToken retrieves the API token from environment variables based on provider
func getAPIToken(provider string) (string, error) {
	var apiToken string
	var envVarName string

	switch provider {
	case ProviderGitHub:
		envVarName = "GITHUB_TOKEN"
		apiToken = os.Getenv(envVarName)
	case ProviderBitbucket:
		envVarName = "BITBUCKET_TOKEN"
		apiToken = os.Getenv(envVarName)
	default:
		errMsg := fmt.Sprintf("Unsupported provider: %s", provider)
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}

	if apiToken == "" {
		errMsg := fmt.Sprintf("%s environment variable is not set", envVarName)
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}

	logger.Debugf("Successfully retrieved %s API token", provider)
	return apiToken, nil
}

// NewReviewer creates a new review provider client
func NewReviewer(providerName string, opts ...Option) (Reviewer, error) {
	logger.Infof("Creating new reviewer with provider: %s", providerName)

	var reviewer Reviewer
	var err error

	apiToken, err := getAPIToken(providerName)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get API token: %v", err)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	options := []Option{
		WithAPIToken(apiToken),
		WithTimeout(60),
	}

	// Check for GitHub Enterprise URL
	if githubURL := os.Getenv("GITHUB_API_URL"); githubURL != "" && providerName == ProviderGitHub {
		logger.Infof("Using GitHub Enterprise URL: %s", githubURL)
		options = append(options, WithBaseURL(githubURL))
	}

	options = append(options, opts...)

	switch providerName {
	case ProviderGitHub:
		logger.Debug("Initializing GitHub reviewer")
		reviewer, err = NewGitHub(options...)
	case ProviderBitbucket:
		logger.Debug("Initializing Bitbucket reviewer")
		reviewer, err = NewBitbucket(options...)
	default:
		errMsg := fmt.Sprintf("unsupported review provider: %s", providerName)
		logger.Error(errMsg)
		err = errors.New(errMsg)
	}

	if err == nil {
		logger.Infof("Successfully created reviewer with provider: %s", providerName)
	} else {
		logger.Errorf("Failed to create reviewer: %v", err)
	}

	return reviewer, err
}
