# LLM Provider Wrapper

A unified interface for multiple LLM providers (Anthropic, OpenAI, Google) with deterministic output guarantees.

## Features

- **Multi-provider support**: Anthropic (Claude), OpenAI (GPT), Google (Gemini)
- **Deterministic output**: Temperature locked at 0.0 for reproducible results
- **Retry logic**: Exponential backoff with configurable retry attempts
- **Timeout handling**: Context-aware timeout support
- **Error wrapping**: Provider-specific error context
- **Thread-safe**: Client instances can be safely reused across goroutines

## Installation

The package uses `github.com/tmc/langchaingo` for LLM abstractions:

```bash
go get github.com/tmc/langchaingo
```

## Quick Start

### Using Default Configuration

```go
package main

import (
    "context"
    "log"

    "github.com/dshills/gocreator/pkg/llm"
)

func main() {
    // Start with defaults
    config := llm.DefaultConfig()
    config.APIKey = "your-api-key-here"

    client, err := llm.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    response, err := client.Generate(ctx, "Explain Go interfaces in one sentence.")
    if err != nil {
        log.Fatal(err)
    }

    log.Println(response)
}
```

### Anthropic (Claude) Client

```go
config := llm.Config{
    Provider:    llm.ProviderAnthropic,
    Model:       "claude-sonnet-4-5",
    Temperature: 0.0, // MUST be 0.0
    APIKey:      "sk-ant-api03-your-key",
    Timeout:     120 * time.Second,
    MaxTokens:   4096,
    MaxRetries:  3,
    RetryDelay:  time.Second,
}

client, err := llm.NewClient(config)
```

### OpenAI (GPT) Client

```go
config := llm.Config{
    Provider:    llm.ProviderOpenAI,
    Model:       "gpt-4",
    Temperature: 0.0, // MUST be 0.0
    APIKey:      "sk-your-openai-key",
    Timeout:     120 * time.Second,
    MaxTokens:   4096,
    MaxRetries:  3,
    RetryDelay:  time.Second,
}

client, err := llm.NewClient(config)
```

### Google (Gemini) Client

```go
config := llm.Config{
    Provider:    llm.ProviderGoogle,
    Model:       "gemini-pro",
    Temperature: 0.0, // MUST be 0.0
    APIKey:      "AIzaSy-your-google-key",
    Timeout:     120 * time.Second,
    MaxTokens:   4096,
    MaxRetries:  3,
    RetryDelay:  time.Second,
}

client, err := llm.NewClient(config)
```

## Usage Examples

### Simple Generation

```go
ctx := context.Background()
prompt := "Explain what a REST API is in one sentence."

response, err := client.Generate(ctx, prompt)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response)
```

### Chat (Multi-turn Conversation)

```go
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
```

### Structured Output (JSON)

```go
ctx := context.Background()
prompt := "Generate information about the Go programming language."

// Define expected schema
schema := map[string]interface{}{
    "name":        "string",
    "year":        "number",
    "creator":     "string",
    "description": "string",
}

response, err := client.GenerateStructured(ctx, prompt, schema)
if err != nil {
    log.Fatal(err)
}

// response is a map[string]interface{} matching the schema
data := response.(map[string]interface{})
fmt.Printf("Language: %s, Year: %.0f\n", data["name"], data["year"])
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.Generate(ctx, "Long prompt...")
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Request timed out")
    } else {
        log.Fatal(err)
    }
}
```

## Configuration

### Config Structure

```go
type Config struct {
    Provider    Provider      // anthropic, openai, google
    Model       string        // Model name
    Temperature float64       // MUST be 0.0 for determinism
    APIKey      string        // Authentication key
    Timeout     time.Duration // Max duration for API calls
    MaxTokens   int           // Max tokens to generate
    MaxRetries  int           // Max retry attempts
    RetryDelay  time.Duration // Initial retry delay (exponential backoff)
}
```

### Default Values

```go
llm.DefaultConfig() returns:
- Provider: anthropic
- Model: claude-sonnet-4
- Temperature: 0.0 (enforced)
- Timeout: 120 seconds
- MaxTokens: 4096
- MaxRetries: 3
- RetryDelay: 1 second
```

### Configuration Validation

The package enforces strict validation:

