# Quickstart Guide: Multi-LLM Provider Support

**Feature**: 002-multi-llm | **Date**: 2025-11-17
**Audience**: GoCreator developers and contributors

---

## Overview

This guide covers:
1. Configuring multiple LLM providers
2. Adding a new provider adapter
3. Querying provider metrics
4. Troubleshooting common issues

---

## 1. Configuring Multiple Providers

### Basic Configuration

Create or edit your `config.yaml`:

```yaml
schema_version: "1.0"

providers:
  # OpenAI GPT-4
  openai-gpt4:
    type: openai
    model: gpt-4-turbo
    api_key: ${OPENAI_API_KEY}
    parameters:
      temperature: 0.7
      max_tokens: 4096

  # Anthropic Claude
  anthropic-claude:
    type: anthropic
    model: claude-3-5-sonnet-20241022
    api_key: ${ANTHROPIC_API_KEY}
    parameters:
      temperature: 0.5
      max_tokens: 8192

  # Google Gemini
  google-gemini:
    type: google
    model: gemini-pro
    api_key: ${GOOGLE_API_KEY}
    parameters:
      temperature: 0.6
      max_tokens: 2048

roles:
  coder:
    provider: openai-gpt4
    fallback:
      - anthropic-claude
    parameters:
      temperature: 0.8  # Higher creativity for code generation

  reviewer:
    provider: anthropic-claude
    parameters:
      temperature: 0.2  # Lower temperature for consistency

  planner:
    provider: anthropic-claude

  clarifier:
    provider: openai-gpt4

default_provider: anthropic-claude

retry:
  max_attempts: 3
  initial_backoff: 1s
  max_backoff: 30s
  multiplier: 2.0
```

### Environment Variables

Set your API keys:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."
```

Or use a `.env` file:

```bash
# .env
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GOOGLE_API_KEY=...
```

### Configuration Validation

Validate your configuration before running:

```bash
gocreator config validate config.yaml
```

This will:
- Check all provider IDs are unique
- Verify all role references exist
- Validate parameter types and ranges
- Test credential validity (requires API keys set)

---

## 2. Adding a New Provider Adapter

### Step 1: Implement the LLMProvider Interface

Create a new file `src/providers/adapters/newprovider.go`:

```go
package adapters

import (
    "context"
    "fmt"
    "time"

    "gocreator/src/providers"
)

type NewProviderAdapter struct {
    id         string
    config     *providers.ProviderConfig
    client     *NewProviderClient  // Your provider's SDK client
    retryConfig *providers.RetryConfig
}

func NewNewProviderAdapter(id string, config *providers.ProviderConfig, retryConfig *providers.RetryConfig) *NewProviderAdapter {
    return &NewProviderAdapter{
        id:          id,
        config:      config,
        retryConfig: retryConfig,
    }
}

// Initialize validates credentials and prepares the provider
func (a *NewProviderAdapter) Initialize(ctx context.Context) error {
    // Initialize your provider's SDK client
    client, err := NewProviderClient.New(a.config.APIKey, a.config.Endpoint)
    if err != nil {
        return fmt.Errorf("failed to initialize client: %w", err)
    }

    // Test credentials with a simple API call
    if err := client.ValidateCredentials(ctx); err != nil {
        return providers.NewProviderError(a.id, providers.ErrorCodeAuth, "invalid credentials", err)
    }

    a.client = client
    return nil
}

// Execute sends a request to the provider with retry logic
func (a *NewProviderAdapter) Execute(ctx context.Context, req providers.Request) (providers.Response, error) {
    var resp providers.Response
    var lastErr error

    err := a.retryConfig.Execute(ctx, func() error {
        start := time.Now()

        // Make the actual API call
        result, err := a.client.Complete(ctx, providers.CompletionRequest{
            Prompt:      req.Prompt,
            MaxTokens:   req.MaxTokens,
            Temperature: req.Temperature,
            // Map other parameters...
        })

        if err != nil {
            // Classify error as retryable or not
            if isRetryable(err) {
                return err  // Will retry
            }
            // Non-retryable error
            return providers.NewProviderError(a.id, classifyError(err), err.Error(), err)
        }

        // Success - build response
        resp = providers.Response{
            Content:        result.Text,
            TokensPrompt:   result.Usage.PromptTokens,
            TokensResponse: result.Usage.CompletionTokens,
            Model:          result.Model,
            Metadata: map[string]string{
                "response_time_ms": fmt.Sprintf("%d", time.Since(start).Milliseconds()),
            },
        }

        return nil
    })

    if err != nil {
        return providers.Response{Error: err}, err
    }

    return resp, nil
}

func (a *NewProviderAdapter) Name() string {
    return a.id
}

func (a *NewProviderAdapter) Type() providers.ProviderType {
    return providers.ProviderType("newprovider")
}

