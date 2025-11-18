package providers

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// ProviderFactory is a function that creates a provider instance
type ProviderFactory func(id string, config *ProviderConfig, retryConfig *RetryConfig) (LLMProvider, error)

var (
	providerFactories = make(map[ProviderType]ProviderFactory)
	factoryMu         sync.RWMutex
)

// RegisterProviderFactory registers a factory function for a provider type
func RegisterProviderFactory(providerType ProviderType, factory ProviderFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	providerFactories[providerType] = factory
}

// Registry implements the ProviderRegistry interface
type Registry struct {
	mu              sync.RWMutex
	providers       map[string]LLMProvider   // Provider ID -> Provider instance
	roleMap         map[Role]*RoleAssignment // Role -> Assignment
	defaultProvider string                   // Default provider ID
	retryConfig     *RetryConfig             // Global retry configuration
	metrics         *MetricsCollector        // Metrics collection
}

// NewRegistry creates and initializes a new provider registry from configuration
func NewRegistry(configPath string) (*Registry, error) {
	// Load configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewRegistryFromConfig(config)
}

// NewRegistryFromConfig creates a registry from an already-loaded configuration
func NewRegistryFromConfig(config *MultiProviderConfig) (*Registry, error) {
	registry := &Registry{
		providers:       make(map[string]LLMProvider),
		roleMap:         config.Roles,
		defaultProvider: config.DefaultProvider,
		retryConfig:     config.Retry,
		metrics:         NewMetricsCollector(),
	}

	// Create provider instances
	for id, providerCfg := range config.Providers {
		provider, err := registry.createProvider(id, providerCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", id, err)
		}
		registry.providers[id] = provider
	}

	// Validate all providers in parallel
	validator := NewValidator(registry.providers)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := validator.ValidateAll(ctx); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	return registry, nil
}

// createProvider creates a provider adapter based on the provider type
func (r *Registry) createProvider(id string, config *ProviderConfig) (LLMProvider, error) {
	factoryMu.RLock()
	factory, exists := providerFactories[config.Type]
	factoryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no factory registered for provider type: %s", config.Type)
	}

	return factory(id, config, r.retryConfig)
}

// SelectProvider chooses the appropriate provider for a given role
func (r *Registry) SelectProvider(_ context.Context, role Role) (LLMProvider, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if role has an assignment
	if assignment, exists := r.roleMap[role]; exists {
		// Try primary provider
		if provider, exists := r.providers[assignment.PrimaryProvider]; exists {
			slog.Info("Provider selected",
				"role", role,
				"provider", assignment.PrimaryProvider,
				"selection_type", "primary",
			)
			return provider, assignment.PrimaryProvider, nil
		}

		// Try fallback providers in order
		for i, fallbackID := range assignment.FallbackProviders {
			if provider, exists := r.providers[fallbackID]; exists {
				slog.Info("Provider selected",
					"role", role,
					"provider", fallbackID,
					"selection_type", "fallback",
					"fallback_index", i,
				)
				return provider, fallbackID, nil
			}
		}
	}

	// Fall back to default provider
	if provider, exists := r.providers[r.defaultProvider]; exists {
		slog.Info("Provider selected",
			"role", role,
			"provider", r.defaultProvider,
			"selection_type", "default",
		)
		return provider, r.defaultProvider, nil
	}

	slog.Error("No provider available",
		"role", role,
	)
	return nil, "", fmt.Errorf("no provider available for role %s", role)
}

// RecordMetrics records performance metrics for a completed task
func (r *Registry) RecordMetrics(_ context.Context, metric ProviderMetrics) error {
	return r.metrics.Record(metric)
}

// GetMetrics retrieves aggregated metrics for a provider-role combination
func (r *Registry) GetMetrics(_ context.Context, providerID string, role Role, since time.Time) (*MetricsSummary, error) {
	return r.metrics.GetSummary(providerID, role, since)
}

