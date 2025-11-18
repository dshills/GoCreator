// Package providers implements multi-LLM provider support with role-based routing.
package providers

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/dshills/gocreator/internal/yamlutil"
)

// ProviderConfig represents the configuration for a single LLM provider
type ProviderConfig struct {
	ID         string         `yaml:"-"`          // Unique identifier (populated from map key)
	Type       ProviderType   `yaml:"type"`       // Provider type (openai, anthropic, google)
	Model      string         `yaml:"model"`      // Model identifier
	APIKey     string         `yaml:"api_key"`    // Authentication credential (supports env vars)
	Endpoint   string         `yaml:"endpoint"`   // Custom API endpoint URL (optional)
	Parameters map[string]any `yaml:"parameters"` // Provider-specific global parameters
}

// Validate validates the provider configuration
func (c *ProviderConfig) Validate() error {
	// Validate ID format
	if c.ID == "" {
		return NewConfigError("providers.<id>", "provider ID must not be empty", nil)
	}
	idPattern := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	if !idPattern.MatchString(c.ID) {
		return NewConfigError("providers."+c.ID, "provider ID must match pattern: ^[a-z0-9][a-z0-9-]*[a-z0-9]$", nil)
	}

	// Validate type
	if c.Type == "" {
		return NewConfigError("providers."+c.ID+".type", "provider type must be specified", nil)
	}
	if c.Type != ProviderTypeOpenAI && c.Type != ProviderTypeAnthropic && c.Type != ProviderTypeGoogle {
		return NewConfigError("providers."+c.ID+".type", fmt.Sprintf("unsupported provider type: %s", c.Type), nil)
	}

	// Validate model
	if c.Model == "" {
		return NewConfigError("providers."+c.ID+".model", "model must be specified", nil)
	}

	// Validate API key (after env var expansion)
	if c.APIKey == "" {
		return NewConfigError("providers."+c.ID+".api_key", "API key must not be empty", nil)
	}

	// Validate parameters if present
	if c.Parameters != nil {
		if temp, ok := c.Parameters["temperature"].(float64); ok {
			if temp < 0.0 || temp > 2.0 {
				return NewConfigError("providers."+c.ID+".parameters.temperature", "temperature must be between 0.0 and 2.0", nil)
			}
		}
		if maxTokens, ok := c.Parameters["max_tokens"].(int); ok {
			if maxTokens <= 0 {
				return NewConfigError("providers."+c.ID+".parameters.max_tokens", "max_tokens must be greater than 0", nil)
			}
		}
	}

	return nil
}

// RoleAssignment maps a specialized role to one or more providers with priority order
type RoleAssignment struct {
	PrimaryProvider    string         `yaml:"provider"`   // Primary provider ID
	FallbackProviders  []string       `yaml:"fallback"`   // Ordered list of fallback provider IDs
	ParameterOverrides map[string]any `yaml:"parameters"` // Role-specific parameter overrides
}

// Validate validates the role assignment
func (r *RoleAssignment) Validate(role Role, providers map[string]*ProviderConfig) error {
	// Validate primary provider exists
	if r.PrimaryProvider == "" {
		return NewConfigError("roles."+string(role)+".provider", "primary provider must be specified", nil)
	}
	if _, exists := providers[r.PrimaryProvider]; !exists {
		return NewConfigError("roles."+string(role)+".provider", fmt.Sprintf("provider '%s' not found", r.PrimaryProvider), nil)
	}

	// Validate fallback providers
	seen := make(map[string]bool)
	for i, fallback := range r.FallbackProviders {
		if _, exists := providers[fallback]; !exists {
			return NewConfigError(fmt.Sprintf("roles.%s.fallback[%d]", role, i), fmt.Sprintf("provider '%s' not found", fallback), nil)
		}
		if seen[fallback] {
			return NewConfigError(fmt.Sprintf("roles.%s.fallback[%d]", role, i), fmt.Sprintf("duplicate fallback provider '%s'", fallback), nil)
		}
		seen[fallback] = true
	}

	// Validate parameter overrides (must not include critical parameters)
	if r.ParameterOverrides != nil {
		criticalParams := []string{"type", "model", "api_key", "endpoint"}
		for _, param := range criticalParams {
			if _, exists := r.ParameterOverrides[param]; exists {
				return NewConfigError("roles."+string(role)+".parameters", fmt.Sprintf("cannot override critical parameter '%s'", param), nil)
			}
		}

		// Validate parameter types and ranges
		if temp, ok := r.ParameterOverrides["temperature"].(float64); ok {
			if temp < 0.0 || temp > 2.0 {
				return NewConfigError("roles."+string(role)+".parameters.temperature", "temperature must be between 0.0 and 2.0", nil)
			}
		}
		if maxTokens, ok := r.ParameterOverrides["max_tokens"].(int); ok {
			if maxTokens <= 0 {
				return NewConfigError("roles."+string(role)+".parameters.max_tokens", "max_tokens must be greater than 0", nil)
			}
		}
	}

	return nil
}

