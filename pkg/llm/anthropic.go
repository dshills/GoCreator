// Package llm provides LLM client interfaces and implementations.
package llm

import (
	"context"
	"encoding/json"
	"fmt"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/dshills/langgraph-go/graph/model"
	"github.com/dshills/langgraph-go/graph/model/anthropic"
)

// anthropicClient implements the Client interface for Anthropic (Claude)
type anthropicClient struct {
	baseClient
	chatModel    *anthropic.ChatModel
	directClient anthropicsdk.Client // Direct SDK client for cache support
	cacheMetrics PromptCacheMetrics  // Track prompt cache usage
}

// newAnthropicClient creates a new Anthropic client
func newAnthropicClient(config Config) (*anthropicClient, error) {
	// Create langgraph-go Anthropic ChatModel
	chatModel := anthropic.NewChatModel(config.APIKey, config.Model)

	// Create direct Anthropic SDK client for cache support
	directClient := anthropicsdk.NewClient(
		option.WithAPIKey(config.APIKey),
	)

	return &anthropicClient{
		baseClient:   baseClient{config: config},
		chatModel:    chatModel,
		directClient: directClient,
		cacheMetrics: PromptCacheMetrics{},
	}, nil
}

// Generate produces text from a single prompt
func (c *anthropicClient) Generate(ctx context.Context, prompt string) (string, error) {
	var result string

	// Execute with retry logic
	err := c.retry(ctx, "generate", func() error {
		// Create messages for the chat
		messages := []model.Message{
			{Role: model.RoleUser, Content: prompt},
		}

		// Call ChatModel
		out, err := c.chatModel.Chat(ctx, messages, nil)
		if err != nil {
			return err
		}

		result = out.Text
		return nil
	})

	if err != nil {
		return "", c.wrapError("generate", err)
	}

	return result, nil
}

// GenerateStructured produces structured output based on a schema
func (c *anthropicClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	// For structured output, we append schema information to the prompt
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, c.wrapError("generate_structured", fmt.Errorf("failed to marshal schema: %w", err))
	}

	structuredPrompt := fmt.Sprintf(`%s

Please respond with valid JSON that matches this schema:
%s

Return ONLY the JSON, with no additional text or explanation.`, prompt, schemaJSON)

	var result string

	// Execute with retry logic
	err = c.retry(ctx, "generate_structured", func() error {
		// Create messages for the chat
		messages := []model.Message{
			{Role: model.RoleUser, Content: structuredPrompt},
		}

		// Call ChatModel
		out, err := c.chatModel.Chat(ctx, messages, nil)
		if err != nil {
			return err
		}

		result = out.Text
		return nil
	})

	if err != nil {
		return nil, c.wrapError("generate_structured", err)
	}

	// Parse the JSON response
	var output interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return nil, c.wrapError("generate_structured", fmt.Errorf("failed to parse JSON response: %w", err))
	}

	return output, nil
}

// Chat processes a sequence of messages and returns the assistant's response
func (c *anthropicClient) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", c.wrapError("chat", fmt.Errorf("messages cannot be empty"))
	}

	// Convert messages to langgraph-go format
	modelMessages := make([]model.Message, 0, len(messages))
	for _, msg := range messages {
		var msgRole string
		switch msg.Role {
		case "system":
			msgRole = model.RoleSystem
		case "user":
			msgRole = model.RoleUser
		case "assistant":
			msgRole = model.RoleAssistant
		default:
			return "", c.wrapError("chat", fmt.Errorf("invalid message role: %s", msg.Role))
		}

		modelMessages = append(modelMessages, model.Message{
			Role:    msgRole,
			Content: msg.Content,
		})
	}

	var result string

	// Execute with retry logic
	err := c.retry(ctx, "chat", func() error {
		// Call ChatModel
		out, err := c.chatModel.Chat(ctx, modelMessages, nil)
		if err != nil {
			return err
		}

		result = out.Text
		return nil
	})

	if err != nil {
		return "", c.wrapError("chat", err)
	}

	return result, nil
}