- **Temperature MUST be 0.0** (for deterministic output)
- Provider must be one of: anthropic, openai, google
- Model name cannot be empty
- API key must be at least 20 characters
- Timeout must be positive
- MaxTokens must be positive
- MaxRetries cannot be negative
- RetryDelay must be positive

```go
if err := config.Validate(); err != nil {
    log.Fatal("Invalid config:", err)
}
```

## Client Interface

All provider implementations satisfy this interface:

```go
type Client interface {
    Generate(ctx context.Context, prompt string) (string, error)
    GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error)
    Chat(ctx context.Context, messages []Message) (string, error)
    Provider() string
    Model() string
}
```

## Error Handling

Errors are wrapped with provider and operation context:

```go
response, err := client.Generate(ctx, prompt)
if err != nil {
    // Error format: llm[provider:model] operation: underlying error
    log.Printf("LLM error: %v", err)
}
```

### Retry Behavior

- Automatic retry with exponential backoff
- Configurable max retry attempts (default: 3)
- Respects context cancellation during retries
- Logs retry attempts at WARN level

## Thread Safety

Client instances are thread-safe and can be reused:

```go
client, _ := llm.NewClient(config)

// Safe to call from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        response, err := client.Generate(ctx, fmt.Sprintf("Prompt %d", id))
        // ...
    }(i)
}
wg.Wait()
```

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/llm/...

# Run with coverage
go test -cover ./pkg/llm/...

# Run specific test
go test -run TestConfig_Validate ./pkg/llm/...
```

### Test Coverage

- Config validation: 76.5% coverage
- API key validation: 80.0% coverage
- Mock client: Full interface compliance tests
- Error handling: Context cancellation tests
- Provider constants: Verified

**Note**: Provider implementations (anthropic.go, openai.go, google.go) require actual API keys for integration testing and are not covered by unit tests.

## Files

### Implementation Files

- **config.go** (117 lines): Configuration types and validation
- **client.go** (160 lines): Client interface, factory, and base implementation
- **anthropic.go** (167 lines): Anthropic (Claude) provider implementation
- **openai.go** (168 lines): OpenAI (GPT) provider implementation
- **google.go** (168 lines): Google (Gemini) provider implementation

### Test Files

- **config_test.go** (238 lines): Configuration and validation tests
- **examples_test.go** (251 lines): Usage examples and documentation
- **tests/unit/llm_client_test.go** (520 lines): Comprehensive unit tests with mocks

## Design Decisions

### Temperature Enforcement

Temperature is **strictly enforced at 0.0** to ensure deterministic output. This is critical for:

- Reproducible code generation
- Consistent test results
- Reliable clarification workflows
- Predictable behavior in production

Any attempt to create a client with non-zero temperature will fail validation.

### Retry Logic

The package implements exponential backoff retry logic:

1. Initial retry delay: 1 second (configurable)
2. Each retry doubles the delay
3. Maximum attempts: 3 (configurable)
4. Context cancellation honored during retries
5. Successful retries logged at INFO level
6. Failed retries logged at WARN level

### Error Wrapping

All errors are wrapped with provider and operation context:

```
llm[anthropic:claude-sonnet-4-5] generate: <underlying error>
```

This makes debugging easier and provides clear error messages.

### Provider Abstraction

The package uses `langchaingo` for provider abstractions, which:

- Handles authentication and API specifics
- Provides consistent interfaces across providers
- Manages rate limiting and retries at the SDK level
- Reduces maintenance burden

## API Key Security

The `Config.String()` method masks API keys:

```go
config := llm.Config{APIKey: "sk-ant-api03-very-secret-key"}
fmt.Println(config.String())
// Output: ... APIKey=sk-a*** ...
```

Never log or print the full API key.

## Best Practices

1. **Use context with timeout**: Always pass a context with timeout to prevent hanging requests
2. **Reuse clients**: Create one client and reuse it (thread-safe)
3. **Handle errors**: Check and handle all errors appropriately
4. **Validate config early**: Call `config.Validate()` before creating clients
5. **Use structured logging**: Log operations with context for debugging
6. **Respect rate limits**: Provider SDKs handle rate limiting, but be mindful of usage

## License

Part of the GoCreator project.
