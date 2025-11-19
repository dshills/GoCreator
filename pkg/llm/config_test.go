package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_DefaultConfig verifies default configuration values
func TestConfig_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, ProviderAnthropic, cfg.Provider)
	assert.Equal(t, "claude-sonnet-4-5", cfg.Model)
	assert.Equal(t, 0.0, cfg.Temperature, "Temperature must be 0.0 for determinism")
	assert.Equal(t, 120*time.Second, cfg.Timeout)
	assert.Equal(t, 4096, cfg.MaxTokens)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.RetryDelay)
}

// TestConfig_Validate tests configuration validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config - anthropic",
			config: Config{
				Provider:    ProviderAnthropic,
				Model:       "claude-sonnet-4-5",
				Temperature: 0.0,
				APIKey:      "sk-ant-api03-test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config - openai",
			config: Config{
				Provider:    ProviderOpenAI,
				Model:       "gpt-4",
				Temperature: 0.0,
				APIKey:      "sk-test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config - google",
			config: Config{
				Provider:    ProviderGoogle,
				Model:       "gemini-pro",
				Temperature: 0.0,
				APIKey:      "AIzaSyTest-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			config: Config{
				Provider:    Provider("invalid"),
				Model:       "test-model",
				Temperature: 0.0,
				APIKey:      "test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "invalid provider",
		},
		{
			name: "empty model",
			config: Config{
				Provider:    ProviderAnthropic,
				Model:       "",
				Temperature: 0.0,
				APIKey:      "test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "model name cannot be empty",
		},
		{
			name: "non-zero temperature",
			config: Config{
				Provider:    ProviderAnthropic,
				Model:       "claude-sonnet-4-5",
				Temperature: 0.7, // MUST be 0.0
				APIKey:      "test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "temperature must be 0.0",
		},
		{
			name: "empty API key",
			config: Config{
				Provider:    ProviderAnthropic,
				Model:       "claude-sonnet-4-5",
				Temperature: 0.0,
				APIKey:      "",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "API key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfig_String verifies that sensitive data is masked
func TestConfig_String(t *testing.T) {
	cfg := Config{
		Provider:    ProviderAnthropic,
		Model:       "claude-sonnet-4-5",
		Temperature: 0.0,
		APIKey:      "sk-ant-api03-very-secret-key-1234567890",
		Timeout:     60 * time.Second,
		MaxTokens:   4096,
	}

	str := cfg.String()

	// Verify API key is masked
	assert.NotContains(t, str, "very-secret-key")
	assert.Contains(t, str, "***")
	assert.Contains(t, str, "Provider=anthropic")
	assert.Contains(t, str, "Model=claude-sonnet-4-5")
	assert.Contains(t, str, "Temperature=0.0")
}

// TestValidateAPIKey tests API key validation
func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		apiKey   string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid anthropic key",
			provider: ProviderAnthropic,
			apiKey:   "sk-ant-api03-test-key-1234567890",
			wantErr:  false,
		},
		{
			name:     "valid openai key",
			provider: ProviderOpenAI,
			apiKey:   "sk-test-key-1234567890",
			wantErr:  false,
		},
		{
			name:     "valid google key",
			provider: ProviderGoogle,
			apiKey:   "AIzaSyTest-key-1234567890",
			wantErr:  false,
		},
		{
			name:     "anthropic key too short",
			provider: ProviderAnthropic,
			apiKey:   "sk-ant-123",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
		{
			name:     "openai key too short",
			provider: ProviderOpenAI,
			apiKey:   "sk-123",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
		{
			name:     "google key too short",
			provider: ProviderGoogle,
			apiKey:   "AIza123",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.provider, tt.apiKey)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestProviderConstants verifies provider constants
func TestProviderConstants(t *testing.T) {
	assert.Equal(t, Provider("anthropic"), ProviderAnthropic)
	assert.Equal(t, Provider("openai"), ProviderOpenAI)
	assert.Equal(t, Provider("google"), ProviderGoogle)
}
