package providers

import (
	"context"
	"time"
)

// LLMProvider defines the interface that all LLM provider adapters must implement
type LLMProvider interface {
	// Initialize validates credentials and prepares the provider for use.
	// Must be called before Execute. Returns error if credentials are invalid.
	// Should complete within 2 seconds.
	Initialize(ctx context.Context) error

	// Execute sends a request to the LLM provider and returns the response.
	// Implements retry logic internally according to RetryConfig.
	// Returns error if all retry attempts fail.
	Execute(ctx context.Context, req Request) (Response, error)

	// Name returns the unique identifier for this provider instance.
	// Used for metrics tracking and logging.
	Name() string

	// Type returns the provider type (openai, anthropic, google).
	Type() ProviderType

	// Shutdown gracefully closes any resources held by the provider.
	// Must be safe to call multiple times.
	Shutdown(ctx context.Context) error
}

// ProviderRegistry defines the interface for managing provider selection and routing
type ProviderRegistry interface {
	// SelectProvider chooses the appropriate provider for a given role.
	// Returns the provider instance and the provider ID.
	// Falls back through FallbackProviders and DefaultProvider if needed.
	// Returns error only if no provider is available for the role.
	SelectProvider(ctx context.Context, role Role) (LLMProvider, string, error)

	// RecordMetrics records performance metrics for a completed task.
	// Non-blocking - metrics are written asynchronously if possible.
	RecordMetrics(ctx context.Context, metric ProviderMetrics) error

	// GetMetrics retrieves aggregated metrics for a provider-role combination.
	// Time range filtering via since parameter (zero value = all time).
	// Returns summary statistics (avg response time, success rate, etc.).
	GetMetrics(ctx context.Context, providerID string, role Role, since time.Time) (*MetricsSummary, error)

	// Shutdown gracefully shuts down all providers and flushes metrics.
	Shutdown(ctx context.Context) error
}

// ProviderMetrics represents a single metric event for tracking
type ProviderMetrics struct {
	ProviderID       string       // Provider that handled the request
	Role             Role         // Role context for the request
	Timestamp        time.Time    // When the metric was recorded
	ResponseTimeMs   int64        // Response time in milliseconds
	TokensPrompt     int          // Number of tokens in the prompt
	TokensCompletion int          // Number of tokens in the completion
	Status           MetricStatus // Outcome (success, failure, retry)
	ErrorMessage     string       // Error details if Status is failure
}

// MetricsSummary represents aggregated metrics for a provider-role combination
type MetricsSummary struct {
	ProviderID      string    // Provider identifier
	Role            Role      // Role identifier
	TimeRange       TimeRange // Time range for the metrics
	AvgResponseTime float64   // Average response time in ms
	P50ResponseTime int64     // 50th percentile response time
	P95ResponseTime int64     // 95th percentile response time
	TotalRequests   int       // Total number of requests
	SuccessCount    int       // Successful requests
	FailureCount    int       // Failed requests
	SuccessRate     float64   // Success rate (0.0 - 1.0)
	TotalTokens     int       // Sum of prompt + completion tokens
	AvgTokensPerReq float64   // Average tokens per request
}

// TimeRange represents a time range for metrics queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}
