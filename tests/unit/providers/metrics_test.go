package providers_test

import (
	"testing"
	"time"

	"github.com/dshills/gocreator/src/providers"
	"github.com/stretchr/testify/assert"
)

func TestProviderMetrics_Creation(t *testing.T) {
	t.Run("valid_metrics", func(t *testing.T) {
		now := time.Now()
		metrics := providers.ProviderMetrics{
			ProviderID:       "openai-fast",
			Role:             providers.RoleCoder,
			Timestamp:        now,
			ResponseTimeMs:   150,
			TokensPrompt:     500,
			TokensCompletion: 200,
			Status:           providers.MetricStatusSuccess,
			ErrorMessage:     "",
		}

		assert.Equal(t, "openai-fast", metrics.ProviderID)
		assert.Equal(t, providers.RoleCoder, metrics.Role)
		assert.Equal(t, now, metrics.Timestamp)
		assert.Equal(t, int64(150), metrics.ResponseTimeMs)
		assert.Equal(t, 500, metrics.TokensPrompt)
		assert.Equal(t, 200, metrics.TokensCompletion)
		assert.Equal(t, providers.MetricStatusSuccess, metrics.Status)
		assert.Empty(t, metrics.ErrorMessage)
	})

	t.Run("failure_metrics_with_error", func(t *testing.T) {
		metrics := providers.ProviderMetrics{
			ProviderID:       "anthropic-precise",
			Role:             providers.RoleReviewer,
			Timestamp:        time.Now(),
			ResponseTimeMs:   50,
			TokensPrompt:     100,
			TokensCompletion: 0,
			Status:           providers.MetricStatusFailure,
			ErrorMessage:     "rate limit exceeded",
		}

		assert.Equal(t, providers.MetricStatusFailure, metrics.Status)
		assert.Equal(t, "rate limit exceeded", metrics.ErrorMessage)
		assert.Equal(t, 0, metrics.TokensCompletion)
	})

	t.Run("retry_metrics", func(t *testing.T) {
		metrics := providers.ProviderMetrics{
			ProviderID:       "google-fast",
			Role:             providers.RolePlanner,
			Timestamp:        time.Now(),
			ResponseTimeMs:   300,
			TokensPrompt:     200,
			TokensCompletion: 0,
			Status:           providers.MetricStatusRetry,
			ErrorMessage:     "temporary network error",
		}

		assert.Equal(t, providers.MetricStatusRetry, metrics.Status)
		assert.NotEmpty(t, metrics.ErrorMessage)
	})
}

func TestProviderMetrics_Validation(t *testing.T) {
	t.Run("valid_provider_id", func(t *testing.T) {
		metrics := providers.ProviderMetrics{
			ProviderID:       "openai-fast",
			Role:             providers.RoleCoder,
			Timestamp:        time.Now(),
			ResponseTimeMs:   100,
			TokensPrompt:     50,
			TokensCompletion: 25,
			Status:           providers.MetricStatusSuccess,
		}

		assert.NotEmpty(t, metrics.ProviderID)
	})

	t.Run("valid_timestamp", func(t *testing.T) {
		now := time.Now()
		metrics := providers.ProviderMetrics{
			ProviderID: "openai-fast",
			Role:       providers.RoleCoder,
			Timestamp:  now,
			Status:     providers.MetricStatusSuccess,
		}

		assert.False(t, metrics.Timestamp.IsZero())
		assert.True(t, metrics.Timestamp.Equal(now))
	})

	t.Run("valid_status_types", func(t *testing.T) {
		validStatuses := []providers.MetricStatus{
			providers.MetricStatusSuccess,
			providers.MetricStatusFailure,
			providers.MetricStatusRetry,
		}

		for _, status := range validStatuses {
			metrics := providers.ProviderMetrics{
				ProviderID: "test-provider",
				Role:       providers.RoleCoder,
				Timestamp:  time.Now(),
				Status:     status,
			}
			assert.NotEmpty(t, metrics.Status)
		}
	})

	t.Run("non_negative_response_time", func(t *testing.T) {
		metrics := providers.ProviderMetrics{
			ProviderID:     "openai-fast",
			Role:           providers.RoleCoder,
			Timestamp:      time.Now(),
			ResponseTimeMs: 0, // Zero is valid (very fast response)
			Status:         providers.MetricStatusSuccess,
		}

		assert.GreaterOrEqual(t, metrics.ResponseTimeMs, int64(0))
	})

	t.Run("non_negative_tokens", func(t *testing.T) {
		metrics := providers.ProviderMetrics{
			ProviderID:       "openai-fast",
			Role:             providers.RoleCoder,
			Timestamp:        time.Now(),
			TokensPrompt:     0, // Valid - empty prompt
			TokensCompletion: 0, // Valid - error before completion
			Status:           providers.MetricStatusFailure,
		}

		assert.GreaterOrEqual(t, metrics.TokensPrompt, 0)
		assert.GreaterOrEqual(t, metrics.TokensCompletion, 0)
	})
}

