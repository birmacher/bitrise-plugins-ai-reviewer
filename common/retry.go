package common

import (
	"time"

	"github.com/bitrise-io/bitrise-plugins-ai-reviewer/logger"
	"github.com/hashicorp/go-retryablehttp"
)

// RetryConfig holds the configuration for HTTP retry logic
type RetryConfig struct {
	// Maximum number of retries
	RetryMax int
	// Minimum time to wait between retries
	RetryWaitMin time.Duration
	// Maximum time to wait between retries
	RetryWaitMax time.Duration
	// Function to determine if a request should be retried
	CheckRetry retryablehttp.CheckRetry
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		RetryMax:     3,
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 5 * time.Second,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
	}
}

// NewRetryableClient creates a new HTTP client with retry capabilities
func NewRetryableClient(config RetryConfig) *retryablehttp.Client {
	retryClient := retryablehttp.NewClient()

	// Apply configuration
	retryClient.RetryMax = config.RetryMax
	retryClient.RetryWaitMin = config.RetryWaitMin
	retryClient.RetryWaitMax = config.RetryWaitMax

	logger.Debugf("Created retryable client with max retries: %d, min wait: %s, max wait: %s",
		config.RetryMax, config.RetryWaitMin, config.RetryWaitMax)

	// Only set CheckRetry if provided (non-nil)
	if config.CheckRetry != nil {
		retryClient.CheckRetry = config.CheckRetry
	}

	// Add logging for retries
	retryClient.Logger = &zapRetryLogger{}

	return retryClient
}

// zapRetryLogger adapts our zap logger to the interface required by retryablehttp
type zapRetryLogger struct{}

func (z *zapRetryLogger) Error(msg string, keysAndValues ...interface{}) {
	logger.Error(append([]interface{}{msg}, keysAndValues...)...)
}

func (z *zapRetryLogger) Info(msg string, keysAndValues ...interface{}) {
	logger.Info(append([]interface{}{msg}, keysAndValues...)...)
}

func (z *zapRetryLogger) Debug(msg string, keysAndValues ...interface{}) {
	logger.Debug(append([]interface{}{msg}, keysAndValues...)...)
}

func (z *zapRetryLogger) Warn(msg string, keysAndValues ...interface{}) {
	logger.Warn(append([]interface{}{msg}, keysAndValues...)...)
}
