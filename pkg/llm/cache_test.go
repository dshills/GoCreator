package llm

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	tests := []struct {
		name    string
		config  CacheConfig
		enabled bool
	}{
		{
			name:    "enabled cache",
			config:  CacheConfig{Enabled: true},
			enabled: true,
		},
		{
			name:    "disabled cache",
			config:  CacheConfig{Enabled: false},
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewCache(tt.config)
			require.NotNil(t, cache)

			// Disabled cache should never return hits
			if !tt.enabled {
				cache.Set("key", "value")
				_, found := cache.Get("key")
				assert.False(t, found)
			}
		})
	}
}

func TestCache_GetSet(t *testing.T) {
	cache := NewCache(CacheConfig{Enabled: true})

	tests := []struct {
		name     string
		key      string
		value    string
		wantHit  bool
		setupFn  func()
		verifyFn func(*testing.T)
	}{
		{
			name:    "cache miss on first get",
			key:     "key1",
			value:   "",
			wantHit: false,
		},
		{
			name:  "cache hit after set",
			key:   "key2",
			value: "cached_response",
			setupFn: func() {
				cache.Set("key2", "cached_response")
			},
			wantHit: true,
		},
		{
			name:  "different keys don't collide",
			key:   "key3",
			value: "",
			setupFn: func() {
				cache.Set("key4", "other_value")
			},
			wantHit: false,
		},
		{
			name:  "cache hit increments hit count",
			key:   "key5",
			value: "value5",
			setupFn: func() {
				cache.Set("key5", "value5")
			},
			wantHit: true,
			verifyFn: func(t *testing.T) {
				// Get multiple times
				cache.Get("key5")
				cache.Get("key5")
				stats := cache.Stats()
				assert.Greater(t, stats.Hits, int64(0))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn()
			}

			value, found := cache.Get(tt.key)

			assert.Equal(t, tt.wantHit, found)
			if tt.wantHit {
				assert.Equal(t, tt.value, value)
			}

			if tt.verifyFn != nil {
				tt.verifyFn(t)
			}
		})
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(CacheConfig{Enabled: true})

	// Add some entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Verify entries exist
	_, found := cache.Get("key1")
	assert.True(t, found)

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	_, found = cache.Get("key1")
	assert.False(t, found)
	_, found = cache.Get("key2")
	assert.False(t, found)
	_, found = cache.Get("key3")
	assert.False(t, found)

	// Verify stats are reset
	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(3), stats.Misses) // 3 misses from checks above
	assert.Equal(t, 0, stats.Entries)
}

func TestCache_Stats(t *testing.T) {
	cache := NewCache(CacheConfig{Enabled: true})

	// Initial stats
	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, 0, stats.Entries)
	assert.Equal(t, 0.0, stats.HitRate)

	// Add entries and generate hits/misses
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	cache.Get("key1") // hit
	cache.Get("key1") // hit
	cache.Get("key3") // miss

	stats = cache.Stats()
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 2, stats.Entries)
	assert.InDelta(t, 0.666, stats.HitRate, 0.01) // 2/3 = 0.666
	assert.GreaterOrEqual(t, stats.TotalSizeKB, int64(0))
}

func TestCache_ThreadSafety(t *testing.T) {
	cache := NewCache(CacheConfig{Enabled: true})

	// Run concurrent operations
	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)

				// Set
				cache.Set(key, value)

				// Get
				got, found := cache.Get(key)
				if found {
					assert.Equal(t, value, got)
				}

				// Stats (read-heavy operation)
				cache.Stats()
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional
	stats := cache.Stats()
	assert.Greater(t, stats.Entries, 0)
}

func TestCache_DisabledBehavior(t *testing.T) {
	cache := NewCache(CacheConfig{Enabled: false})

	// Set should not cache
	cache.Set("key1", "value1")

	// Get should always miss
	_, found := cache.Get("key1")
	assert.False(t, found)

	// Stats should show no activity
	stats := cache.Stats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, 0, stats.Entries)
}

// Mock client for testing CachedClient
type mockLLMClient struct {
	generateCount           int
	generateStructuredCount int
	chatCount               int
	mu                      sync.Mutex
}

func (m *mockLLMClient) Generate(_ context.Context, prompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateCount++
	return fmt.Sprintf("response_to_%s", prompt), nil
}

func (m *mockLLMClient) GenerateStructured(_ context.Context, _ string, _ interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateStructuredCount++
	return map[string]string{"result": "structured"}, nil
}

func (m *mockLLMClient) Chat(_ context.Context, messages []Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatCount++
	return fmt.Sprintf("chat_response_%d", len(messages)), nil
}

func (m *mockLLMClient) Provider() string {
	return "mock"
}

func (m *mockLLMClient) Model() string {
	return "mock-model"
}

func (m *mockLLMClient) getCount() (int, int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateCount, m.generateStructuredCount, m.chatCount
}