// MultiProviderConfig represents the complete multi-provider configuration
type MultiProviderConfig struct {
	SchemaVersion   string                     `yaml:"schema_version"`   // Schema version for compatibility
	Providers       map[string]*ProviderConfig `yaml:"providers"`        // Provider ID -> Config
	Roles           map[Role]*RoleAssignment   `yaml:"roles"`            // Role -> Assignment
	DefaultProvider string                     `yaml:"default_provider"` // Fallback if role has no assignment
	Retry           *RetryConfig               `yaml:"retry"`            // Global retry configuration
}

// LoadConfig loads and validates a multi-provider configuration from a YAML file
func LoadConfig(path string) (*MultiProviderConfig, error) {
	// Read file
	//nolint:gosec // File path comes from configuration, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewConfigError("", fmt.Sprintf("failed to read config file: %s", path), err)
	}

	// Expand environment variables in the YAML content
	expandedData := os.ExpandEnv(string(data))

	// Parse YAML with enhanced error reporting
	var config MultiProviderConfig
	if err := yamlutil.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, NewConfigError("", "failed to parse provider configuration", err)
	}

	// Populate provider IDs from map keys
	for id, providerCfg := range config.Providers {
		providerCfg.ID = id
	}

	// Apply backward compatibility transformation if needed
	config = *transformLegacyConfig(&config)

	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidateConfig validates a multi-provider configuration
func ValidateConfig(config *MultiProviderConfig) error {
	// At least one provider must be defined
	if len(config.Providers) == 0 {
		return NewConfigError("providers", "at least one provider must be defined", nil)
	}

	// Validate each provider
	for _, providerCfg := range config.Providers {
		if err := providerCfg.Validate(); err != nil {
			return err
		}
	}

	// Validate default provider exists
	if config.DefaultProvider == "" {
		return NewConfigError("default_provider", "default provider must be specified", nil)
	}
	if _, exists := config.Providers[config.DefaultProvider]; !exists {
		return NewConfigError("default_provider", fmt.Sprintf("default provider '%s' not found", config.DefaultProvider), nil)
	}

	// Validate each role assignment
	for role, assignment := range config.Roles {
		if err := assignment.Validate(role, config.Providers); err != nil {
			return err
		}
	}

	// Check for circular fallback chains
	if err := validateNoCircularFallbacks(config.Roles); err != nil {
		return err
	}

	// Validate retry configuration
	if config.Retry != nil {
		if config.Retry.MaxAttempts <= 0 || config.Retry.MaxAttempts > 10 {
			return NewConfigError("retry.max_attempts", "max_attempts must be between 1 and 10", nil)
		}
		if config.Retry.InitialBackoff <= 0 {
			return NewConfigError("retry.initial_backoff", "initial_backoff must be greater than 0", nil)
		}
		if config.Retry.MaxBackoff <= config.Retry.InitialBackoff {
			return NewConfigError("retry.max_backoff", "max_backoff must be greater than initial_backoff", nil)
		}
		if config.Retry.Multiplier <= 1.0 {
			return NewConfigError("retry.multiplier", "multiplier must be greater than 1.0", nil)
		}
	} else {
		// Set defaults if not provided
		config.Retry = DefaultRetryConfig()
	}

	return nil
}

// validateNoCircularFallbacks checks that there are no circular references in fallback chains
func validateNoCircularFallbacks(_ map[Role]*RoleAssignment) error {
	// This is a simple validation - we don't need to detect cycles in fallback chains
	// because fallbacks are provider IDs, not role references
	// The only validation needed is that fallback providers exist, which is done in RoleAssignment.Validate
	return nil
}

// transformLegacyConfig converts old single-provider config to new multi-provider format
func transformLegacyConfig(config *MultiProviderConfig) *MultiProviderConfig {
	// Check if this is a legacy single-provider configuration
	// Legacy format would have a single "provider" field instead of "providers" map
	// Since we're using the new format in the struct, this is a no-op for now
	// This function exists for future backward compatibility needs
	return config
}

// MergeParameters merges global provider parameters with role-specific overrides
// Role parameters take precedence over global parameters
func MergeParameters(globalParams, roleParams map[string]any) map[string]any {
	merged := make(map[string]any)

	// Copy global parameters
	for k, v := range globalParams {
		merged[k] = v
	}

	// Apply role overrides
	for k, v := range roleParams {
		merged[k] = v
	}

	return merged
}

// ParseDuration parses a duration string from YAML (supports units like "1s", "30s", etc.)
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
