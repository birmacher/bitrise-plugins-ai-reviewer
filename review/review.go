package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

// BaseReviewer contains common fields and methods shared by all reviewer implementations
type BaseReviewer struct {
	Provider string
	ApiToken string
	Timeout  int
	BaseURL  string
}

// NewBaseReviewer creates a new base reviewer with common options applied
func NewBaseReviewer(provider string, opts ...Option) (*BaseReviewer, error) {
	baseReviewer := &BaseReviewer{
		Provider: provider,
		Timeout:  60, // Default timeout
	}

	// Get API token
	apiToken, err := getAPIToken(provider)
	if err != nil {
		return nil, err
	}
	baseReviewer.ApiToken = apiToken

	// Apply additional options
	for _, opt := range opts {
		switch opt.Type {
		case APITokenOption:
			if token, ok := opt.Value.(string); ok {
				baseReviewer.ApiToken = token
				logger.Debugf("%s API token configured", provider)
			}
		case TimeoutOption:
			if timeout, ok := opt.Value.(int); ok {
				baseReviewer.Timeout = timeout
				logger.Debugf("%s API timeout set to %d seconds", provider, timeout)
			}
		case BaseURLOption:
			if baseURL, ok := opt.Value.(string); ok {
				baseReviewer.BaseURL = baseURL
				logger.Debugf("%s base URL configured: %s", provider, baseURL)
			}
		}
	}

	// Validate required options
	if baseReviewer.ApiToken == "" {
		errMsg := fmt.Sprintf("%s API token is required", provider)
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	return baseReviewer, nil
}

// CreateTimeoutContext creates a timeout context for API calls
func (br *BaseReviewer) CreateTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(br.Timeout)*time.Second)
}

// FormatNitpickComments formats nitpick comments for display in PR summaries
func FormatNitpickComments(provider string, nitpickCommentsByFile map[string][]common.LineLevel) []string {
	nitpickComments := []string{}

	for filepath, comments := range nitpickCommentsByFile {
		content := strings.Builder{}
		content.WriteString("<details>\n")
		content.WriteString("<summary>" + filepath + " (" + strconv.Itoa(len(comments)) + ")</summary>\n\n")

		for _, c := range comments {
			line := fmt.Sprintf("%d", c.LineNumber)
			if c.IsMultiline() {
				line = line + "-" + fmt.Sprintf("%d", c.LastLineNumber)
			}
			content.WriteString("<!-- bitrise-plugin-ai-reviewer: " + filepath + ":" + line + " -->\n")
			content.WriteString("`" + line + "`: **" + c.Title + "**\n\n")
			content.WriteString(c.Body + "\n\n")
		}
		content.WriteString("</details>\n\n")

		nitpickComments = append(nitpickComments, content.String())
	}

	return nitpickComments
}

// ProcessLineFeedbackItems processes line feedback items and checks for duplicates
func ProcessLineFeedbackItems(
	provider string,
	client *git.Client,
	commitHash string,
	existingComments []common.LineLevel,
	lineFeedback common.LineLevelFeedback,
) (map[string][]common.LineLevel, error) {
	nitpickCommentsByFile := make(map[string][]common.LineLevel)

	// Process nitpick comments
	for _, ll := range lineFeedback.GetNitpickFeedback() {
		if ll.File == "" || ll.LineNumber <= 0 {
			continue
		}
		if nitpickCommentsByFile[ll.File] == nil {
			nitpickCommentsByFile[ll.File] = []common.LineLevel{}
		}
		nitpickCommentsByFile[ll.File] = append(nitpickCommentsByFile[ll.File], ll)
	}

	return nitpickCommentsByFile, nil
}

// Reviewer defines the interface for code review interactions
type Reviewer interface {
	GetProvider() string
	SupportCollapsibleMarkdown() bool
	// ListComments(repoOwner, repoName string, pr int) ([]string, error)
	PostSummaryUnderReview(repoOwner, repoName string, pr int, header string) error
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

// FormatOverallReview formats the overall review comment including nitpick comments
func FormatOverallReview(actionableCommentCount int, nitpickComments []string) string {
	overallReview := strings.Builder{}
	overallReview.WriteString("_This is an AI-generated review. Please review it carefully._\n\n")
	overallReview.WriteString(fmt.Sprintf("**Actionable comments posted: %d**\n\n", actionableCommentCount))

	if len(nitpickComments) > 0 {
		overallReview.WriteString("<details>\n")
		overallReview.WriteString("<summary>🧹 Nitpick comments</summary>\n")
		overallReview.WriteString(strings.Join(nitpickComments, "\n\n---\n\n"))
		overallReview.WriteString("</details>\n\n")
	}

	return overallReview.String()
}

// CreateCommonPRComment formats a common PR comment for line feedback
func CreateCommonPRComment(provider string, nitpickComments []string, commentCount int) string {
	return FormatOverallReview(commentCount, nitpickComments)
}
