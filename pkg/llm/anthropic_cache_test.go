package llm

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestCacheableMessage(t *testing.T) {
	tests := []struct {
		name string
		msg  CacheableMessage
		want string
	}{
		{
			name: "message without cache",
			msg: CacheableMessage{
				Role:    "user",
				Content: "Hello",
				Cache:   nil,
			},
			want: "Hello",
		},
		{
			name: "message with 5m cache",
			msg: CacheableMessage{
				Role:    "system",
				Content: "System instructions",
				Cache:   NewCacheControl("5m"),
			},
			want: "System instructions",
		},
		{
			name: "message with 1h cache",
			msg: CacheableMessage{
				Role:    "system",
				Content: "Long-lived context",
				Cache:   NewCacheControl("1h"),
			},
			want: "Long-lived context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.msg.Content != tt.want {
				t.Errorf("CacheableMessage.Content = %v, want %v", tt.msg.Content, tt.want)
			}

			// Verify cache control if present
			if tt.msg.Cache != nil {
				if tt.msg.Cache.Type != "ephemeral" {
					t.Errorf("CacheControl.Type = %v, want ephemeral", tt.msg.Cache.Type)
				}
			}
		})
	}
}

func TestPromptCacheMetrics(t *testing.T) {
	metrics := PromptCacheMetrics{
		CacheCreationTokens: 1000,
		CacheReadTokens:     5000,
		InputTokens:         500,
		OutputTokens:        200,
		CacheHits:           10,
		CacheMisses:         2,
	}

	// Test cache hit rate
	hitRate := metrics.CacheHitRate()
	expectedHitRate := 10.0 / 12.0 * 100.0 // 83.33%
	if hitRate < expectedHitRate-0.1 || hitRate > expectedHitRate+0.1 {
		t.Errorf("CacheHitRate() = %v, want ~%v", hitRate, expectedHitRate)
	}

	// Test tokens saved
	saved := metrics.TokensSaved()
	expectedSaved := int64(float64(5000) * 0.9) // 4500
	if saved != expectedSaved {
		t.Errorf("TokensSaved() = %v, want %v", saved, expectedSaved)
	}

	// Test cost savings percent
	savings := metrics.CostSavingsPercent()
	if savings <= 0 || savings >= 100 {
		t.Errorf("CostSavingsPercent() = %v, want value between 0 and 100", savings)
	}
}

func TestPromptBuilder(t *testing.T) {
	builder := NewPromptBuilder("5m")

	builder.AddCacheable("System instructions: You are a helpful assistant.")
	builder.AddCacheable("Guidelines: Follow Go best practices.")
	builder.AddDynamic("User question: What is the capital of France?")

	messages := builder.Build()

	if len(messages) != 2 {
		t.Fatalf("Build() returned %d messages, want 2", len(messages))
	}

	// First message should be system with cache
	if messages[0].Role != "system" {
		t.Errorf("messages[0].Role = %v, want system", messages[0].Role)
	}
	if messages[0].Cache == nil {
		t.Error("messages[0].Cache is nil, want cache control")
	}
	if messages[0].Cache != nil && messages[0].Cache.TTL != "5m" {
		t.Errorf("messages[0].Cache.TTL = %v, want 5m", messages[0].Cache.TTL)
	}

	// Second message should be user without cache
	if messages[1].Role != "user" {
		t.Errorf("messages[1].Role = %v, want user", messages[1].Role)
	}
	if messages[1].Cache != nil {
		t.Error("messages[1].Cache is not nil, want no cache for dynamic content")
	}
}

func TestPromptBuilderSinglePrompt(t *testing.T) {
	builder := NewPromptBuilder("5m")
	builder.AddCacheable("Part 1")
	builder.AddDynamic("Part 2")

	prompt := builder.BuildSinglePrompt()

	if prompt == "" {
		t.Error("BuildSinglePrompt() returned empty string")
	}

	// Check that both parts are present
	if !containsString(prompt, "Part 1") || !containsString(prompt, "Part 2") {
		t.Error("BuildSinglePrompt() missing expected content")
	}
}

