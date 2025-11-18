package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// Message represents a chat message with role and content
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// Client provides a unified interface for LLM operations across multiple providers
type Client interface {
	// Generate produces text from a single prompt
	Generate(ctx context.Context, prompt string) (string, error)

	// GenerateStructured produces structured output based on a schema
	// The schema parameter defines the expected output structure
	// The returned interface{} should be unmarshaled according to the schema
	GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error)

	// Chat processes a sequence of messages and returns the assistant's response
	Chat(ctx context.Context, messages []Message) (string, error)

	// Provider returns the name of the LLM provider
	Provider() string

	// Model returns the model being used
	Model() string
}

// CacheableClient extends Client with prompt caching support (currently Anthropic only)
type CacheableClient interface {
	Client

	// GenerateWithCache generates text using cacheable messages for prompt caching
	// This method is only supported by Anthropic clients
	GenerateWithCache(ctx context.Context, messages []CacheableMessage) (string, error)

	// GetCacheMetrics returns the current prompt cache metrics
	GetCacheMetrics() PromptCacheMetrics

	// ResetCacheMetrics resets the cache metrics counters
	ResetCacheMetrics()
}

// baseClient provides common functionality for all provider implementations
type baseClient struct {
	config Config
}

// Provider returns the provider name
func (b *baseClient) Provider() string {
	return string(b.config.Provider)
}

// Model returns the model name
func (b *baseClient) Model() string {
	return b.config.Model
}

// retry executes a function with exponential backoff retry logic
func (b *baseClient) retry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	delay := b.config.RetryDelay

	for attempt := 0; attempt <= b.config.MaxRetries; attempt++ {
		// Execute the operation
		err := fn()

		if err == nil {
			if attempt > 0 {
				log.Info().
					Str("provider", string(b.config.Provider)).
					Str("operation", operation).
					Int("attempt", attempt+1).
					Msg("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// Check if context was cancelled
		if ctx.Err() != nil {
			return fmt.Errorf("%s cancelled: %w", operation, ctx.Err())
		}

		// Don't retry if this was the last attempt
		if attempt < b.config.MaxRetries {
			log.Warn().
				Err(err).
				Str("provider", string(b.config.Provider)).
				Str("operation", operation).
				Int("attempt", attempt+1).
				Dur("retry_delay", delay).
				Msg("Operation failed, retrying")

			// Wait before retry with exponential backoff
			select {
			case <-time.After(delay):
				delay *= 2 // Exponential backoff
			case <-ctx.Done():
				return fmt.Errorf("%s cancelled during retry: %w", operation, ctx.Err())
			}
		}
	}

	return fmt.Errorf("%s failed after %d attempts: %w", operation, b.config.MaxRetries+1, lastErr)
}

// wrapError wraps an error with provider and operation context
func (b *baseClient) wrapError(operation string, err error) error {
	return fmt.Errorf("llm[%s:%s] %s: %w", b.config.Provider, b.config.Model, operation, err)
}

// NewClient creates a new LLM client based on the provided configuration
func NewClient(config Config) (Client, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Log client creation (without sensitive data)
	log.Info().
		Str("provider", string(config.Provider)).
		Str("model", config.Model).
		Float64("temperature", config.Temperature).
		Dur("timeout", config.Timeout).
		Int("max_tokens", config.MaxTokens).
		Msg("Creating LLM client")

	// Create provider-specific client
	switch config.Provider {
	case ProviderAnthropic:
		return newAnthropicClient(config)
	case ProviderOpenAI:
		return newOpenAIClient(config)
	case ProviderGoogle:
		return newGoogleClient(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// ValidateAPIKey checks if an API key is valid (basic validation)
func ValidateAPIKey(provider Provider, apiKey string) error {
	// Provider-specific length validation
	switch provider {
	case ProviderAnthropic:
		if len(apiKey) < 20 {
			return fmt.Errorf("anthropic API key should be at least 20 characters")
		}
	case ProviderOpenAI:
		if len(apiKey) < 20 {
			return fmt.Errorf("OpenAI API key should be at least 20 characters")
		}
	case ProviderGoogle:
		if len(apiKey) < 20 {
			return fmt.Errorf("google API key should be at least 20 characters")
		}
	default:
		// For unknown providers, basic validation
		if len(apiKey) < 10 {
			return fmt.Errorf("API key for %s is too short (minimum 10 characters)", provider)
		}
	}

	return nil
}
