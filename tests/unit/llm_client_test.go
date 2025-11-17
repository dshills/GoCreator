package unit

import (
	"context"
	"testing"
	"time"

	"github.com/dshills/gocreator/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_DefaultConfig verifies default configuration values
func TestConfig_DefaultConfig(t *testing.T) {
	cfg := llm.DefaultConfig()

	assert.Equal(t, llm.ProviderAnthropic, cfg.Provider)
	assert.Equal(t, "claude-sonnet-4", cfg.Model)
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
		config  llm.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config - anthropic",
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
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
			config: llm.Config{
				Provider:    llm.ProviderOpenAI,
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
			config: llm.Config{
				Provider:    llm.ProviderGoogle,
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
			config: llm.Config{
				Provider:    llm.Provider("invalid"),
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
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
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
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
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
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
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
		{
			name: "zero timeout",
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
				Temperature: 0.0,
				APIKey:      "test-key-1234567890",
				Timeout:     0,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "negative max tokens",
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
				Temperature: 0.0,
				APIKey:      "test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   -1,
				MaxRetries:  3,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "max tokens must be positive",
		},
		{
			name: "negative max retries",
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
				Temperature: 0.0,
				APIKey:      "test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  -1,
				RetryDelay:  time.Second,
			},
			wantErr: true,
			errMsg:  "max retries cannot be negative",
		},
		{
			name: "zero retry delay",
			config: llm.Config{
				Provider:    llm.ProviderAnthropic,
				Model:       "claude-sonnet-4",
				Temperature: 0.0,
				APIKey:      "test-key-1234567890",
				Timeout:     60 * time.Second,
				MaxTokens:   4096,
				MaxRetries:  3,
				RetryDelay:  0,
			},
			wantErr: true,
			errMsg:  "retry delay must be positive",
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
	cfg := llm.Config{
		Provider:    llm.ProviderAnthropic,
		Model:       "claude-sonnet-4",
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
	assert.Contains(t, str, "Model=claude-sonnet-4")
	assert.Contains(t, str, "Temperature=0.0")
}

// TestValidateAPIKey tests API key validation
func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider llm.Provider
		apiKey   string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid anthropic key",
			provider: llm.ProviderAnthropic,
			apiKey:   "sk-ant-api03-test-key-1234567890",
			wantErr:  false,
		},
		{
			name:     "valid openai key",
			provider: llm.ProviderOpenAI,
			apiKey:   "sk-test-key-1234567890",
			wantErr:  false,
		},
		{
			name:     "valid google key",
			provider: llm.ProviderGoogle,
			apiKey:   "AIzaSyTest-key-1234567890",
			wantErr:  false,
		},
		{
			name:     "too short key",
			provider: llm.ProviderAnthropic,
			apiKey:   "short",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
		{
			name:     "anthropic key too short",
			provider: llm.ProviderAnthropic,
			apiKey:   "sk-ant-123",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
		{
			name:     "openai key too short",
			provider: llm.ProviderOpenAI,
			apiKey:   "sk-123",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
		{
			name:     "google key too short",
			provider: llm.ProviderGoogle,
			apiKey:   "AIza123",
			wantErr:  true,
			errMsg:   "at least 20 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := llm.ValidateAPIKey(tt.provider, tt.apiKey)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestNewClient_InvalidConfig tests that NewClient rejects invalid configs
func TestNewClient_InvalidConfig(t *testing.T) {
	invalidConfig := llm.Config{
		Provider:    llm.ProviderAnthropic,
		Model:       "claude-sonnet-4",
		Temperature: 0.7, // Invalid: must be 0.0
		APIKey:      "test-key-1234567890",
		Timeout:     60 * time.Second,
		MaxTokens:   4096,
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}

	client, err := llm.NewClient(invalidConfig)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "temperature must be 0.0")
}

// TestMessage validates the Message struct
func TestMessage(t *testing.T) {
	tests := []struct {
		name    string
		message llm.Message
	}{
		{
			name: "system message",
			message: llm.Message{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
		},
		{
			name: "user message",
			message: llm.Message{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
		{
			name: "assistant message",
			message: llm.Message{
				Role:    "assistant",
				Content: "I'm doing well, thank you!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.message.Role)
			assert.NotEmpty(t, tt.message.Content)
		})
	}
}

// TestClient_Interface verifies that provider clients implement the Client interface
func TestClient_Interface(t *testing.T) {
	// This is a compile-time check to ensure all providers implement the interface
	// We can't instantiate real clients without valid API keys, so we just verify
	// the interface is properly defined

	var _ llm.Client = (*mockClient)(nil)
}

// mockClient is a test implementation of the Client interface
type mockClient struct {
	generateFunc           func(ctx context.Context, prompt string) (string, error)
	generateStructuredFunc func(ctx context.Context, prompt string, schema interface{}) (interface{}, error)
	chatFunc               func(ctx context.Context, messages []llm.Message) (string, error)
	provider               string
	model                  string
}

func (m *mockClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "mock response", nil
}

func (m *mockClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	if m.generateStructuredFunc != nil {
		return m.generateStructuredFunc(ctx, prompt, schema)
	}
	return map[string]interface{}{"result": "mock"}, nil
}

func (m *mockClient) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, messages)
	}
	return "mock chat response", nil
}

func (m *mockClient) Provider() string {
	return m.provider
}

func (m *mockClient) Model() string {
	return m.model
}

// TestMockClient_BasicOperations tests basic operations with mock client
func TestMockClient_BasicOperations(t *testing.T) {
	mock := &mockClient{
		provider: "test-provider",
		model:    "test-model",
	}

	ctx := context.Background()

	t.Run("Generate", func(t *testing.T) {
		result, err := mock.Generate(ctx, "test prompt")
		require.NoError(t, err)
		assert.Equal(t, "mock response", result)
	})

	t.Run("GenerateStructured", func(t *testing.T) {
		schema := map[string]interface{}{"type": "object"}
		result, err := mock.GenerateStructured(ctx, "test prompt", schema)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Chat", func(t *testing.T) {
		messages := []llm.Message{
			{Role: "user", Content: "Hello"},
		}
		result, err := mock.Chat(ctx, messages)
		require.NoError(t, err)
		assert.Equal(t, "mock chat response", result)
	})

	t.Run("Provider", func(t *testing.T) {
		assert.Equal(t, "test-provider", mock.Provider())
	})

	t.Run("Model", func(t *testing.T) {
		assert.Equal(t, "test-model", mock.Model())
	})
}

// TestMockClient_ErrorHandling tests error handling with mock client
func TestMockClient_ErrorHandling(t *testing.T) {
	testErr := assert.AnError

	mock := &mockClient{
		generateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", testErr
		},
		generateStructuredFunc: func(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
			return nil, testErr
		},
		chatFunc: func(ctx context.Context, messages []llm.Message) (string, error) {
			return "", testErr
		},
	}

	ctx := context.Background()

	t.Run("Generate error", func(t *testing.T) {
		_, err := mock.Generate(ctx, "test")
		require.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("GenerateStructured error", func(t *testing.T) {
		_, err := mock.GenerateStructured(ctx, "test", nil)
		require.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("Chat error", func(t *testing.T) {
		_, err := mock.Chat(ctx, []llm.Message{{Role: "user", Content: "test"}})
		require.Error(t, err)
		assert.Equal(t, testErr, err)
	})
}

// TestMockClient_ContextCancellation tests context cancellation handling
func TestMockClient_ContextCancellation(t *testing.T) {
	mock := &mockClient{
		generateFunc: func(ctx context.Context, prompt string) (string, error) {
			// Simulate checking context
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "result", nil
		},
	}

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := mock.Generate(ctx, "test")
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure timeout

		_, err := mock.Generate(ctx, "test")
		require.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

// TestProviderConstants verifies provider constants
func TestProviderConstants(t *testing.T) {
	assert.Equal(t, llm.Provider("anthropic"), llm.ProviderAnthropic)
	assert.Equal(t, llm.Provider("openai"), llm.ProviderOpenAI)
	assert.Equal(t, llm.Provider("google"), llm.ProviderGoogle)
}
