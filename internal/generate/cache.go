package generate

import (
	"sync"
	"time"

	"github.com/dshills/gocreator/internal/models"
)

// CacheConfig configures the generation cache
type CacheConfig struct {
	Enabled bool
	MaxSize int
	TTL     time.Duration
}

// CacheStats contains cache statistics
type CacheStats struct {
	Entries int
	Hits    int
	Misses  int
}

// cacheEntry represents a single cache entry
type cacheEntry struct {
	fcsHash     string
	packageName string
	files       []models.GeneratedFile
	timestamp   time.Time
}

// cacheKey uniquely identifies a cache entry
type cacheKey struct {
	fcsHash     string
	packageName string
}

// GenerationCache caches generated code to avoid redundant regeneration
type GenerationCache struct {
	config  CacheConfig
	entries map[cacheKey]*cacheEntry
	mu      sync.RWMutex
	hits    int
	misses  int
}

// NewGenerationCache creates a new generation cache
func NewGenerationCache(config CacheConfig) *GenerationCache {
	if config.MaxSize <= 0 {
		config.MaxSize = 100 // Default max size
	}
	if config.TTL <= 0 {
		config.TTL = 24 * time.Hour // Default TTL
	}

	return &GenerationCache{
		config:  config,
		entries: make(map[cacheKey]*cacheEntry),
	}
}

// Get retrieves cached files for a specific FCS hash and package
func (gc *GenerationCache) Get(fcsHash, packageName string) ([]models.GeneratedFile, bool) {
	if !gc.config.Enabled {
		return nil, false
	}

	// First, try to read with read lock
	gc.mu.RLock()
	key := cacheKey{fcsHash: fcsHash, packageName: packageName}
	entry, exists := gc.entries[key]
	gc.mu.RUnlock()

	if !exists {
		// Need write lock to update misses
		gc.mu.Lock()
		gc.misses++
		gc.mu.Unlock()
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.timestamp) > gc.config.TTL {
		// Entry expired, need write lock to remove and update stats
		gc.mu.Lock()
		delete(gc.entries, key)
		gc.misses++
		gc.mu.Unlock()
		return nil, false
	}

	// Hit - need write lock to update hits counter
	gc.mu.Lock()
	gc.hits++
	gc.mu.Unlock()

	return entry.files, true
}

// Put stores files in the cache for a specific FCS hash and package
func (gc *GenerationCache) Put(fcsHash, packageName string, files []models.GeneratedFile) {
	if !gc.config.Enabled {
		return
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()

	key := cacheKey{fcsHash: fcsHash, packageName: packageName}

	// Check if we need to evict old entries
	if len(gc.entries) >= gc.config.MaxSize {
		gc.evictOldest()
	}

	// Store the new entry
	gc.entries[key] = &cacheEntry{
		fcsHash:     fcsHash,
		packageName: packageName,
		files:       files,
		timestamp:   time.Now(),
	}
}

// Invalidate removes all entries for a specific FCS hash
func (gc *GenerationCache) Invalidate(fcsHash string) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Remove all entries matching the FCS hash
	for key := range gc.entries {
		if key.fcsHash == fcsHash {
			delete(gc.entries, key)
		}
	}
}

// Clear removes all entries from the cache
func (gc *GenerationCache) Clear() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.entries = make(map[cacheKey]*cacheEntry)
	gc.hits = 0
	gc.misses = 0
}

// Stats returns cache statistics
func (gc *GenerationCache) Stats() CacheStats {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	return CacheStats{
		Entries: len(gc.entries),
		Hits:    gc.hits,
		Misses:  gc.misses,
	}
}

// evictOldest removes the oldest cache entry (LRU eviction)
func (gc *GenerationCache) evictOldest() {
	if len(gc.entries) == 0 {
		return
	}

	var oldestKey cacheKey
	var oldestTime time.Time
	first := true

	for key, entry := range gc.entries {
		if first || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
			first = false
		}
	}

	delete(gc.entries, oldestKey)
}
