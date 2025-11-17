package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CacheEntry represents a cached LLM response with metadata
type CacheEntry struct {
	Response  string
	Timestamp time.Time
	HitCount  int
}

// Cache provides in-memory caching of LLM responses for development
type Cache interface {
	// Get retrieves a cached response if available
	Get(key string) (string, bool)

	// Set stores a response in the cache
	Set(key string, response string)

	// Clear removes all entries from the cache
	Clear()

	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheStats provides statistics about cache performance
type CacheStats struct {
	Hits        int64
	Misses      int64
	Entries     int
	HitRate     float64
	TotalSizeKB int64
}

// inMemoryCache implements the Cache interface with thread-safe in-memory storage
type inMemoryCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	hits    int64
	misses  int64
	enabled bool
}

// CacheConfig holds configuration for the cache
type CacheConfig struct {
	// Enabled controls whether caching is active
	Enabled bool
}

// NewCache creates a new in-memory cache
func NewCache(cfg CacheConfig) Cache {
	return &inMemoryCache{
		entries: make(map[string]*CacheEntry),
		enabled: cfg.Enabled,
	}
}

// Get retrieves a cached response if available
func (c *inMemoryCache) Get(key string) (string, bool) {
	if !c.enabled {
		return "", false
	}

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return "", false
	}

	// Update hit count
	c.mu.Lock()
	c.hits++
	entry.HitCount++
	c.mu.Unlock()

	keyPreview := key
	if len(key) > 16 {
		keyPreview = key[:16] + "..."
	}
	log.Debug().
		Str("cache_key", keyPreview).
		Int("hit_count", entry.HitCount).
		Msg("LLM cache hit")

	return entry.Response, true
}

// Set stores a response in the cache
func (c *inMemoryCache) Set(key string, response string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		Response:  response,
		Timestamp: time.Now(),
		HitCount:  0,
	}

	keyPreview := key
	if len(key) > 16 {
		keyPreview = key[:16] + "..."
	}
	log.Debug().
		Str("cache_key", keyPreview).
		Int("total_entries", len(c.entries)).
		Msg("LLM response cached")
}

// Clear removes all entries from the cache
func (c *inMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.hits = 0
	c.misses = 0

	log.Info().Msg("LLM cache cleared")
}

// Stats returns cache statistics
func (c *inMemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalRequests := c.hits + c.misses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(c.hits) / float64(totalRequests)
	}

	// Calculate total cache size
	totalSize := int64(0)
	for _, entry := range c.entries {
		totalSize += int64(len(entry.Response))
	}

	return CacheStats{
		Hits:        c.hits,
		Misses:      c.misses,
		Entries:     len(c.entries),
		HitRate:     hitRate,
		TotalSizeKB: totalSize / 1024,
	}
}

// CachedClient wraps an LLM client with caching capabilities
type CachedClient struct {
	client Client
	cache  Cache
}

// NewCachedClient creates a new cached LLM client
func NewCachedClient(client Client, cache Cache) Client {
	return &CachedClient{
		client: client,
		cache:  cache,
	}
}

// Generate produces text from a single prompt (with caching)
func (c *CachedClient) Generate(ctx context.Context, prompt string) (string, error) {
	// Generate cache key
	key := c.generateCacheKey(prompt, nil)

	// Check cache first
	if cached, found := c.cache.Get(key); found {
		return cached, nil
	}

	// Cache miss - call underlying client
	response, err := c.client.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	// Store in cache
	c.cache.Set(key, response)

	return response, nil
}

// GenerateStructured produces structured output based on a schema (with caching)
func (c *CachedClient) GenerateStructured(ctx context.Context, prompt string, schema interface{}) (interface{}, error) {
	// For structured generation, we'll cache the JSON-serialized response
	key := c.generateCacheKey(prompt, schema)

	// Check cache first
	if cached, found := c.cache.Get(key); found {
		// Deserialize cached response
		var result interface{}
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
		// If deserialization fails, fall through to make actual call
	}

	// Cache miss - call underlying client
	response, err := c.client.GenerateStructured(ctx, prompt, schema)
	if err != nil {
		return nil, err
	}

	// Serialize and cache
	if serialized, err := json.Marshal(response); err == nil {
		c.cache.Set(key, string(serialized))
	}

	return response, nil
}

// Chat processes a sequence of messages and returns the assistant's response (with caching)
func (c *CachedClient) Chat(ctx context.Context, messages []Message) (string, error) {
	// Generate cache key from messages
	key := c.generateChatCacheKey(messages)

	// Check cache first
	if cached, found := c.cache.Get(key); found {
		return cached, nil
	}

	// Cache miss - call underlying client
	response, err := c.client.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	// Store in cache
	c.cache.Set(key, response)

	return response, nil
}

// Provider returns the name of the LLM provider
func (c *CachedClient) Provider() string {
	return c.client.Provider()
}

// Model returns the model being used
func (c *CachedClient) Model() string {
	return c.client.Model()
}

// generateCacheKey creates a unique cache key based on model, temperature, and prompt
func (c *CachedClient) generateCacheKey(prompt string, schema interface{}) string {
	// Include provider, model, and prompt in cache key
	// Note: Temperature is always 0.0 for determinism, so we don't need to include it
	keyData := fmt.Sprintf("%s:%s:%s", c.client.Provider(), c.client.Model(), prompt)

	// If schema is provided, include it in the key
	if schema != nil {
		if schemaJSON, err := json.Marshal(schema); err == nil {
			keyData += ":" + string(schemaJSON)
		}
	}

	// Hash the key data to create a fixed-length key
	hash := sha256.Sum256([]byte(keyData))
	return hex.EncodeToString(hash[:])
}

// generateChatCacheKey creates a unique cache key for chat messages
func (c *CachedClient) generateChatCacheKey(messages []Message) string {
	// Serialize messages to JSON for hashing
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		// Fallback to simple concatenation if JSON fails
		var keyData string
		for _, msg := range messages {
			keyData += msg.Role + ":" + msg.Content + "|"
		}
		hash := sha256.Sum256([]byte(keyData))
		return hex.EncodeToString(hash[:])
	}

	// Include provider and model in cache key
	keyData := fmt.Sprintf("%s:%s:%s", c.client.Provider(), c.client.Model(), string(messagesJSON))

	hash := sha256.Sum256([]byte(keyData))
	return hex.EncodeToString(hash[:])
}
