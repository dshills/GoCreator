package providers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dshills/gocreator/internal/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ValidConfiguration(t *testing.T) {
	// Create a temporary YAML file with valid configuration
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
schema_version: "1.0"

providers:
  openai-gpt4:
    type: openai
    model: gpt-4-turbo
    api_key: sk-test-key-123
    parameters:
      temperature: 0.7
      max_tokens: 4096

  anthropic-claude:
    type: anthropic
    model: claude-3-5-sonnet-20241022
    api_key: sk-ant-test-key-456
    parameters:
      temperature: 0.5
      max_tokens: 8192

roles:
  coder:
    provider: openai-gpt4
    fallback:
      - anthropic-claude
    parameters:
      temperature: 0.8

  reviewer:
    provider: anthropic-claude
    parameters:
      temperature: 0.2

default_provider: anthropic-claude

retry:
  max_attempts: 3
  initial_backoff: 1s
  max_backoff: 30s
  multiplier: 2.0
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load configuration
	config, err := providers.LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Validate structure
	assert.Equal(t, "1.0", config.SchemaVersion)
	assert.Len(t, config.Providers, 2)
	assert.Len(t, config.Roles, 2)
	assert.Equal(t, "anthropic-claude", config.DefaultProvider)

	// Validate provider configurations
	openai := config.Providers["openai-gpt4"]
	require.NotNil(t, openai)
	assert.Equal(t, "openai-gpt4", openai.ID)
	assert.Equal(t, providers.ProviderTypeOpenAI, openai.Type)
	assert.Equal(t, "gpt-4-turbo", openai.Model)
	assert.Equal(t, "sk-test-key-123", openai.APIKey)
	assert.Equal(t, 0.7, openai.Parameters["temperature"])
	assert.Equal(t, 4096, openai.Parameters["max_tokens"])

	// Validate role assignments
	coder := config.Roles[providers.RoleCoder]
	require.NotNil(t, coder)
	assert.Equal(t, "openai-gpt4", coder.PrimaryProvider)
	assert.Equal(t, []string{"anthropic-claude"}, coder.FallbackProviders)
	assert.Equal(t, 0.8, coder.ParameterOverrides["temperature"])

	// Validate retry configuration
	require.NotNil(t, config.Retry)
	assert.Equal(t, 3, config.Retry.MaxAttempts)
}

func TestLoadConfig_EnvironmentVariableExpansion(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_OPENAI_KEY", "sk-env-test-123")
	os.Setenv("TEST_ANTHROPIC_KEY", "sk-ant-env-456")
	defer os.Unsetenv("TEST_OPENAI_KEY")
	defer os.Unsetenv("TEST_ANTHROPIC_KEY")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
schema_version: "1.0"

providers:
  openai-gpt4:
    type: openai
    model: gpt-4-turbo
    api_key: ${TEST_OPENAI_KEY}

  anthropic-claude:
    type: anthropic
    model: claude-3-5-sonnet-20241022
    api_key: ${TEST_ANTHROPIC_KEY}

default_provider: openai-gpt4
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load configuration
	config, err := providers.LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Validate environment variable expansion
	assert.Equal(t, "sk-env-test-123", config.Providers["openai-gpt4"].APIKey)
	assert.Equal(t, "sk-ant-env-456", config.Providers["anthropic-claude"].APIKey)
}

func TestLoadConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		config        string
		expectedError string
	}{
		{
			name: "no_providers",
			config: `
schema_version: "1.0"
default_provider: openai
`,
			expectedError: "at least one provider must be defined",
		},
		{
			name: "invalid_provider_type",
			config: `
schema_version: "1.0"
providers:
  test-provider:
    type: invalid-type
    model: test-model
    api_key: test-key
default_provider: test-provider
`,
			expectedError: "unsupported provider type",
		},
		{
			name: "missing_api_key",
			config: `
schema_version: "1.0"
providers:
  test-provider:
    type: openai
    model: gpt-4
    api_key: ""
default_provider: test-provider
`,
			expectedError: "API key must not be empty",
		},
		{
			name: "invalid_temperature",
			config: `
schema_version: "1.0"
providers:
  test-provider:
    type: openai
    model: gpt-4
    api_key: test-key
    parameters:
      temperature: 3.0
default_provider: test-provider
`,
			expectedError: "temperature must be between 0.0 and 2.0",
		},
		{
			name: "provider_not_found_in_role",
			config: `
schema_version: "1.0"
providers:
  test-provider:
    type: openai
    model: gpt-4
    api_key: test-key
roles:
  coder:
    provider: nonexistent-provider
default_provider: test-provider
`,
			expectedError: "provider 'nonexistent-provider' not found",
		},
		{
			name: "duplicate_fallback_provider",
			config: `
schema_version: "1.0"
providers:
  provider1:
    type: openai
    model: gpt-4
    api_key: test-key
  provider2:
    type: anthropic
    model: claude-3
    api_key: test-key
roles:
  coder:
    provider: provider1
    fallback:
      - provider2
      - provider2
default_provider: provider1
`,
			expectedError: "duplicate fallback provider",
		},
		{
			name: "critical_parameter_override",
			config: `
schema_version: "1.0"
providers:
  test-provider:
    type: openai
    model: gpt-4
    api_key: test-key
roles:
  coder:
    provider: test-provider
    parameters:
      api_key: override-key
default_provider: test-provider
`,
			expectedError: "cannot override critical parameter 'api_key'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			// Load configuration should fail
			_, err = providers.LoadConfig(configPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestValidateConfig_RetryConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		retry         *providers.RetryConfig
		expectedError string
	}{
		{
			name: "valid_retry",
			retry: &providers.RetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 1000000000,  // 1s in nanoseconds
				MaxBackoff:     30000000000, // 30s in nanoseconds
				Multiplier:     2.0,
			},
			expectedError: "",
		},
		{
			name: "max_attempts_too_high",
			retry: &providers.RetryConfig{
				MaxAttempts:    11,
				InitialBackoff: 1000000000,
				MaxBackoff:     30000000000,
				Multiplier:     2.0,
			},
			expectedError: "max_attempts must be between 1 and 10",
		},
		{
			name: "invalid_multiplier",
			retry: &providers.RetryConfig{
				MaxAttempts:    3,
				InitialBackoff: 1000000000,
				MaxBackoff:     30000000000,
				Multiplier:     0.5,
			},
			expectedError: "multiplier must be greater than 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &providers.MultiProviderConfig{
				Providers: map[string]*providers.ProviderConfig{
					"test": {
						ID:     "test",
						Type:   providers.ProviderTypeOpenAI,
						Model:  "gpt-4",
						APIKey: "test-key",
					},
				},
				DefaultProvider: "test",
				Retry:           tt.retry,
			}

			err := providers.ValidateConfig(config)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestMergeParameters(t *testing.T) {
	global := map[string]any{
		"temperature": 0.7,
		"max_tokens":  4096,
		"top_p":       0.9,
	}

	roleOverrides := map[string]any{
		"temperature": 0.8,  // Override
		"max_tokens":  8192, // Override
		// top_p not overridden
	}

	merged := providers.MergeParameters(global, roleOverrides)

	assert.Equal(t, 0.8, merged["temperature"], "temperature should be overridden")
	assert.Equal(t, 8192, merged["max_tokens"], "max_tokens should be overridden")
	assert.Equal(t, 0.9, merged["top_p"], "top_p should retain global value")
}
