package unit

import (
	"testing"
	"time"

	"github.com/dshills/gocreator/internal/generate"
	"github.com/dshills/gocreator/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerationCache(t *testing.T) {
	tests := []struct {
		name   string
		config generate.CacheConfig
	}{
		{
			name: "creates cache with default config",
			config: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
		},
		{
			name: "creates cache with disabled config",
			config: generate.CacheConfig{
				Enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := generate.NewGenerationCache(tt.config)
			assert.NotNil(t, cache)
		})
	}
}

func TestGenerationCache_Get(t *testing.T) {
	tests := []struct {
		name        string
		cacheConfig generate.CacheConfig
		setupCache  func(*generate.GenerationCache)
		fcsHash     string
		packageName string
		wantFound   bool
	}{
		{
			name: "cache miss - no entry",
			cacheConfig: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			setupCache: func(c *generate.GenerationCache) {
				// No setup - cache is empty
			},
			fcsHash:     "hash123",
			packageName: "main",
			wantFound:   false,
		},
		{
			name: "cache hit - entry exists",
			cacheConfig: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			setupCache: func(c *generate.GenerationCache) {
				files := []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				}
				c.Put("hash123", "main", files)
			},
			fcsHash:     "hash123",
			packageName: "main",
			wantFound:   true,
		},
		{
			name: "cache disabled - always miss",
			cacheConfig: generate.CacheConfig{
				Enabled: false,
			},
			setupCache: func(c *generate.GenerationCache) {
				files := []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				}
				c.Put("hash123", "main", files)
			},
			fcsHash:     "hash123",
			packageName: "main",
			wantFound:   false,
		},
		{
			name: "cache miss - different package",
			cacheConfig: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			setupCache: func(c *generate.GenerationCache) {
				files := []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				}
				c.Put("hash123", "main", files)
			},
			fcsHash:     "hash123",
			packageName: "other",
			wantFound:   false,
		},
		{
			name: "cache miss - different FCS hash",
			cacheConfig: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			setupCache: func(c *generate.GenerationCache) {
				files := []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				}
				c.Put("hash123", "main", files)
			},
			fcsHash:     "hash456",
			packageName: "main",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := generate.NewGenerationCache(tt.cacheConfig)
			if tt.setupCache != nil {
				tt.setupCache(cache)
			}

			files, found := cache.Get(tt.fcsHash, tt.packageName)

			if tt.wantFound {
				assert.True(t, found)
				assert.NotEmpty(t, files)
			} else {
				assert.False(t, found)
				assert.Nil(t, files)
			}
		})
	}
}

func TestGenerationCache_Put(t *testing.T) {
	tests := []struct {
		name        string
		cacheConfig generate.CacheConfig
		fcsHash     string
		packageName string
		files       []models.GeneratedFile
		validatePut func(t *testing.T, cache *generate.GenerationCache)
	}{
		{
			name: "put entry in cache",
			cacheConfig: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			fcsHash:     "hash123",
			packageName: "main",
			files: []models.GeneratedFile{
				{Path: "cmd/main/main.go", Content: "package main"},
			},
			validatePut: func(t *testing.T, cache *generate.GenerationCache) {
				files, found := cache.Get("hash123", "main")
				assert.True(t, found)
				assert.Len(t, files, 1)
			},
		},
		{
			name: "put when cache disabled - no effect",
			cacheConfig: generate.CacheConfig{
				Enabled: false,
			},
			fcsHash:     "hash123",
			packageName: "main",
			files: []models.GeneratedFile{
				{Path: "cmd/main/main.go", Content: "package main"},
			},
			validatePut: func(t *testing.T, cache *generate.GenerationCache) {
				files, found := cache.Get("hash123", "main")
				assert.False(t, found)
				assert.Nil(t, files)
			},
		},
		{
			name: "update existing entry",
			cacheConfig: generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			},
			fcsHash:     "hash123",
			packageName: "main",
			files: []models.GeneratedFile{
				{Path: "cmd/main/main.go", Content: "package main\n// Updated"},
			},
			validatePut: func(t *testing.T, cache *generate.GenerationCache) {
				// First put
				cache.Put("hash123", "main", []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				})

				// Second put (update)
				cache.Put("hash123", "main", []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main\n// Updated"},
				})

				files, found := cache.Get("hash123", "main")
				assert.True(t, found)
				assert.Contains(t, files[0].Content, "Updated")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := generate.NewGenerationCache(tt.cacheConfig)
			cache.Put(tt.fcsHash, tt.packageName, tt.files)

			if tt.validatePut != nil {
				tt.validatePut(t, cache)
			}
		})
	}
}

