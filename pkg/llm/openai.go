package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// openaiClient implements the Client interface for OpenAI (GPT)
type openaiClient struct {
	baseClient
	llm *openai.LLM
}

// newOpenAIClient creates a new OpenAI client
func newOpenAIClient(config Config) (*openaiClient, error) {
	// Create langchaingo OpenAI LLM with options
	llm, err := openai.New(
		openai.WithToken(config.APIKey),
		openai.WithModel(config.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &openaiClient{
		baseClient: baseClient{config: config},
		llm:        llm,
	}, nil
}

// Generate produces text from a single prompt
func (c *openaiClient) Generate(ctx context.Context, prompt string) (string, error) {
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
func (c *openaiClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
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
			llms.WithJSONMode(), // Enable JSON mode for OpenAI
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
func (c *openaiClient) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", c.wrapError("chat", fmt.Errorf("messages cannot be empty"))
	}

	// Convert messages to langchaingo format
	msgContents := make([]llms.MessageContent, 0, len(messages))
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