// Shutdown gracefully shuts down all providers and flushes metrics
func (r *Registry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	// Shutdown all providers
	for id, provider := range r.providers {
		if err := provider.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown provider %s: %w", id, err))
		}
	}

	// Flush metrics
	if err := r.metrics.Flush(); err != nil {
		errors = append(errors, fmt.Errorf("failed to flush metrics: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// GetProviderForRole returns the provider and merged parameters for a given role
func (r *Registry) GetProviderForRole(ctx context.Context, role Role, globalConfig *ProviderConfig) (LLMProvider, map[string]any, error) {
	provider, providerID, err := r.SelectProvider(ctx, role)
	if err != nil {
		return nil, nil, err
	}

	// Get role assignment to check for parameter overrides
	var params map[string]any
	if assignment, exists := r.roleMap[role]; exists && providerID == assignment.PrimaryProvider {
		// Merge global provider parameters with role-specific overrides
		params = MergeParameters(globalConfig.Parameters, assignment.ParameterOverrides)
	} else {
		// No overrides, use global parameters
		params = globalConfig.Parameters
	}

	return provider, params, nil
}

// MetricsCollector collects and aggregates provider metrics
type MetricsCollector struct {
	mu     sync.RWMutex
	events []ProviderMetrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		events: make([]ProviderMetrics, 0),
	}
}

// Record records a single metric event
func (m *MetricsCollector) Record(event ProviderMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)
	return nil
}

// GetSummary computes aggregated metrics for a provider-role combination
func (m *MetricsCollector) GetSummary(providerID string, role Role, since time.Time) (*MetricsSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Filter events by provider, role, and time range
	var filtered []ProviderMetrics
	for _, event := range m.events {
		if event.ProviderID == providerID && event.Role == role {
			if since.IsZero() || event.Timestamp.After(since) {
				filtered = append(filtered, event)
			}
		}
	}

	if len(filtered) == 0 {
		return &MetricsSummary{
			ProviderID: providerID,
			Role:       role,
		}, nil
	}

	// Compute aggregates
	summary := &MetricsSummary{
		ProviderID: providerID,
		Role:       role,
		TimeRange: TimeRange{
			Start: filtered[0].Timestamp,
			End:   filtered[len(filtered)-1].Timestamp,
		},
		TotalRequests: len(filtered),
	}

	var totalResponseTime int64
	var totalTokens int
	responseTimes := make([]int64, 0, len(filtered))

	for _, event := range filtered {
		totalResponseTime += event.ResponseTimeMs
		totalTokens += event.TokensPrompt + event.TokensCompletion
		responseTimes = append(responseTimes, event.ResponseTimeMs)

		switch event.Status {
		case MetricStatusSuccess:
			summary.SuccessCount++
		case MetricStatusFailure:
			summary.FailureCount++
		}
	}

	summary.AvgResponseTime = float64(totalResponseTime) / float64(len(filtered))
	summary.TotalTokens = totalTokens
	summary.AvgTokensPerReq = float64(totalTokens) / float64(len(filtered))
	summary.SuccessRate = float64(summary.SuccessCount) / float64(summary.TotalRequests)

	// Calculate percentiles
	if len(responseTimes) > 0 {
		// Sort response times for percentile calculation
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})

		// P50 (median)
		p50Idx := len(responseTimes) / 2
		if p50Idx >= len(responseTimes) {
			p50Idx = len(responseTimes) - 1
		}

		// P95
		p95Idx := int(float64(len(responseTimes)) * 0.95)
		if p95Idx >= len(responseTimes) {
			p95Idx = len(responseTimes) - 1
		}

		summary.P50ResponseTime = responseTimes[p50Idx]
		summary.P95ResponseTime = responseTimes[p95Idx]
	}

	return summary, nil
}

// Flush flushes all pending metrics (placeholder for future SQLite implementation)
func (m *MetricsCollector) Flush() error {
	// TODO: Implement SQLite persistence in Phase 5
	return nil
}