func TestCachedClient_Generate(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache)

	ctx := context.Background()

	// First call - cache miss
	resp1, err := client.Generate(ctx, "test_prompt")
	require.NoError(t, err)
	assert.Equal(t, "response_to_test_prompt", resp1)

	genCount1, _, _ := mock.getCount()
	assert.Equal(t, 1, genCount1)

	// Second call with same prompt - cache hit
	resp2, err := client.Generate(ctx, "test_prompt")
	require.NoError(t, err)
	assert.Equal(t, "response_to_test_prompt", resp2)

	genCount2, _, _ := mock.getCount()
	assert.Equal(t, genCount1, genCount2, "mock should not be called again on cache hit")

	// Verify cache stats
	stats := cache.Stats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses) // First call was a miss
}

func TestCachedClient_GenerateStructured(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache)

	ctx := context.Background()
	schema := map[string]string{"type": "object"}

	// First call - cache miss
	resp1, err := client.GenerateStructured(ctx, "test_prompt", schema)
	require.NoError(t, err)
	assert.NotNil(t, resp1)

	_, genStructCount1, _ := mock.getCount()
	assert.Equal(t, 1, genStructCount1)

	// Second call with same prompt and schema - cache hit
	resp2, err := client.GenerateStructured(ctx, "test_prompt", schema)
	require.NoError(t, err)
	assert.NotNil(t, resp2)

	_, genStructCount2, _ := mock.getCount()
	assert.Equal(t, 1, genStructCount2, "mock should not be called again on cache hit")
}

func TestCachedClient_Chat(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache)

	ctx := context.Background()
	messages := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	// First call - cache miss
	resp1, err := client.Chat(ctx, messages)
	require.NoError(t, err)
	assert.Equal(t, "chat_response_3", resp1)

	_, _, chatCount1 := mock.getCount()
	assert.Equal(t, 1, chatCount1)

	// Second call with same messages - cache hit
	resp2, err := client.Chat(ctx, messages)
	require.NoError(t, err)
	assert.Equal(t, "chat_response_3", resp2)

	_, _, chatCount2 := mock.getCount()
	assert.Equal(t, 1, chatCount2, "mock should not be called again on cache hit")
}

func TestCachedClient_DifferentPromptsNoCacheHit(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache)

	ctx := context.Background()

	// First prompt
	_, err := client.Generate(ctx, "prompt1")
	require.NoError(t, err)

	// Different prompt - should not hit cache
	_, err = client.Generate(ctx, "prompt2")
	require.NoError(t, err)

	genCount, _, _ := mock.getCount()
	assert.Equal(t, 2, genCount, "both calls should hit the underlying client")

	// Verify cache has 2 entries
	stats := cache.Stats()
	assert.Equal(t, 2, stats.Entries)
	assert.Equal(t, int64(0), stats.Hits)
}

func TestCachedClient_ProxyMethods(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache)

	assert.Equal(t, "mock", client.Provider())
	assert.Equal(t, "mock-model", client.Model())
}

func TestGenerateCacheKey_Consistency(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache).(*CachedClient)

	// Same prompt should generate same key
	key1 := client.generateCacheKey("test_prompt", nil)
	key2 := client.generateCacheKey("test_prompt", nil)
	assert.Equal(t, key1, key2)

	// Different prompts should generate different keys
	key3 := client.generateCacheKey("different_prompt", nil)
	assert.NotEqual(t, key1, key3)

	// Same prompt with different schemas should generate different keys
	schema1 := map[string]string{"type": "object"}
	schema2 := map[string]string{"type": "array"}
	key4 := client.generateCacheKey("test_prompt", schema1)
	key5 := client.generateCacheKey("test_prompt", schema2)
	assert.NotEqual(t, key4, key5)
}

func TestGenerateChatCacheKey_Consistency(t *testing.T) {
	mock := &mockLLMClient{}
	cache := NewCache(CacheConfig{Enabled: true})
	client := NewCachedClient(mock, cache).(*CachedClient)

	messages1 := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	}

	messages2 := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
	}

	messages3 := []Message{
		{Role: "user", Content: "Different"},
		{Role: "assistant", Content: "Message"},
	}

	// Same messages should generate same key
	key1 := client.generateChatCacheKey(messages1)
	key2 := client.generateChatCacheKey(messages2)
	assert.Equal(t, key1, key2)

	// Different messages should generate different keys
	key3 := client.generateChatCacheKey(messages3)
	assert.NotEqual(t, key1, key3)
}

func BenchmarkCache_Get(b *testing.B) {
	cache := NewCache(CacheConfig{Enabled: true})
	cache.Set("benchmark_key", "benchmark_value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("benchmark_key")
	}
}

func BenchmarkCache_Set(b *testing.B) {
	cache := NewCache(CacheConfig{Enabled: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key_%d", i)
		cache.Set(key, "value")
	}
}

func BenchmarkCache_Concurrent(b *testing.B) {
	cache := NewCache(CacheConfig{Enabled: true})

	// Pre-populate with some entries
	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key_%d", i), "value")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key_%d", i%100)
			if i%2 == 0 {
				cache.Get(key)
			} else {
				cache.Set(key, "value")
			}
			i++
		}
	})
}