// TestAnthropicCacheIntegration tests the cache functionality with real API calls
// This test requires ANTHROPIC_API_KEY environment variable
func TestAnthropicCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	config := Config{
		Provider:      ProviderAnthropic,
		Model:         "claude-3-5-sonnet-20241022",
		Temperature:   0.0,
		APIKey:        apiKey,
		Timeout:       30 * time.Second,
		MaxTokens:     1024,
		MaxRetries:    3,
		RetryDelay:    time.Second,
		EnableCaching: true,
		CacheTTL:      "5m",
	}

	client, err := newAnthropicClient(config)
	if err != nil {
		t.Fatalf("newAnthropicClient() error = %v", err)
	}

	ctx := context.Background()

	// Create cacheable messages with stable system content
	builder := NewPromptBuilder("5m")
	builder.AddCacheable(`You are an expert Go developer.

Follow these guidelines:
1. Write idiomatic Go code
2. Use proper error handling
3. Include comprehensive documentation
4. Follow Go best practices

This is a large block of context that should be cached for subsequent requests.
It contains detailed instructions and examples that don't change between requests.`)
	builder.AddDynamic("Write a simple hello world function in Go.")

	messages := builder.Build()

	// First request - should create cache
	t.Log("Making first request (cache creation)...")
	response1, err := client.GenerateWithCache(ctx, messages)
	if err != nil {
		t.Fatalf("GenerateWithCache() first call error = %v", err)
	}

	if response1 == "" {
		t.Error("First response is empty")
	}

	metrics1 := client.GetCacheMetrics()
	t.Logf("After first request:")
	t.Logf("  Cache creation tokens: %d", metrics1.CacheCreationTokens)
	t.Logf("  Cache read tokens: %d", metrics1.CacheReadTokens)
	t.Logf("  Input tokens: %d", metrics1.InputTokens)
	t.Logf("  Cache hits: %d", metrics1.CacheHits)
	t.Logf("  Cache misses: %d", metrics1.CacheMisses)

	// Second request with same cacheable content but different dynamic part
	builder2 := NewPromptBuilder("5m")
	builder2.AddCacheable(`You are an expert Go developer.

Follow these guidelines:
1. Write idiomatic Go code
2. Use proper error handling
3. Include comprehensive documentation
4. Follow Go best practices

This is a large block of context that should be cached for subsequent requests.
It contains detailed instructions and examples that don't change between requests.`)
	builder2.AddDynamic("Write a function to add two numbers in Go.")

	messages2 := builder2.Build()

	t.Log("Making second request (should hit cache)...")
	response2, err := client.GenerateWithCache(ctx, messages2)
	if err != nil {
		t.Fatalf("GenerateWithCache() second call error = %v", err)
	}

	if response2 == "" {
		t.Error("Second response is empty")
	}

	metrics2 := client.GetCacheMetrics()
	t.Logf("After second request:")
	t.Logf("  Cache creation tokens: %d", metrics2.CacheCreationTokens)
	t.Logf("  Cache read tokens: %d", metrics2.CacheReadTokens)
	t.Logf("  Input tokens: %d", metrics2.InputTokens)
	t.Logf("  Cache hits: %d", metrics2.CacheHits)
	t.Logf("  Cache misses: %d", metrics2.CacheMisses)
	t.Logf("  Cache hit rate: %.2f%%", metrics2.CacheHitRate())
	t.Logf("  Tokens saved: %d", metrics2.TokensSaved())
	t.Logf("  Cost savings: %.2f%%", metrics2.CostSavingsPercent())

	// Verify cache was hit on second request
	if metrics2.CacheReadTokens == 0 {
		t.Error("Second request did not hit cache (CacheReadTokens = 0)")
	}

	// Verify cache hit count increased
	if metrics2.CacheHits == 0 {
		t.Error("Cache hits did not increase")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
