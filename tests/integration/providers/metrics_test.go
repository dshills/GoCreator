package providers_test

import (
	"context"
	"testing"
	"time"

	"github.com/dshills/gocreator/src/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetrics_EndToEndFlow tests the complete metrics lifecycle:
// 1. Execute task (simulated)
// 2. Record metrics
// 3. Query metrics
// 4. Verify aggregation
func TestMetrics_EndToEndFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration
	config, err := providers.LoadConfig("../../../tests/fixtures/providers/valid_config.yaml")
	require.NoError(t, err)

	// Create registry
	registry, err := providers.NewRegistryFromConfig(config)
	require.NoError(t, err)
	defer func() {
		_ = registry.Shutdown(context.Background())
	}()

	ctx := context.Background()

	// Simulate executing multiple tasks and recording metrics
	tasks := []struct {
		providerID   string
		role         providers.Role
		responseTime int64
		promptTokens int
		compTokens   int
		status       providers.MetricStatus
		errorMsg     string
	}{
		{"openai-fast", providers.RoleCoder, 150, 500, 200, providers.MetricStatusSuccess, ""},
		{"openai-fast", providers.RoleCoder, 200, 600, 250, providers.MetricStatusSuccess, ""},
		{"openai-fast", providers.RoleCoder, 180, 550, 220, providers.MetricStatusSuccess, ""},
		{"openai-fast", providers.RoleCoder, 300, 700, 0, providers.MetricStatusFailure, "rate limit"},
		{"anthropic-precise", providers.RoleReviewer, 250, 400, 150, providers.MetricStatusSuccess, ""},
		{"anthropic-precise", providers.RoleReviewer, 280, 450, 180, providers.MetricStatusSuccess, ""},
	}

	now := time.Now()

	for i, task := range tasks {
		metric := providers.ProviderMetrics{
			ProviderID:       task.providerID,
			Role:             task.role,
			Timestamp:        now.Add(time.Duration(i) * time.Second),
			ResponseTimeMs:   task.responseTime,
			TokensPrompt:     task.promptTokens,
			TokensCompletion: task.compTokens,
			Status:           task.status,
			ErrorMessage:     task.errorMsg,
		}

		err := registry.RecordMetrics(ctx, metric)
		assert.NoError(t, err, "Failed to record metric for task %d", i)
	}

	// Query metrics for openai-fast + coder
	summary, err := registry.GetMetrics(ctx, "openai-fast", providers.RoleCoder, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, summary)

	// Verify aggregation
	assert.Equal(t, "openai-fast", summary.ProviderID)
	assert.Equal(t, providers.RoleCoder, summary.Role)
	assert.Equal(t, 4, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessCount)
	assert.Equal(t, 1, summary.FailureCount)
	assert.InDelta(t, 0.75, summary.SuccessRate, 0.01) // 3/4 = 0.75

	// Verify average response time: (150 + 200 + 180 + 300) / 4 = 207.5
	assert.InDelta(t, 207.5, summary.AvgResponseTime, 1.0)

	// Verify total tokens: (500+200) + (600+250) + (550+220) + (700+0) = 3020
	assert.Equal(t, 3020, summary.TotalTokens)

	// Verify average tokens per request: 3020 / 4 = 755
	assert.InDelta(t, 755.0, summary.AvgTokensPerReq, 0.1)

	// Verify percentiles (sorted: 150, 180, 200, 300)
	assert.Equal(t, int64(200), summary.P50ResponseTime) // Index 2
	assert.Equal(t, int64(300), summary.P95ResponseTime) // Index 3 (last)

	// Query metrics for anthropic-precise + reviewer
	summary2, err := registry.GetMetrics(ctx, "anthropic-precise", providers.RoleReviewer, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, summary2)

	assert.Equal(t, "anthropic-precise", summary2.ProviderID)
	assert.Equal(t, providers.RoleReviewer, summary2.Role)
	assert.Equal(t, 2, summary2.TotalRequests)
	assert.Equal(t, 2, summary2.SuccessCount)
	assert.Equal(t, 0, summary2.FailureCount)
	assert.Equal(t, 1.0, summary2.SuccessRate)
}

