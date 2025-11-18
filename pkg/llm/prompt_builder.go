package llm

import "strings"

// PromptBuilder helps construct prompts with cacheable and non-cacheable sections
type PromptBuilder struct {
	cacheableParts []string // Static parts that should be cached
	dynamicParts   []string // Dynamic parts that shouldn't be cached
	cacheTTL       string   // Cache TTL (5m or 1h)
}

// NewPromptBuilder creates a new prompt builder
func NewPromptBuilder(cacheTTL string) *PromptBuilder {
	if cacheTTL == "" {
		cacheTTL = "5m"
	}
	return &PromptBuilder{
		cacheableParts: []string{},
		dynamicParts:   []string{},
		cacheTTL:       cacheTTL,
	}
}

// AddCacheable adds a static section that should be cached
// These sections should be stable across multiple calls (e.g., FCS schema, guidelines)
func (pb *PromptBuilder) AddCacheable(content string) *PromptBuilder {
	pb.cacheableParts = append(pb.cacheableParts, content)
	return pb
}

// AddDynamic adds a dynamic section that shouldn't be cached
// These sections change between calls (e.g., specific task instructions, file context)
func (pb *PromptBuilder) AddDynamic(content string) *PromptBuilder {
	pb.dynamicParts = append(pb.dynamicParts, content)
	return pb
}

// Build returns a slice of CacheableMessages ready for use with GenerateWithCache
// The cacheable parts are marked with cache_control, dynamic parts are not
func (pb *PromptBuilder) Build() []CacheableMessage {
	messages := []CacheableMessage{}

	// Add cacheable parts as system messages with cache control
	if len(pb.cacheableParts) > 0 {
		// Combine all cacheable parts into one message
		combinedCacheable := strings.Join(pb.cacheableParts, "\n\n")
		messages = append(messages, CacheableMessage{
			Role:    "system",
			Content: combinedCacheable,
			Cache:   NewCacheControl(pb.cacheTTL),
		})
	}

	// Add dynamic parts as user message (no cache control)
	if len(pb.dynamicParts) > 0 {
		combinedDynamic := strings.Join(pb.dynamicParts, "\n\n")
		messages = append(messages, CacheableMessage{
			Role:    "user",
			Content: combinedDynamic,
			Cache:   nil, // Dynamic content is not cached
		})
	}

	return messages
}

// BuildSinglePrompt returns a single concatenated prompt string for non-caching clients
func (pb *PromptBuilder) BuildSinglePrompt() string {
	allParts := append(pb.cacheableParts, pb.dynamicParts...)
	return strings.Join(allParts, "\n\n")
}