// GenerateWithCache generates text using cacheable messages for Anthropic prompt caching
// This method uses the Anthropic SDK directly to support cache_control
func (c *anthropicClient) GenerateWithCache(ctx context.Context, messages []CacheableMessage) (string, error) {
	if !c.config.EnableCaching {
		// Fall back to regular generation if caching is disabled
		regularMessages := make([]Message, len(messages))
		for i, msg := range messages {
			regularMessages[i] = Message{Role: msg.Role, Content: msg.Content}
		}
		return c.Chat(ctx, regularMessages)
	}

	// Build Anthropic SDK message request with cache support
	var systemBlocks []anthropicsdk.TextBlockParam
	var userMessages []anthropicsdk.MessageParam

	for _, msg := range messages {
		if msg.Role == "system" {
			// Create text block parameter
			textBlockParam := anthropicsdk.TextBlockParam{
				Text: msg.Content,
			}

			// Add cache control if specified
			if msg.Cache != nil {
				cacheCtrl := anthropicsdk.NewCacheControlEphemeralParam()
				if msg.Cache.TTL == "1h" {
					cacheCtrl.TTL = anthropicsdk.CacheControlEphemeralTTLTTL1h
				} else {
					cacheCtrl.TTL = anthropicsdk.CacheControlEphemeralTTLTTL5m
				}
				textBlockParam.CacheControl = cacheCtrl
			}
			systemBlocks = append(systemBlocks, textBlockParam)
		} else {
			// User or assistant messages
			textBlock := anthropicsdk.NewTextBlock(msg.Content)

			// Build message parameter
			userMsg := anthropicsdk.NewUserMessage(textBlock)
			if msg.Role == "assistant" {
				userMsg = anthropicsdk.NewAssistantMessage(textBlock)
			}

			userMessages = append(userMessages, userMsg)
		}
	}

	var result string

	// Execute with retry logic
	err := c.retry(ctx, "generate_with_cache", func() error {
		// Create message request
		params := anthropicsdk.MessageNewParams{
			Model:     anthropicsdk.Model(c.config.Model),
			MaxTokens: int64(c.config.MaxTokens),
			Messages:  userMessages,
		}

		// Add system blocks if present
		if len(systemBlocks) > 0 {
			params.System = systemBlocks
		}

		// Call Anthropic API with cache support
		response, err := c.directClient.Messages.New(ctx, params)
		if err != nil {
			return err
		}

		// Extract text from response
		if len(response.Content) > 0 {
			// ContentBlockUnion has direct Text field when Type is "text"
			if response.Content[0].Type == "text" {
				result = response.Content[0].Text
			}
		}

		// Update cache metrics from usage
		if response.Usage.CacheCreationInputTokens > 0 {
			c.cacheMetrics.CacheCreationTokens += int64(response.Usage.CacheCreationInputTokens)
			c.cacheMetrics.CacheMisses++
		}
		if response.Usage.CacheReadInputTokens > 0 {
			c.cacheMetrics.CacheReadTokens += int64(response.Usage.CacheReadInputTokens)
			c.cacheMetrics.CacheHits++
		}
		if response.Usage.InputTokens > 0 {
			c.cacheMetrics.InputTokens += int64(response.Usage.InputTokens)
		}
		if response.Usage.OutputTokens > 0 {
			c.cacheMetrics.OutputTokens += int64(response.Usage.OutputTokens)
		}

		return nil
	})

	if err != nil {
		return "", c.wrapError("generate_with_cache", err)
	}

	return result, nil
}

// GetCacheMetrics returns the current prompt cache metrics
func (c *anthropicClient) GetCacheMetrics() PromptCacheMetrics {
	return c.cacheMetrics
}

// ResetCacheMetrics resets the cache metrics counters
func (c *anthropicClient) ResetCacheMetrics() {
	c.cacheMetrics = PromptCacheMetrics{}
}
