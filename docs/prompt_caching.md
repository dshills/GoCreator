# Anthropic Prompt Caching Implementation

## Overview

This document describes the implementation of Anthropic's Prompt Caching API in GoCreator to reduce token costs by 60-80%.

## What is Prompt Caching?

Anthropic's Prompt Caching allows you to mark stable portions of prompts (like FCS schemas, guidelines, and examples) to be cached between API calls. Cached content is:
- **Stored for 5 minutes (default) or 1 hour**
- **Costs 10% of regular input tokens** when read from cache (90% savings)
- **Costs 1.25x for 5m cache or 2x for 1h cache** when initially created
- **Automatically shared** across requests with identical cached content

## Architecture

### Core Components

1. **CacheableMessage** (`pkg/llm/prompt_cache.go`):
   - Represents a message with optional cache control
   - Contains Role, Content, and optional Cache metadata

2. **CacheControl** (`pkg/llm/prompt_cache.go`):
   - Specifies caching behavior (type: "ephemeral", TTL: "5m" or "1h")

3. **PromptBuilder** (`pkg/llm/prompt_builder.go`):
   - Separates cacheable (static) from dynamic content
   - Simplifies prompt construction with caching

4. **AnthropicClient.GenerateWithCache** (`pkg/llm/anthropic.go`):
   - Uses Anthropic SDK directly for cache_control support
   - Tracks cache metrics (hits, misses, tokens saved)

5. **PromptCacheMetrics** (`pkg/llm/prompt_cache.go`):
   - Tracks cache performance
   - Calculates cost savings and hit rates

## Usage

### Basic Example

```go
import (
	"context"
	"github.com/dshills/gocreator/pkg/llm"
)

// Create LLM client with caching enabled
config := llm.DefaultConfig()
config.APIKey = "your-api-key"
config.EnableCaching = true
config.CacheTTL = "5m"  // or "1h"

client, err := llm.NewClient(config)
if err != nil {
	panic(err)
}

// Get Anthropic client for cache support
anthropicClient := client.(*llm.AnthropicClient)

// Build prompt with cacheable and dynamic parts
builder := llm.NewPromptBuilder("5m")

// CACHEABLE: Static instructions that don't change
builder.AddCacheable(`You are an expert Go developer.

Follow these guidelines:
1. Write idiomatic Go code
2. Use proper error handling
3. Include comprehensive documentation
4. Follow Go best practices

This large block of context will be cached for 5 minutes.`)

// DYNAMIC: Task-specific content that changes per request
builder.AddDynamic("Write a hello world function in Go.")

messages := builder.Build()

// Make request with caching
ctx := context.Background()
response, err := anthropicClient.GenerateWithCache(ctx, messages)
if err != nil {
	panic(err)
}

// Get cache metrics
metrics := anthropicClient.GetCacheMetrics()
fmt.Printf("Cache hit rate: %.2f%%\n", metrics.CacheHitRate())
fmt.Printf("Tokens saved: %d\n", metrics.TokensSaved())
fmt.Printf("Cost savings: %.2f%%\n", metrics.CostSavingsPercent())
```

### Integrating with Generation Pipeline

For GoCreator's generation pipeline, the typical pattern is:

```go
// In planner.go, coder.go, tester.go
func buildPromptWithCaching(fcs *models.FinalClarifiedSpecification, taskDetails string) []llm.CacheableMessage {
	builder := llm.NewPromptBuilder("5m")

	// CACHEABLE: FCS schema, generation guidelines (same across files)
	builder.AddCacheable(buildFCSSchema())
	builder.AddCacheable(buildGenerationGuidelines())
	builder.AddCacheable(buildCodeStandards())
	builder.AddCacheable(serializeFCS(fcs))  // FCS is stable during generation

	// DYNAMIC: Specific file/task instructions (changes per file)
	builder.AddDynamic(taskDetails)

	return builder.Build()
}
```

### Manual Message Construction

For advanced use cases:

```go
messages := []llm.CacheableMessage{
	{
		Role:    "system",
		Content: "Large static context...",
		Cache:   llm.NewCacheControl("1h"),  // Cache for 1 hour
	},
	{
		Role:    "system",
		Content: "More static context...",
		Cache:   llm.NewCacheControl("5m"),  // Cache for 5 minutes
	},
	{
		Role:    "user",
		Content: "Dynamic task instructions...",
		Cache:   nil,  // Don't cache dynamic content
	},
}

response, err := anthropicClient.GenerateWithCache(ctx, messages)
```

## Caching Strategy

### What to Cache

✅ **Good candidates for caching:**
- FCS schema and structure (static across entire generation)
- Code generation guidelines and best practices
- Coding standards and patterns
- Example code snippets
- Test generation instructions
- Large context documents (>1024 tokens minimum)

❌ **Don't cache:**
- Specific task instructions
- File-specific context
- User queries
- Anything that changes between requests

### Cache Breakpoints

Anthropic allows up to **4 cache breakpoints** per request. Organize content hierarchically:

1. **Most stable** (1h cache): Universal guidelines, schemas
2. **Project-stable** (5m cache): FCS details, project standards
3. **Phase-stable** (5m cache): Phase-specific context
4. **Dynamic**: Task-specific instructions

## Metrics and Monitoring

### Available Metrics

```go
type PromptCacheMetrics struct {
	CacheCreationTokens int64  // Tokens used to create cache entries
	CacheReadTokens     int64  // Tokens read from cache
	InputTokens         int64  // Regular uncached input tokens
	OutputTokens        int64  // Generated output tokens
	CacheHits           int64  // Number of cache hits
	CacheMisses         int64  // Number of cache misses
}
```

### Metric Methods

```go
metrics := client.GetCacheMetrics()

// Cache hit percentage
hitRate := metrics.CacheHitRate()  // Returns 0-100%

// Estimated tokens saved
saved := metrics.TokensSaved()

// Estimated cost savings percentage
savings := metrics.CostSavingsPercent()  // Returns 0-100%
```

### Logging Cache Performance

```go
log.Info().
	Int64("cache_hits", metrics.CacheHits).
	Int64("cache_misses", metrics.CacheMisses).
	Float64("hit_rate_pct", metrics.CacheHitRate()).
	Int64("tokens_saved", metrics.TokensSaved()).
	Float64("cost_savings_pct", metrics.CostSavingsPercent()).
	Msg("Prompt cache performance")
```

## Cost Analysis

### Pricing

- **Cache creation** (5m): 1.25x base input token cost
- **Cache creation** (1h): 2.0x base input token cost
- **Cache read**: 0.1x base input token cost (90% savings)
- **Regular input**: 1.0x base input token cost
- **Output**: Standard output pricing

### Example Cost Calculation

For a prompt with:
- 5,000 tokens of cacheable content
- 500 tokens of dynamic content
- 10 requests with same cacheable content

**Without caching:**
- Total input tokens: (5,000 + 500) × 10 = 55,000 tokens
- Cost: 55,000 × $1.00 = $55.00 (example rate)

**With caching (5m):**
- First request: 5,000 × 1.25 + 500 = 6,750 tokens
- Next 9 requests: (5,000 × 0.1 + 500) × 9 = 9,000 tokens
- Total: 15,750 tokens
- Cost: 15,750 × $1.00 = $15.75
- **Savings: 71.4%**

## Configuration

### Config Options

```go
type Config struct {
	// ... other fields ...

	// EnableCaching enables prompt caching (Anthropic only)
	EnableCaching bool

	// CacheTTL specifies the cache time-to-live ("5m" or "1h")
	CacheTTL string
}
```

### Default Configuration

```go
config := llm.DefaultConfig()
// EnableCaching: true (enabled by default)
// CacheTTL: "5m" (5-minute cache by default)
```

### Disabling Caching

```go
config := llm.DefaultConfig()
config.EnableCaching = false
```

## Best Practices

1. **Minimum Cache Size**: Anthropic recommends caching content ≥1024 tokens for cost-effectiveness

2. **Cache Ordering**: Place most stable content first in system messages

3. **TTL Selection**:
   - Use `5m` for content that may change occasionally
   - Use `1h` for highly stable content with frequent reuse

4. **Monitor Hit Rates**: Target 75%+ cache hit rate for optimal savings

5. **Batch Similar Requests**: Group requests with same cacheable content together

6. **Avoid Micro-Caching**: Don't cache very small blocks (<1024 tokens)

## Testing

Run the integration test:

```bash
export ANTHROPIC_API_KEY="your-key"
go test -v ./pkg/llm -run TestAnthropicCacheIntegration
```

Expected output shows:
- Cache creation on first request
- Cache hits on subsequent requests
- Token savings metrics
- Cost savings percentage

## Troubleshooting

### Cache Not Being Hit

**Problem**: CacheReadTokens remains 0

**Solutions**:
- Ensure cached content is identical (even whitespace matters)
- Verify content exceeds 1024 token minimum
- Check that cache hasn't expired (5m or 1h)
- Confirm EnableCaching is true

### Lower Than Expected Savings

**Problem**: Cost savings <60%

**Solutions**:
- Increase proportion of cacheable content
- Ensure high cache hit rate (>75%)
- Check that dynamic content isn't too large
- Consider using 1h cache for very stable content

### Type Errors

**Problem**: Cannot use anthropicClient

**Solution**: Type assert the client:
```go
anthropicClient, ok := client.(*llm.AnthropicClient)
if !ok {
	// Handle error: not an Anthropic client
}
```

## Future Enhancements

1. **Auto-detection**: Automatically identify cacheable vs dynamic content
2. **Cache Analytics**: Detailed per-phase cache performance reports
3. **Adaptive TTL**: Automatically adjust TTL based on usage patterns
4. **Multi-Provider**: Extend caching to other providers as they add support
5. **Cache Warming**: Pre-populate cache for known generation phases

## References

- [Anthropic Prompt Caching Documentation](https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching)
- [Anthropic SDK for Go](https://github.com/anthropics/anthropic-sdk-go)
- GoCreator Specification: `specs/003-prompt-caching/spec.md`
