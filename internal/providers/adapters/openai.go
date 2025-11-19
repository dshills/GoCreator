// Package adapters provides LLM provider adapter implementations.
package adapters

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dshills/gocreator/internal/providers"
	"github.com/dshills/langgraph-go/graph/model"
	"github.com/dshills/langgraph-go/graph/model/openai"
)

//nolint:gochecknoinits // Init function required for provider registration pattern
func init() {
	providers.RegisterProviderFactory(providers.ProviderTypeOpenAI, func(id string, config *providers.ProviderConfig, retryConfig *providers.RetryConfig) (providers.LLMProvider, error) {
		return NewOpenAIAdapter(id, config, retryConfig), nil
	})
}

// OpenAIAdapter implements the LLMProvider interface for OpenAI
type OpenAIAdapter struct {
	id          string
	config      *providers.ProviderConfig
	retryConfig *providers.RetryConfig
	chatModel   *openai.ChatModel
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(id string, config *providers.ProviderConfig, retryConfig *providers.RetryConfig) *OpenAIAdapter {
	return &OpenAIAdapter{
		id:          id,
		config:      config,
		retryConfig: retryConfig,
	}
}

// Initialize validates credentials and prepares the provider
func (a *OpenAIAdapter) Initialize(ctx context.Context) error {
	// Create OpenAI ChatModel
	chatModel := openai.NewChatModel(a.config.APIKey, a.config.Model)
	a.chatModel = chatModel

	// Test credentials with a simple message
	messages := []model.Message{
		{Role: model.RoleUser, Content: "test"},
	}

	_, err := a.chatModel.Chat(ctx, messages, nil)
	if err != nil {
		return providers.NewProviderError(a.id, providers.ErrorCodeAuth, "credential validation failed", err)
	}

	return nil
}

// Execute sends a request to the OpenAI provider with retry logic
func (a *OpenAIAdapter) Execute(ctx context.Context, req providers.Request) (providers.Response, error) {
	var resp providers.Response
	startTime := time.Now()

	slog.Info("Provider request started",
		"provider_id", a.id,
		"role", req.Role,
		"model", a.config.Model,
		"max_tokens", req.MaxTokens,
		"temperature", req.Temperature,
		"prompt_length", len(req.Prompt),
	)

	err := a.retryConfig.Execute(ctx, func() error {
		// Create messages for the chat
		messages := []model.Message{
			{Role: model.RoleUser, Content: req.Prompt},
		}

		// Execute the LLM call
		out, err := a.chatModel.Chat(ctx, messages, nil)
		if err != nil {
			// Classify error for retry logic
			if isRetryableError(err) {
				slog.Warn("Provider request retryable error",
					"provider_id", a.id,
					"role", req.Role,
					"error", err.Error(),
				)
				return err // Will retry
			}
			return providers.NewProviderError(a.id, classifyError(err), err.Error(), err)
		}

		// Build response
		resp = providers.Response{
			Content:        out.Text,
			Model:          a.config.Model,
			TokensPrompt:   estimateTokens(req.Prompt),
			TokensResponse: estimateTokens(out.Text),
		}

		return nil
	})

	duration := time.Since(startTime)

	if err != nil {
		slog.Error("Provider request failed",
			"provider_id", a.id,
			"role", req.Role,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		resp.Error = err
		return resp, err
	}

	slog.Info("Provider request completed",
		"provider_id", a.id,
		"role", req.Role,
		"model", a.config.Model,
		"duration_ms", duration.Milliseconds(),
		"tokens_prompt", resp.TokensPrompt,
		"tokens_response", resp.TokensResponse,
		"response_length", len(resp.Content),
	)

	return resp, nil
}

// Name returns the provider identifier
func (a *OpenAIAdapter) Name() string {
	return a.id
}

// Type returns the provider type
func (a *OpenAIAdapter) Type() providers.ProviderType {
	return providers.ProviderTypeOpenAI
}

// Shutdown gracefully closes resources
func (a *OpenAIAdapter) Shutdown(_ context.Context) error {
	// OpenAI client doesn't require explicit shutdown
	return nil
}

// Helper functions

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	code := classifyError(err)
	switch code {
	case providers.ErrorCodeRateLimit, providers.ErrorCodeNetwork, providers.ErrorCodeTimeout, providers.ErrorCodeServerError:
		return true
	case providers.ErrorCodeAuth, providers.ErrorCodeInvalidInput:
		return false
	default:
		// For unknown errors, be conservative and don't retry
		return false
	}
}

func classifyError(err error) providers.ErrorCode {
	if err == nil {
		return providers.ErrorCodeUnknown
	}

	errMsg := err.Error()

	// Check for authentication errors
	if contains(errMsg, "401", "unauthorized", "invalid_api_key", "authentication", "api key") {
		return providers.ErrorCodeAuth
	}

	// Check for rate limit errors
	if contains(errMsg, "429", "rate_limit", "quota", "too many requests") {
		return providers.ErrorCodeRateLimit
	}

	// Check for invalid input errors
	if contains(errMsg, "400", "invalid_request", "bad request", "invalid input", "validation") {
		return providers.ErrorCodeInvalidInput
	}

	// Check for server errors
	if contains(errMsg, "500", "502", "503", "504", "internal_error", "server_error", "service unavailable") {
		return providers.ErrorCodeServerError
	}

	// Check for timeout errors
	if contains(errMsg, "timeout", "deadline exceeded", "context deadline") {
		return providers.ErrorCodeTimeout
	}

	// Check for network errors
	if contains(errMsg, "connection refused", "connection reset", "no such host", "network", "EOF", "broken pipe") {
		return providers.ErrorCodeNetwork
	}

	// Default to unknown
	return providers.ErrorCodeUnknown
}

// contains checks if any of the substrings exist in the string (case-insensitive)
func contains(s string, substrings ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range substrings {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token
	// In production, use tiktoken library for accurate counts
	return len(text) / 4
}
