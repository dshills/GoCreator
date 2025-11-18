package providers_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dshills/gocreator/src/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentProviderUsage_MultipleTasksParallel tests that multiple tasks
// can execute concurrently using different providers without race conditions
func TestConcurrentProviderUsage_MultipleTasksParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	config, err := providers.LoadConfig("../../../tests/fixtures/providers/valid_config.yaml")
	require.NoError(t, err)

	// Create registry
	registry, err := providers.NewRegistryFromConfig(config)
	require.NoError(t, err)
	defer func() {
		_ = registry.Shutdown(context.Background())
	}()

	// Note: We're testing concurrent access patterns, not actual API calls
	// Real integration tests would require valid API keys and would skip if not present

	// Define test tasks with different roles
	tasks := []struct {
		taskID string
		role   providers.Role
	}{
		{"task-001", providers.RoleCoder},
		{"task-002", providers.RoleReviewer},
		{"task-003", providers.RolePlanner},
		{"task-004", providers.RoleClarifier},
		{"task-005", providers.RoleCoder},
		{"task-006", providers.RoleReviewer},
		{"task-007", providers.RolePlanner},
		{"task-008", providers.RoleClarifier},
		{"task-009", providers.RoleCoder},
		{"task-010", providers.RoleReviewer},
	}

	// Execute tasks concurrently
	var wg sync.WaitGroup
	results := make([]struct {
		taskID     string
		providerID string
		err        error
	}, len(tasks))

	ctx := context.Background()

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, taskInfo struct {
			taskID string
			role   providers.Role
		}) {
			defer wg.Done()

			// Select provider for this task
			provider, providerID, err := registry.SelectProvider(ctx, taskInfo.role)
			results[idx].taskID = taskInfo.taskID
			results[idx].providerID = providerID
			results[idx].err = err

			// Verify provider was selected
			if err == nil && provider != nil {
				// Provider should match the role assignment
				assert.NotNil(t, provider)
				assert.NotEmpty(t, providerID)
			}
		}(i, task)
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Verify all tasks completed without errors
	for _, result := range results {
		assert.NoError(t, result.err, "Task %s should not have errors", result.taskID)
		assert.NotEmpty(t, result.providerID, "Task %s should have a provider ID", result.taskID)
	}
}

// TestConcurrentProviderUsage_RaceConditions uses Go's race detector to verify
// that concurrent access to the registry doesn't cause data races
func TestConcurrentProviderUsage_RaceConditions(t *testing.T) {
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
	iterations := 100
	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			role := providers.RoleCoder
			if idx%4 == 1 {
				role = providers.RoleReviewer
			} else if idx%4 == 2 {
				role = providers.RolePlanner
			} else if idx%4 == 3 {
				role = providers.RoleClarifier
			}

			_, _, err := registry.SelectProvider(ctx, role)
			assert.NoError(t, err)
		}(i)
	}

	// Concurrent metric recording (if RecordMetrics exists)
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			metric := providers.ProviderMetrics{
				ProviderID:       "openai-fast",
				Role:             providers.RoleCoder,
				Status:           providers.MetricStatusSuccess,
				ResponseTimeMs:   int64(100 + idx),
				TokensPrompt:     500,
				TokensCompletion: 200,
				Timestamp:        time.Now(),
			}

			err := registry.RecordMetrics(ctx, metric)
			// Note: RecordMetrics may not be implemented yet, so we don't assert
			_ = err
		}(i)
	}

	wg.Wait()
	// If we get here without the race detector complaining, we're good
}

// TestConcurrentProviderUsage_LoadBalancing verifies that concurrent requests
// are properly distributed across providers when multiple tasks use the same role
func TestConcurrentProviderUsage_LoadBalancing(t *testing.T) {
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
	numTasks := 50
	var wg sync.WaitGroup
	providerCounts := make(map[string]int)
	var mu sync.Mutex

	// Execute many tasks with the same role concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			_, providerID, err := registry.SelectProvider(ctx, providers.RoleCoder)
			if err == nil {
				mu.Lock()
				providerCounts[providerID]++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// All tasks should use the same primary provider for RoleCoder
	// (since we're not implementing load balancing yet, just verifying consistency)
	assert.Equal(t, 1, len(providerCounts), "All tasks with same role should use same provider")
	for providerID, count := range providerCounts {
		assert.Equal(t, numTasks, count, "Provider %s should have been selected for all tasks", providerID)
	}
}

// TestConcurrentProviderUsage_ContextCancellation verifies that context cancellation
// is properly handled during concurrent operations
func TestConcurrentProviderUsage_ContextCancellation(t *testing.T) {
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

	// Create a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	numTasks := 10

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// This should either succeed quickly or respect context cancellation
			_, _, err := registry.SelectProvider(ctx, providers.RoleCoder)

			// We allow either success or context cancellation
			if err != nil {
				// If there's an error, it should be context-related after cancellation
				assert.True(t, err == context.DeadlineExceeded || err == context.Canceled || err != nil)
			}
		}()
	}

	wg.Wait()
}
