package models

import "time"

// ContextFilterMetrics tracks metrics for context filtering
type ContextFilterMetrics struct {
	FilePath             string
	OriginalEntityCount  int
	FilteredEntityCount  int
	OriginalPackageCount int
	FilteredPackageCount int
	ReductionPercentage  float64
	FilterDuration       time.Duration
}

// GenerationMetrics tracks comprehensive metrics for code generation
type GenerationMetrics struct {
	// Timing
	TotalDuration      time.Duration
	PhaseTimings       map[string]time.Duration
	LLMCallDuration    time.Duration
	TemplateGeneration time.Duration

	// Token Usage
	TotalInputTokens  int64
	TotalOutputTokens int64
	CachedTokens      int64 // Tokens served from cache
	TokensSaved       int64 // Tokens saved by caching

	// Cache Performance
	CacheHitRate float64
	CacheHits    int
	CacheMisses  int

	// API Efficiency
	TotalLLMCalls     int
	BatchedCalls      int
	TemplateFiles     int
	LLMGeneratedFiles int

	// Cost
	EstimatedCostUSD float64
	CostBreakdown    map[string]float64 // Per provider

	// Parallelization
	ParallelPhases int
	TimeSaved      time.Duration // Time saved by parallelization

	// Context Filtering
	ContextFilteringMetrics []ContextFilterMetrics
	AvgReductionPercentage  float64
}

// AddContextFilterMetrics adds context filtering metrics
func (m *GenerationMetrics) AddContextFilterMetrics(metric ContextFilterMetrics) {
	m.ContextFilteringMetrics = append(m.ContextFilteringMetrics, metric)

	// Recalculate average reduction
	if len(m.ContextFilteringMetrics) > 0 {
		total := 0.0
		for _, cm := range m.ContextFilteringMetrics {
			total += cm.ReductionPercentage
		}
		m.AvgReductionPercentage = total / float64(len(m.ContextFilteringMetrics))
	}
}
