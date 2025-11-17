package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

// anthropicClient implements the Client interface for Anthropic (Claude)
type anthropicClient struct {
	baseClient
	llm *anthropic.LLM
}

// newAnthropicClient creates a new Anthropic client
func newAnthropicClient(config Config) (*anthropicClient, error) {
	// Create langchaingo Anthropic LLM with options
	llm, err := anthropic.New(
		anthropic.WithToken(config.APIKey),
		anthropic.WithModel(config.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	return &anthropicClient{
		baseClient: baseClient{config: config},
		llm:        llm,
	}, nil
}

// Generate produces text from a single prompt
func (c *anthropicClient) Generate(ctx context.Context, prompt string) (string, error) {
	var result string

	// Execute with retry logic
	err := c.retry(ctx, "generate", func() error {
		// Call LLM with enforced temperature of 0.0
		resp, err := c.llm.Call(ctx, prompt,
			llms.WithTemperature(c.config.Temperature), // MUST be 0.0
			llms.WithMaxTokens(c.config.MaxTokens),
		)
		if err != nil {
			return err
		}

		result = resp
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
	// and request JSON formatted response
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
		// Call LLM with enforced temperature of 0.0
		resp, err := c.llm.Call(ctx, structuredPrompt,
			llms.WithTemperature(c.config.Temperature), // MUST be 0.0
			llms.WithMaxTokens(c.config.MaxTokens),
		)
		if err != nil {
			return err
		}

		result = resp
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

	// Convert messages to langchaingo format
	var msgContents []llms.MessageContent
	for _, msg := range messages {
		var chatMsgType llms.ChatMessageType
		switch msg.Role {
		case "system":
			chatMsgType = llms.ChatMessageTypeSystem
		case "user":
			chatMsgType = llms.ChatMessageTypeHuman
		case "assistant":
			chatMsgType = llms.ChatMessageTypeAI
		default:
			return "", c.wrapError("chat", fmt.Errorf("invalid message role: %s", msg.Role))
		}

		msgContents = append(msgContents, llms.MessageContent{
			Role: chatMsgType,
			Parts: []llms.ContentPart{
				llms.TextPart(msg.Content),
			},
		})
	}

	var result string

	// Execute with retry logic
	err := c.retry(ctx, "chat", func() error {
		// Call LLM with enforced temperature of 0.0
		resp, err := c.llm.GenerateContent(ctx, msgContents,
			llms.WithTemperature(c.config.Temperature), // MUST be 0.0
			llms.WithMaxTokens(c.config.MaxTokens),
		)
		if err != nil {
			return err
		}

		// Extract text from response
		if len(resp.Choices) == 0 {
			return fmt.Errorf("no response choices returned")
		}

		if resp.Choices[0].Content == "" {
			return fmt.Errorf("empty response content")
		}

		result = resp.Choices[0].Content
		return nil
	})

	if err != nil {
		return "", c.wrapError("chat", err)
	}

	return result, nil
}
