package llm_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dshills/gocreator/pkg/llm"
)

// ExampleNewClient_anthropic demonstrates creating an Anthropic (Claude) client
func ExampleNewClient_anthropic() {
	config := llm.Config{
		Provider:    llm.ProviderAnthropic,
		Model:       "claude-sonnet-4-5",
		Temperature: 0.0, // MUST be 0.0 for deterministic output
		APIKey:      "sk-ant-api03-your-api-key-here",
		Timeout:     120 * time.Second,
		MaxTokens:   4096,
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created %s client using model %s\n", client.Provider(), client.Model())
	// Output would be: Created anthropic client using model claude-sonnet-4
}

// ExampleNewClient_openai demonstrates creating an OpenAI (GPT) client
func ExampleNewClient_openai() {
	config := llm.Config{
		Provider:    llm.ProviderOpenAI,
		Model:       "gpt-4",
		Temperature: 0.0, // MUST be 0.0 for deterministic output
		APIKey:      "sk-your-openai-api-key-here",
		Timeout:     120 * time.Second,
		MaxTokens:   4096,
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created %s client using model %s\n", client.Provider(), client.Model())
	// Output would be: Created openai client using model gpt-4
}

// ExampleNewClient_google demonstrates creating a Google (Gemini) client
func ExampleNewClient_google() {
	config := llm.Config{
		Provider:    llm.ProviderGoogle,
		Model:       "gemini-pro",
		Temperature: 0.0, // MUST be 0.0 for deterministic output
		APIKey:      "AIzaSy-your-google-api-key-here",
		Timeout:     120 * time.Second,
		MaxTokens:   4096,
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created %s client using model %s\n", client.Provider(), client.Model())
	// Output would be: Created google client using model gemini-pro
}

// ExampleClient_Generate demonstrates simple text generation
func ExampleClient_Generate() {
	// Use default config and customize as needed
	config := llm.DefaultConfig()
	config.APIKey = "your-api-key-here"

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	prompt := "Explain what a REST API is in one sentence."

	response, err := client.Generate(ctx, prompt)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
	// Output would contain a single-sentence explanation of REST APIs
}

// ExampleClient_Chat demonstrates multi-turn conversation
func ExampleClient_Chat() {
	config := llm.DefaultConfig()
	config.APIKey = "your-api-key-here"

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	messages := []llm.Message{
		{
			Role:    "system",
			Content: "You are a helpful coding assistant.",
		},
		{
			Role:    "user",
			Content: "What is the difference between a slice and an array in Go?",
		},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)
	// Output would contain an explanation of Go slices vs arrays
}

// ExampleClient_GenerateStructured demonstrates structured JSON output
func ExampleClient_GenerateStructured() {
	config := llm.DefaultConfig()
	config.APIKey = "your-api-key-here"

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	prompt := "Generate information about the Go programming language."

	// Define expected schema
	schema := map[string]interface{}{
		"name":        "string",
		"year":        "number",
		"creator":     "string",
		"description": "string",
		"is_compiled": "boolean",
	}

	response, err := client.GenerateStructured(ctx, prompt, schema)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Received structured data: %T\n", response)
	// Output would be: Received structured data: map[string]interface {}
}

// ExampleDefaultConfig demonstrates using default configuration
func ExampleDefaultConfig() {
	config := llm.DefaultConfig()

	fmt.Printf("Provider: %s\n", config.Provider)
	fmt.Printf("Model: %s\n", config.Model)
	fmt.Printf("Temperature: %.1f\n", config.Temperature)
	fmt.Printf("Timeout: %v\n", config.Timeout)
	fmt.Printf("MaxTokens: %d\n", config.MaxTokens)
	fmt.Printf("MaxRetries: %d\n", config.MaxRetries)

	// Output:
	// Provider: anthropic
	// Model: claude-sonnet-4
	// Temperature: 0.0
	// Timeout: 2m0s
	// MaxTokens: 4096
	// MaxRetries: 3
}

// ExampleConfig_Validate demonstrates configuration validation
func ExampleConfig_Validate() {
	// Invalid config - temperature must be 0.0
	invalidConfig := llm.Config{
		Provider:    llm.ProviderAnthropic,
		Model:       "claude-sonnet-4-5",
		Temperature: 0.7, // Invalid!
		APIKey:      "sk-ant-api03-test-key-1234567890",
		Timeout:     60 * time.Second,
		MaxTokens:   4096,
		MaxRetries:  3,
		RetryDelay:  time.Second,
	}

	if err := invalidConfig.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
	}

	// Valid config
	validConfig := invalidConfig
	validConfig.Temperature = 0.0

	if err := validConfig.Validate(); err != nil {
		fmt.Printf("Validation error: %v\n", err)
	} else {
		fmt.Println("Config is valid")
	}

	// Output:
	// Validation error: temperature must be 0.0 for deterministic output, got: 0.700000
	// Config is valid
}

// ExampleConfig_String demonstrates safe config printing
func ExampleConfig_String() {
	config := llm.Config{
		Provider:    llm.ProviderAnthropic,
		Model:       "claude-sonnet-4-5",
		Temperature: 0.0,
		APIKey:      "sk-ant-api03-very-secret-key-that-should-not-be-shown",
		Timeout:     60 * time.Second,
		MaxTokens:   4096,
	}

	// API key is masked in string representation
	fmt.Println(config.String())
	// Output will contain masked API key: sk-a***
}

// ExampleValidateAPIKey demonstrates API key validation
func ExampleValidateAPIKey() {
	// Valid API key
	err := llm.ValidateAPIKey(llm.ProviderAnthropic, "sk-ant-api03-valid-key-1234567890-abcdef")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Valid Anthropic API key")
	}

	// Invalid API key (too short)
	err = llm.ValidateAPIKey(llm.ProviderOpenAI, "sk-short")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Output:
	// Valid Anthropic API key
	// Error: OpenAI API key should be at least 20 characters
}
