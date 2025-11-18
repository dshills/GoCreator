package providers

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// RetryConfig defines the retry behavior for provider operations
type RetryConfig struct {
	MaxAttempts    int           `yaml:"max_attempts"`    // Maximum number of retry attempts (1-10)
	InitialBackoff time.Duration `yaml:"initial_backoff"` // Initial backoff duration (must be > 0)
	MaxBackoff     time.Duration `yaml:"max_backoff"`     // Maximum backoff duration (must be > InitialBackoff)
	Multiplier     float64       `yaml:"multiplier"`      // Backoff multiplier (must be > 1.0)
}

// UnmarshalYAML implements custom YAML unmarshaling for duration strings
func (r *RetryConfig) UnmarshalYAML(node *yaml.Node) error {
	// Define a temporary struct with string fields for durations
	var temp struct {
		MaxAttempts    int     `yaml:"max_attempts"`
		InitialBackoff string  `yaml:"initial_backoff"`
		MaxBackoff     string  `yaml:"max_backoff"`
		Multiplier     float64 `yaml:"multiplier"`
	}

	if err := node.Decode(&temp); err != nil {
		return err
	}

	r.MaxAttempts = temp.MaxAttempts
	r.Multiplier = temp.Multiplier

	// Parse duration strings
	if temp.InitialBackoff != "" {
		d, err := time.ParseDuration(temp.InitialBackoff)
		if err != nil {
			return fmt.Errorf("invalid initial_backoff duration: %w", err)
		}
		r.InitialBackoff = d
	}

	if temp.MaxBackoff != "" {
		d, err := time.ParseDuration(temp.MaxBackoff)
		if err != nil {
			return fmt.Errorf("invalid max_backoff duration: %w", err)
		}
		r.MaxBackoff = d
	}

	return nil
}

// Execute executes the given function with retry logic using exponential backoff.
// It respects context cancellation and returns immediately if the context is done.
// The function fn should return nil on success, or an error that will trigger a retry.
func (r *RetryConfig) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	backoff := r.InitialBackoff

	for attempt := 1; attempt <= r.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}
		lastErr = err

		// If this was the last attempt, don't sleep
		if attempt >= r.MaxAttempts {
			break
		}

		// Sleep with exponential backoff, respecting context cancellation
		select {
		case <-time.After(backoff):
			// Calculate next backoff duration
			backoff = time.Duration(float64(backoff) * r.Multiplier)
			if backoff > r.MaxBackoff {
				backoff = r.MaxBackoff
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", r.MaxAttempts, lastErr)
}

// DefaultRetryConfig returns a retry configuration with sensible defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
	}
}