func (a *NewProviderAdapter) Shutdown(ctx context.Context) error {
    if a.client != nil {
        return a.client.Close()
    }
    return nil
}

// Helper functions
func isRetryable(err error) bool {
    // Implement logic to determine if error is retryable
    // Rate limits, timeouts, 5xx errors -> true
    // Auth errors, 4xx errors -> false
    return false
}

func classifyError(err error) providers.ErrorCode {
    // Classify error into ErrorCode enum
    // Check error type/message and return appropriate code
    return providers.ErrorCodeUnknown
}
```

### Step 2: Register the Provider Type

Edit `src/providers/registry.go` to add your provider:

```go
func (r *Registry) createProvider(id string, config *ProviderConfig) (LLMProvider, error) {
    switch config.Type {
    case ProviderTypeOpenAI:
        return adapters.NewOpenAIAdapter(id, config, r.retryConfig), nil
    case ProviderTypeAnthropic:
        return adapters.NewAnthropicAdapter(id, config, r.retryConfig), nil
    case ProviderTypeGoogle:
        return adapters.NewGoogleAdapter(id, config, r.retryConfig), nil
    case ProviderType("newprovider"):  // Add your provider
        return adapters.NewNewProviderAdapter(id, config, r.retryConfig), nil
    default:
        return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
    }
}
```

### Step 3: Add Tests

Create `tests/unit/providers/newprovider_test.go`:

```go
package providers_test

import (
    "context"
    "testing"
    "time"

    "gocreator/src/providers"
    "gocreator/src/providers/adapters"
)

func TestNewProviderAdapter_Initialize(t *testing.T) {
    config := &providers.ProviderConfig{
        Type:   "newprovider",
        Model:  "test-model",
        APIKey: "test-key",
    }
    retryConfig := &providers.RetryConfig{
        MaxAttempts:    3,
        InitialBackoff: 1 * time.Second,
    }

    adapter := adapters.NewNewProviderAdapter("test-provider", config, retryConfig)

    ctx := context.Background()
    err := adapter.Initialize(ctx)

    if err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }

    if adapter.Name() != "test-provider" {
        t.Errorf("Expected name 'test-provider', got '%s'", adapter.Name())
    }
}

func TestNewProviderAdapter_Execute(t *testing.T) {
    // Test successful execution
    // Test retry on retryable error
    // Test failure on non-retryable error
}
```

### Step 4: Update Documentation

Add your provider to `CLAUDE.md`:

```markdown
## Active Technologies
- Go 1.21+ (requires generics support) (001-core-implementation)
- NewProvider LLM integration (002-multi-llm)
```

---

## 3. Querying Provider Metrics

### CLI Commands

#### View Metrics Summary

```bash
# All providers, all roles, all time
gocreator metrics summary

# Specific provider and role
gocreator metrics summary --provider openai-gpt4 --role coder

# Time range filter (last 24 hours)
gocreator metrics summary --since 24h

# JSON output for scripting
gocreator metrics summary --format json
```

Example output:

```
Provider Metrics Summary
========================

Provider: openai-gpt4, Role: coder
  Total Requests: 150
  Success Rate: 96.7%
  Avg Response Time: 1,250ms
  P95 Response Time: 2,100ms
  Total Tokens: 450,000
  Avg Tokens/Request: 3,000

Provider: anthropic-claude, Role: reviewer
  Total Requests: 80
  Success Rate: 100%
  Avg Response Time: 800ms
  P95 Response Time: 1,500ms
  Total Tokens: 240,000
  Avg Tokens/Request: 3,000
```

#### Export Metrics

```bash
# Export to CSV
gocreator metrics export --format csv --output metrics.csv

# Export to JSON
gocreator metrics export --format json --output metrics.json

# Export with time range
gocreator metrics export --since 7d --until 1d --output last-week.csv
```

### Programmatic Access

```go
package main

import (
    "context"
    "fmt"
    "time"

    "gocreator/src/providers"
)

