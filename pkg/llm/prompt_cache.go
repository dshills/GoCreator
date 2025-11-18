package llm

// CacheableMessage represents a message with optional cache control
// This allows marking specific portions of prompts for caching (Anthropic prompt caching)
type CacheableMessage struct {
	Role    string        // "system", "user", "assistant"
	Content string        // Message content
	Cache   *CacheControl // Optional cache control (nil means not cached)
}

// CacheControl specifies caching behavior for a message (Anthropic prompt caching)
type CacheControl struct {
	// Type is always "ephemeral" for Anthropic
	Type string

	// TTL specifies cache duration: "5m" or "1h"
	// Defaults to "5m" if not specified
	TTL string
}

// NewCacheControl creates a cache control with the specified TTL
func NewCacheControl(ttl string) *CacheControl {
	if ttl == "" {
		ttl = "5m"
	}
	return &CacheControl{
		Type: "ephemeral",
		TTL:  ttl,
	}
}

// PromptCacheMetrics tracks Anthropic prompt cache usage statistics
type PromptCacheMetrics struct {
	// CacheCreationTokens are tokens used to create cache entries (1.25x cost for 5m, 2x for 1h)
	CacheCreationTokens int64

	// CacheReadTokens are tokens read from cache (0.1x cost)
	CacheReadTokens int64

	// InputTokens are regular uncached input tokens (1x cost)
	InputTokens int64

	// OutputTokens are generated tokens (standard output pricing)
	OutputTokens int64

	// CacheHits is the number of successful cache hits
	CacheHits int64

	// CacheMisses is the number of cache misses
	CacheMisses int64
}

// CacheHitRate returns the percentage of requests that hit the cache
func (m *PromptCacheMetrics) CacheHitRate() float64 {
	total := m.CacheHits + m.CacheMisses
	if total == 0 {
		return 0.0
	}
	return float64(m.CacheHits) / float64(total) * 100.0
}

// TokensSaved returns the estimated tokens saved through caching
// Calculation: cache_read_tokens saved 90% of their cost compared to regular input
func (m *PromptCacheMetrics) TokensSaved() int64 {
	// Each cache read token saves 0.9x its value (costs 0.1x instead of 1.0x)
	return int64(float64(m.CacheReadTokens) * 0.9)
}

// CostSavingsPercent returns the estimated cost savings from caching
// Compares cached scenario to non-cached scenario
func (m *PromptCacheMetrics) CostSavingsPercent() float64 {
	// Calculate cost without caching (all tokens at 1x)
	costWithoutCache := float64(m.InputTokens + m.CacheCreationTokens + m.CacheReadTokens)

	if costWithoutCache == 0 {
		return 0.0
	}

	// Calculate actual cost with caching
	// - Regular input tokens: 1.0x
	// - Cache creation (5m): 1.25x
	// - Cache creation (1h): 2.0x (we'll assume 1.25x as average)
	// - Cache reads: 0.1x
	costWithCache := float64(m.InputTokens) +
		float64(m.CacheCreationTokens)*1.25 +
		float64(m.CacheReadTokens)*0.1

	savings := (costWithoutCache - costWithCache) / costWithoutCache * 100.0
	return savings
}