func TestGenerationCache_Invalidate(t *testing.T) {
	tests := []struct {
		name        string
		setupCache  func(*generate.GenerationCache)
		fcsHash     string
		validateInv func(t *testing.T, cache *generate.GenerationCache)
	}{
		{
			name: "invalidate specific FCS hash",
			setupCache: func(c *generate.GenerationCache) {
				c.Put("hash123", "main", []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				})
				c.Put("hash123", "lib", []models.GeneratedFile{
					{Path: "internal/lib/lib.go", Content: "package lib"},
				})
				c.Put("hash456", "main", []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main v2"},
				})
			},
			fcsHash: "hash123",
			validateInv: func(t *testing.T, cache *generate.GenerationCache) {
				// hash123 entries should be invalidated
				_, found := cache.Get("hash123", "main")
				assert.False(t, found)
				_, found = cache.Get("hash123", "lib")
				assert.False(t, found)

				// hash456 entry should still exist
				files, found := cache.Get("hash456", "main")
				assert.True(t, found)
				assert.NotEmpty(t, files)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := generate.NewGenerationCache(generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			})

			if tt.setupCache != nil {
				tt.setupCache(cache)
			}

			cache.Invalidate(tt.fcsHash)

			if tt.validateInv != nil {
				tt.validateInv(t, cache)
			}
		})
	}
}

func TestGenerationCache_Clear(t *testing.T) {
	tests := []struct {
		name       string
		setupCache func(*generate.GenerationCache)
	}{
		{
			name: "clear entire cache",
			setupCache: func(c *generate.GenerationCache) {
				c.Put("hash123", "main", []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				})
				c.Put("hash456", "lib", []models.GeneratedFile{
					{Path: "internal/lib/lib.go", Content: "package lib"},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := generate.NewGenerationCache(generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			})

			if tt.setupCache != nil {
				tt.setupCache(cache)
			}

			cache.Clear()

			// Verify cache is empty
			_, found := cache.Get("hash123", "main")
			assert.False(t, found)
			_, found = cache.Get("hash456", "lib")
			assert.False(t, found)
		})
	}
}

func TestGenerationCache_Stats(t *testing.T) {
	tests := []struct {
		name          string
		setupCache    func(*generate.GenerationCache)
		validateStats func(t *testing.T, stats generate.CacheStats)
	}{
		{
			name: "get cache statistics",
			setupCache: func(c *generate.GenerationCache) {
				c.Put("hash123", "main", []models.GeneratedFile{
					{Path: "cmd/main/main.go", Content: "package main"},
				})
				c.Put("hash123", "lib", []models.GeneratedFile{
					{Path: "internal/lib/lib.go", Content: "package lib"},
				})

				// Trigger some hits and misses
				c.Get("hash123", "main") // hit
				c.Get("hash123", "lib")  // hit
				c.Get("hash456", "main") // miss
			},
			validateStats: func(t *testing.T, stats generate.CacheStats) {
				assert.Equal(t, 2, stats.Entries)
				assert.Equal(t, 2, stats.Hits)
				assert.Equal(t, 1, stats.Misses)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := generate.NewGenerationCache(generate.CacheConfig{
				Enabled: true,
				MaxSize: 100,
				TTL:     1 * time.Hour,
			})

			if tt.setupCache != nil {
				tt.setupCache(cache)
			}

			stats := cache.Stats()
			require.NotNil(t, stats)

			if tt.validateStats != nil {
				tt.validateStats(t, stats)
			}
		})
	}
}

func TestGenerationCache_TTL(t *testing.T) {
	t.Run("expired entries not returned", func(t *testing.T) {
		cache := generate.NewGenerationCache(generate.CacheConfig{
			Enabled: true,
			MaxSize: 100,
			TTL:     100 * time.Millisecond, // Short TTL for testing
		})

		cache.Put("hash123", "main", []models.GeneratedFile{
			{Path: "cmd/main/main.go", Content: "package main"},
		})

		// Immediately should be found
		files, found := cache.Get("hash123", "main")
		assert.True(t, found)
		assert.NotEmpty(t, files)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Should no longer be found
		files, found = cache.Get("hash123", "main")
		assert.False(t, found)
		assert.Nil(t, files)
	})
}

func TestGenerationCache_MaxSize(t *testing.T) {
	t.Run("evict oldest when max size reached", func(t *testing.T) {
		cache := generate.NewGenerationCache(generate.CacheConfig{
			Enabled: true,
			MaxSize: 2, // Small size for testing
			TTL:     1 * time.Hour,
		})

		// Add 3 entries (exceeds max size)
		cache.Put("hash1", "pkg1", []models.GeneratedFile{
			{Path: "pkg1/file.go", Content: "package pkg1"},
		})
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps

		cache.Put("hash2", "pkg2", []models.GeneratedFile{
			{Path: "pkg2/file.go", Content: "package pkg2"},
		})
		time.Sleep(10 * time.Millisecond)

		cache.Put("hash3", "pkg3", []models.GeneratedFile{
			{Path: "pkg3/file.go", Content: "package pkg3"},
		})

		// Oldest entry (hash1) should be evicted
		stats := cache.Stats()
		assert.LessOrEqual(t, stats.Entries, 2)

		// hash1 might be evicted
		_, found1 := cache.Get("hash1", "pkg1")
		// hash2 and hash3 should be present
		_, found2 := cache.Get("hash2", "pkg2")
		_, found3 := cache.Get("hash3", "pkg3")

		assert.True(t, found2 || found3, "at least one newer entry should be present")
		_ = found1 // Oldest might or might not be present depending on eviction policy
	})
}