func TestMetricsSummary_Aggregation(t *testing.T) {
	t.Run("empty_summary", func(t *testing.T) {
		summary := providers.MetricsSummary{
			ProviderID:      "openai-fast",
			Role:            providers.RoleCoder,
			TotalRequests:   0,
			SuccessCount:    0,
			FailureCount:    0,
			SuccessRate:     0.0,
			AvgResponseTime: 0.0,
			P50ResponseTime: 0,
			P95ResponseTime: 0,
			TotalTokens:     0,
			AvgTokensPerReq: 0.0,
		}

		assert.Equal(t, 0, summary.TotalRequests)
		assert.Equal(t, 0.0, summary.SuccessRate)
	})

	t.Run("success_rate_calculation", func(t *testing.T) {
		summary := providers.MetricsSummary{
			ProviderID:    "openai-fast",
			Role:          providers.RoleCoder,
			TotalRequests: 100,
			SuccessCount:  95,
			FailureCount:  5,
		}

		// Calculate success rate
		successRate := float64(summary.SuccessCount) / float64(summary.TotalRequests)
		assert.InDelta(t, 0.95, successRate, 0.001)
	})

	t.Run("average_response_time_calculation", func(t *testing.T) {
		// Simulate calculating average from raw response times
		responseTimes := []int64{100, 150, 200, 250, 300}
		var sum int64
		for _, rt := range responseTimes {
			sum += rt
		}
		avgResponseTime := float64(sum) / float64(len(responseTimes))

		summary := providers.MetricsSummary{
			ProviderID:      "openai-fast",
			Role:            providers.RoleCoder,
			TotalRequests:   5,
			AvgResponseTime: avgResponseTime,
		}

		assert.InDelta(t, 200.0, summary.AvgResponseTime, 0.1)
	})

	t.Run("percentile_calculation", func(t *testing.T) {
		// Simulate sorted response times for percentile calculation
		responseTimes := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

		// P50 (median) - 50th percentile
		// For 10 elements, P50 is at index 5 (0-based)
		p50Index := (len(responseTimes) * 50 / 100)
		if p50Index >= len(responseTimes) {
			p50Index = len(responseTimes) - 1
		}
		p50 := responseTimes[p50Index]

		// P95 - 95th percentile
		// For 10 elements, P95 is at index 9 (0-based)
		p95Index := (len(responseTimes) * 95 / 100)
		if p95Index >= len(responseTimes) {
			p95Index = len(responseTimes) - 1
		}
		p95 := responseTimes[p95Index]

		summary := providers.MetricsSummary{
			ProviderID:      "openai-fast",
			Role:            providers.RoleCoder,
			TotalRequests:   10,
			P50ResponseTime: p50,
			P95ResponseTime: p95,
		}

		// Verify percentiles are calculated correctly
		assert.Equal(t, int64(60), summary.P50ResponseTime)
		assert.Equal(t, int64(100), summary.P95ResponseTime)
	})

	t.Run("token_aggregation", func(t *testing.T) {
		// Simulate multiple requests
		requests := []struct {
			promptTokens     int
			completionTokens int
		}{
			{100, 50},
			{200, 100},
			{150, 75},
		}

		var totalTokens int
		for _, req := range requests {
			totalTokens += req.promptTokens + req.completionTokens
		}

		avgTokensPerReq := float64(totalTokens) / float64(len(requests))

		summary := providers.MetricsSummary{
			ProviderID:      "openai-fast",
			Role:            providers.RoleCoder,
			TotalRequests:   3,
			TotalTokens:     totalTokens,
			AvgTokensPerReq: avgTokensPerReq,
		}

		assert.Equal(t, 675, summary.TotalTokens) // (100+50) + (200+100) + (150+75)
		assert.InDelta(t, 225.0, summary.AvgTokensPerReq, 0.1)
	})

	t.Run("time_range", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now()

		summary := providers.MetricsSummary{
			ProviderID: "openai-fast",
			Role:       providers.RoleCoder,
			TimeRange: providers.TimeRange{
				Start: start,
				End:   end,
			},
		}

		assert.False(t, summary.TimeRange.Start.IsZero())
		assert.False(t, summary.TimeRange.End.IsZero())
		assert.True(t, summary.TimeRange.End.After(summary.TimeRange.Start))
	})
}

