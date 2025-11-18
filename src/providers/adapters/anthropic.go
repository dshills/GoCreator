// Package adapters provides LLM provider adapter implementations.
package adapters

import (
	"context"
	"log/slog"
	"time"

	"github.com/dshills/gocreator/src/providers"
	"github.com/dshills/langgraph-go/graph/model"
	"github.com/dshills/langgraph-go/graph/model/anthropic"
)

//nolint:gochecknoinits // Init function required for provider registration pattern
func init() {
	providers.RegisterProviderFactory(providers.ProviderTypeAnthropic, func(id string, config *providers.ProviderConfig, retryConfig *providers.RetryConfig) (providers.LLMProvider, error) {
		return NewAnthropicAdapter(id, config, retryConfig), nil
	})
}

// AnthropicAdapter implements the LLMProvider interface for Anthropic Claude
type AnthropicAdapter struct {
	id          string
	config      *providers.ProviderConfig
	retryConfig *providers.RetryConfig
	chatModel   *anthropic.ChatModel
}

// NewAnthropicAdapter creates a new Anthropic adapter
func NewAnthropicAdapter(id string, config *providers.ProviderConfig, retryConfig *providers.RetryConfig) *AnthropicAdapter {
	return &AnthropicAdapter{
		id:          id,
		config:      config,
		retryConfig: retryConfig,
	}
}

// Initialize validates credentials and prepares the provider
func (a *AnthropicAdapter) Initialize(ctx context.Context) error {
	// Create Anthropic ChatModel
	chatModel := anthropic.NewChatModel(a.config.APIKey, a.config.Model)
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

// Execute sends a request to the Anthropic provider with retry logic
func (a *AnthropicAdapter) Execute(ctx context.Context, req providers.Request) (providers.Response, error) {
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
func (a *AnthropicAdapter) Name() string {
	return a.id
}

// Type returns the provider type
func (a *AnthropicAdapter) Type() providers.ProviderType {
	return providers.ProviderTypeAnthropic
}

// Shutdown gracefully closes resources
func (a *AnthropicAdapter) Shutdown(_ context.Context) error {
	// Anthropic client doesn't require explicit shutdown
	return nil
}
