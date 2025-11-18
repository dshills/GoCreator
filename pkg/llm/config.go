package llm

import (
	"fmt"
	"strings"
	"time"
)

// Provider represents supported LLM providers
type Provider string

const (
	// ProviderAnthropic represents Anthropic (Claude) provider
	ProviderAnthropic Provider = "anthropic"
	// ProviderOpenAI represents OpenAI (GPT) provider
	ProviderOpenAI Provider = "openai"
	// ProviderGoogle represents Google (Gemini) provider
	ProviderGoogle Provider = "google"
)

// Config holds LLM client configuration
type Config struct {
	// Provider specifies which LLM provider to use (anthropic, openai, google)
	Provider Provider

	// Model specifies the model name (e.g., "claude-sonnet-4", "gpt-4", "gemini-pro")
	Model string

	// Temperature controls randomness in responses. MUST be 0.0 for determinism.
	Temperature float64

	// APIKey is the authentication key for the provider
	APIKey string

	// Timeout specifies the maximum duration for API calls
	Timeout time.Duration

	// MaxTokens specifies the maximum number of tokens to generate
	MaxTokens int

	// MaxRetries specifies the maximum number of retry attempts on failure
	MaxRetries int

	// RetryDelay specifies the initial delay between retries (exponential backoff)
	RetryDelay time.Duration

	// EnableCaching enables prompt caching (Anthropic only)
	// When enabled, stable portions of prompts (FCS schema, guidelines) are cached
	EnableCaching bool

	// CacheTTL specifies the cache time-to-live (5m or 1h)
	// Defaults to 5m if not specified
	CacheTTL string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Provider:      ProviderAnthropic,
		Model:         "claude-sonnet-4",
		Temperature:   0.0, // MUST be 0.0 for deterministic output
		Timeout:       120 * time.Second,
		MaxTokens:     4096,
		MaxRetries:    3,
		RetryDelay:    time.Second,
		EnableCaching: true, // Enable caching by default for cost savings
		CacheTTL:      "5m", // Default to 5-minute cache
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	// Validate provider
	switch c.Provider {
	case ProviderAnthropic, ProviderOpenAI, ProviderGoogle:
		// Valid provider
	default:
		return fmt.Errorf("invalid provider: %s (must be one of: anthropic, openai, google)", c.Provider)
	}

	// Validate model name
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// Enforce temperature = 0.0 for determinism
	if c.Temperature != 0.0 {
		return fmt.Errorf("temperature must be 0.0 for deterministic output, got: %f", c.Temperature)
	}

	// Validate API key
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("API key cannot be empty for provider: %s", c.Provider)
	}

	// Validate timeout
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got: %v", c.Timeout)
	}

	// Validate max tokens
	if c.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive, got: %d", c.MaxTokens)
	}

	// Validate retry settings
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative, got: %d", c.MaxRetries)
	}

	if c.RetryDelay <= 0 {
		return fmt.Errorf("retry delay must be positive, got: %v", c.RetryDelay)
	}

	// Validate cache TTL if caching is enabled
	if c.EnableCaching {
		if c.CacheTTL != "" && c.CacheTTL != "5m" && c.CacheTTL != "1h" {
			return fmt.Errorf("cache TTL must be either '5m' or '1h', got: %s", c.CacheTTL)
		}
		// Note: Default CacheTTL ("5m") is set in DefaultConfig()
	}

	return nil
}

// String returns a human-readable representation of the config (without sensitive data)
func (c Config) String() string {
	apiKeyMasked := "***"
	if len(c.APIKey) > 4 {
		apiKeyMasked = c.APIKey[:4] + "***"
	}

	return fmt.Sprintf("Provider=%s Model=%s Temperature=%.1f Timeout=%v MaxTokens=%d APIKey=%s",
		c.Provider, c.Model, c.Temperature, c.Timeout, c.MaxTokens, apiKeyMasked)
}