func main() {
    registry := providers.NewRegistry(config)

    ctx := context.Background()
    since := time.Now().Add(-24 * time.Hour)

    summary, err := registry.GetMetrics(ctx, "openai-gpt4", providers.RoleCoder, since)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Avg Response Time: %.2fms\n", summary.AvgResponseTime)
    fmt.Printf("Success Rate: %.2f%%\n", summary.SuccessRate*100)
    fmt.Printf("Total Requests: %d\n", summary.TotalRequests)
}
```

---

## 4. Troubleshooting

### Issue: "Provider authentication failed"

**Symptoms**: Startup fails with "AUTH_FAILED" error

**Solutions**:
1. Check API key is set in environment:
   ```bash
   echo $OPENAI_API_KEY  # Should not be empty
   ```

2. Verify API key format:
   - OpenAI: `sk-...`
   - Anthropic: `sk-ant-...`
   - Google: varies

3. Test API key manually:
   ```bash
   curl https://api.openai.com/v1/models \
     -H "Authorization: Bearer $OPENAI_API_KEY"
   ```

4. Check for whitespace/newlines in key:
   ```bash
   export OPENAI_API_KEY=$(echo $OPENAI_API_KEY | tr -d '\n\r ')
   ```

---

### Issue: "Role provider not found"

**Symptoms**: Configuration validation fails with "provider 'xyz' not found"

**Solutions**:
1. Check provider ID spelling in roles section matches providers section
2. Ensure provider is defined before being referenced
3. Validate YAML syntax (no tabs, correct indentation)

Example of incorrect configuration:

```yaml
roles:
  coder:
    provider: openai-gpt4  # ❌ Provider not defined yet

providers:
  openai-gpt-4:  # ❌ Typo: dash vs underscore
    type: openai
```

Corrected:

```yaml
providers:
  openai-gpt4:  # ✅ Define first
    type: openai

roles:
  coder:
    provider: openai-gpt4  # ✅ Exact match
```

---

### Issue: "All providers failed"

**Symptoms**: Workflow execution fails with "all providers failed for role"

**Solutions**:
1. Check provider status:
   ```bash
   gocreator providers status
   ```

2. Verify network connectivity:
   ```bash
   ping api.openai.com
   curl https://api.anthropic.com
   ```

3. Check rate limits:
   - View recent errors: `gocreator metrics summary --status failure`
   - Look for "RATE_LIMIT" error codes
   - Increase `retry.initial_backoff` if rate-limited

4. Add more fallback providers:
   ```yaml
   roles:
     coder:
       provider: openai-gpt4
       fallback:
         - anthropic-claude
         - google-gemini  # Add more fallbacks
   ```

---

### Issue: "Metrics queries are slow"

**Symptoms**: `gocreator metrics summary` takes > 5 seconds

**Solutions**:
1. Check metrics database size:
   ```bash
   ls -lh ~/.gocreator/metrics.db
   ```

2. Archive old metrics:
   ```bash
   gocreator metrics archive --before 30d --output old-metrics.json
   gocreator metrics prune --before 30d
   ```

3. Rebuild indexes:
   ```bash
   gocreator metrics reindex
   ```

4. Use time range filters:
   ```bash
   # Only last 7 days
   gocreator metrics summary --since 7d
   ```

---

### Issue: "Provider parameter overrides not working"

**Symptoms**: Role-specific temperature/max_tokens not applied

**Solutions**:
1. Verify parameter names match provider expectations:
   ```yaml
   parameters:
     temperature: 0.8  # ✅ Correct
     temp: 0.8         # ❌ Wrong parameter name
   ```

2. Check parameter types:
   ```yaml
   parameters:
     temperature: 0.8   # ✅ Float
     temperature: "0.8" # ❌ String
     max_tokens: 4096   # ✅ Int
     max_tokens: 4096.0 # ❌ Float
   ```

3. Ensure parameters are in `parameters` block, not top-level:
   ```yaml
   roles:
     coder:
       provider: openai-gpt4
       temperature: 0.8  # ❌ Wrong location
       parameters:       # ✅ Correct
         temperature: 0.8
   ```

4. Enable debug logging to see applied parameters:
   ```bash
   GOCREATOR_LOG_LEVEL=debug gocreator generate spec.md
   ```

---

## Advanced Topics

### Custom Retry Strategies

For specific providers that need different retry behavior:

```go
// In your provider adapter
func (a *CustomAdapter) Execute(ctx context.Context, req providers.Request) (providers.Response, error) {
    // Custom retry for rate limits
    customRetry := providers.RetryConfig{
        MaxAttempts:    5,  // More attempts for rate limits
        InitialBackoff: 5 * time.Second,
        MaxBackoff:     60 * time.Second,
        Multiplier:     2.0,
    }

    return customRetry.Execute(ctx, func() error {
        // Your provider call
    })
}
```

### Monitoring Provider Health

Create a health check endpoint:

```go
func (r *Registry) HealthCheck(ctx context.Context) map[string]bool {
    health := make(map[string]bool)

    for id, provider := range r.providers {
        err := provider.Initialize(ctx)
        health[id] = (err == nil)
    }

    return health
}
```

---

## Next Steps

1. **Read the full architecture**: See [research.md](./research.md) for design decisions
2. **Review data model**: See [data-model.md](./data-model.md) for entity definitions
3. **Check API contracts**: See [contracts/provider-registry.yaml](./contracts/provider-registry.yaml)
4. **Run tests**: `go test ./tests/unit/providers/...`

---

**Last Updated**: 2025-11-17
**Maintainer**: GoCreator Team
