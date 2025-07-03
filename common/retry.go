package common

import (
	"time"

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

	// Only set CheckRetry if provided (non-nil)
	if config.CheckRetry != nil {
		retryClient.CheckRetry = config.CheckRetry
	}

	return retryClient
}