// TestMetrics_TimeRangeFiltering tests that metrics queries can be filtered by time range
func TestMetrics_TimeRangeFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := providers.LoadConfig("../../../tests/fixtures/providers/valid_config.yaml")
	require.NoError(t, err)

	registry, err := providers.NewRegistryFromConfig(config)
	require.NoError(t, err)
	defer func() {
		_ = registry.Shutdown(context.Background())
	}()

	ctx := context.Background()

	// Record metrics at different times
	baseTime := time.Now().Add(-1 * time.Hour)

	// Old metrics (> 30 minutes ago)
	for i := 0; i < 3; i++ {
		metric := providers.ProviderMetrics{
			ProviderID:       "openai-fast",
			Role:             providers.RoleCoder,
			Timestamp:        baseTime.Add(time.Duration(i) * time.Minute),
			ResponseTimeMs:   100,
			TokensPrompt:     100,
			TokensCompletion: 50,
			Status:           providers.MetricStatusSuccess,
		}
		err := registry.RecordMetrics(ctx, metric)
		require.NoError(t, err)
	}

	// Recent metrics (< 30 minutes ago)
	recentTime := time.Now().Add(-15 * time.Minute)
	for i := 0; i < 2; i++ {
		metric := providers.ProviderMetrics{
			ProviderID:       "openai-fast",
			Role:             providers.RoleCoder,
			Timestamp:        recentTime.Add(time.Duration(i) * time.Minute),
			ResponseTimeMs:   200,
			TokensPrompt:     200,
			TokensCompletion: 100,
			Status:           providers.MetricStatusSuccess,
		}
		err := registry.RecordMetrics(ctx, metric)
		require.NoError(t, err)
	}

	// Query all metrics
	allSummary, err := registry.GetMetrics(ctx, "openai-fast", providers.RoleCoder, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, 5, allSummary.TotalRequests)

	// Query only recent metrics (last 30 minutes)
	since := time.Now().Add(-30 * time.Minute)
	recentSummary, err := registry.GetMetrics(ctx, "openai-fast", providers.RoleCoder, since)
	require.NoError(t, err)
	assert.Equal(t, 2, recentSummary.TotalRequests)
	assert.InDelta(t, 200.0, recentSummary.AvgResponseTime, 0.1)
}

// TestMetrics_NoData tests that querying metrics when no data exists returns empty summary
func TestMetrics_NoData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := providers.LoadConfig("../../../tests/fixtures/providers/valid_config.yaml")
	require.NoError(t, err)

	registry, err := providers.NewRegistryFromConfig(config)
	require.NoError(t, err)
	defer func() {
		_ = registry.Shutdown(context.Background())
	}()

	ctx := context.Background()

	// Query for provider-role combination with no metrics
	summary, err := registry.GetMetrics(ctx, "nonexistent-provider", providers.RoleCoder, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, "nonexistent-provider", summary.ProviderID)
	assert.Equal(t, providers.RoleCoder, summary.Role)
	assert.Equal(t, 0, summary.TotalRequests)
	assert.Equal(t, 0, summary.SuccessCount)
	assert.Equal(t, 0, summary.FailureCount)
	assert.Equal(t, 0.0, summary.SuccessRate)
}

// TestMetrics_ProviderRoleIsolation tests that metrics are isolated by provider-role combination
func TestMetrics_ProviderRoleIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := providers.LoadConfig("../../../tests/fixtures/providers/valid_config.yaml")
	require.NoError(t, err)

	registry, err := providers.NewRegistryFromConfig(config)
	require.NoError(t, err)
	defer func() {
		_ = registry.Shutdown(context.Background())
	}()

	ctx := context.Background()

	// Record metrics for different provider-role combinations
	combinations := []struct {
		providerID string
		role       providers.Role
	}{
		{"openai-fast", providers.RoleCoder},
		{"openai-fast", providers.RoleReviewer},
		{"anthropic-precise", providers.RoleCoder},
		{"anthropic-precise", providers.RoleReviewer},
	}

	for _, combo := range combinations {
		metric := providers.ProviderMetrics{
			ProviderID:       combo.providerID,
			Role:             combo.role,
			Timestamp:        time.Now(),
			ResponseTimeMs:   100,
			TokensPrompt:     100,
			TokensCompletion: 50,
			Status:           providers.MetricStatusSuccess,
		}
		err := registry.RecordMetrics(ctx, metric)
		require.NoError(t, err)
	}

	// Verify each combination has exactly 1 metric
	for _, combo := range combinations {
		summary, err := registry.GetMetrics(ctx, combo.providerID, combo.role, time.Time{})
		require.NoError(t, err)
		assert.Equal(t, 1, summary.TotalRequests,
			"Provider %s + Role %s should have exactly 1 metric",
			combo.providerID, combo.role)
	}
}
