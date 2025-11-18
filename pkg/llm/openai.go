package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dshills/langgraph-go/graph/model"
	"github.com/dshills/langgraph-go/graph/model/openai"
)

// openaiClient implements the Client interface for OpenAI (GPT)
type openaiClient struct {
	baseClient
	chatModel *openai.ChatModel
}

// newOpenAIClient creates a new OpenAI client
func newOpenAIClient(config Config) (*openaiClient, error) {
	// Create langgraph-go OpenAI ChatModel
	chatModel := openai.NewChatModel(config.APIKey, config.Model)

	return &openaiClient{
		baseClient: baseClient{config: config},
		chatModel:  chatModel,
	}, nil
}

// Generate produces text from a single prompt
func (c *openaiClient) Generate(ctx context.Context, prompt string) (string, error) {
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
func (c *openaiClient) Chat(ctx context.Context, messages []Message) (string, error) {
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