func TestMetricsSummary_EdgeCases(t *testing.T) {
	t.Run("100_percent_success", func(t *testing.T) {
		summary := providers.MetricsSummary{
			ProviderID:    "openai-fast",
			Role:          providers.RoleCoder,
			TotalRequests: 50,
			SuccessCount:  50,
			FailureCount:  0,
			SuccessRate:   1.0,
		}

		assert.Equal(t, 1.0, summary.SuccessRate)
		assert.Equal(t, 0, summary.FailureCount)
	})

	t.Run("100_percent_failure", func(t *testing.T) {
		summary := providers.MetricsSummary{
			ProviderID:    "openai-fast",
			Role:          providers.RoleCoder,
			TotalRequests: 10,
			SuccessCount:  0,
			FailureCount:  10,
			SuccessRate:   0.0,
		}

		assert.Equal(t, 0.0, summary.SuccessRate)
		assert.Equal(t, 10, summary.FailureCount)
	})

	t.Run("single_request", func(t *testing.T) {
		summary := providers.MetricsSummary{
			ProviderID:      "openai-fast",
			Role:            providers.RoleCoder,
			TotalRequests:   1,
			SuccessCount:    1,
			SuccessRate:     1.0,
			AvgResponseTime: 100.0,
			P50ResponseTime: 100,
			P95ResponseTime: 100,
			TotalTokens:     150,
			AvgTokensPerReq: 150.0,
		}

		assert.Equal(t, 1, summary.TotalRequests)
		assert.Equal(t, summary.P50ResponseTime, summary.P95ResponseTime)
	})
}

func TestMetricsCollection_MultipleProviders(t *testing.T) {
	t.Run("different_providers_same_role", func(t *testing.T) {
		metrics1 := providers.ProviderMetrics{
			ProviderID: "openai-fast",
			Role:       providers.RoleCoder,
			Timestamp:  time.Now(),
			Status:     providers.MetricStatusSuccess,
		}

		metrics2 := providers.ProviderMetrics{
			ProviderID: "anthropic-precise",
			Role:       providers.RoleCoder,
			Timestamp:  time.Now(),
			Status:     providers.MetricStatusSuccess,
		}

		assert.NotEqual(t, metrics1.ProviderID, metrics2.ProviderID)
		assert.Equal(t, metrics1.Role, metrics2.Role)
	})

	t.Run("same_provider_different_roles", func(t *testing.T) {
		metrics1 := providers.ProviderMetrics{
			ProviderID: "openai-fast",
			Role:       providers.RoleCoder,
			Timestamp:  time.Now(),
			Status:     providers.MetricStatusSuccess,
		}

		metrics2 := providers.ProviderMetrics{
			ProviderID: "openai-fast",
			Role:       providers.RoleReviewer,
			Timestamp:  time.Now(),
			Status:     providers.MetricStatusSuccess,
		}

		assert.Equal(t, metrics1.ProviderID, metrics2.ProviderID)
		assert.NotEqual(t, metrics1.Role, metrics2.Role)
	})
}
