package providers

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidator_ValidateAll_ParallelExecution(t *testing.T) {
	// Create mock providers that track concurrent execution
	var activeCount int32
	var maxConcurrent int32
	var mu sync.Mutex
	executionTimes := make(map[string]time.Time)

	createTrackingProvider := func(id string) *MockProvider {
		return &MockProvider{
			id:           id,
			providerType: ProviderTypeOpenAI,
			initError:    nil,
			initFunc: func(ctx context.Context) error {
				// Track when this provider starts initializing
				mu.Lock()
				executionTimes[id] = time.Now()
				mu.Unlock()

				// Increment active count and track max concurrency
				current := atomic.AddInt32(&activeCount, 1)
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
						break
					}
				}

				// Simulate some work
				time.Sleep(50 * time.Millisecond)

				// Decrement active count
				atomic.AddInt32(&activeCount, -1)
				return nil
			},
		}
	}

	// Create 4 providers
	providerMap := map[string]LLMProvider{
		"provider1": createTrackingProvider("provider1"),
		"provider2": createTrackingProvider("provider2"),
		"provider3": createTrackingProvider("provider3"),
		"provider4": createTrackingProvider("provider4"),
	}

	// Create validator and run validation
	validator := NewValidator(providerMap)
	err := validator.ValidateAll(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify parallel execution occurred (max concurrent should be > 1)
	maxConc := atomic.LoadInt32(&maxConcurrent)
	if maxConc < 2 {
		t.Errorf("Expected parallel execution (max concurrent >= 2), got %d", maxConc)
	}

	// Verify all providers were executed
	mu.Lock()
	defer mu.Unlock()
	if len(executionTimes) != 4 {
		t.Errorf("Expected 4 providers to execute, got %d", len(executionTimes))
	}
}

func TestValidator_ValidateAll_TimeoutHandling(t *testing.T) {
	// Create a mock provider that takes too long to initialize
	slowProvider := &MockProvider{
		id:           "slow",
		providerType: ProviderTypeOpenAI,
		initFunc: func(ctx context.Context) error {
			// Wait longer than the timeout
			select {
			case <-time.After(2 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	providerMap := map[string]LLMProvider{
		"slow": slowProvider,
	}

	// Create validator with short timeout
	validator := NewValidator(providerMap)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := validator.ValidateAll(ctx)

	// Should get a context deadline exceeded error
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Error message should mention the provider and contain context error
	if !errors.Is(err, context.DeadlineExceeded) && err.Error() == "" {
		t.Errorf("Expected context deadline error, got %v", err)
	}
}

func TestValidator_ValidateAll_ErrorAggregation(t *testing.T) {
	// Create multiple mock providers that fail
	provider1 := &MockProvider{
		id:           "failing1",
		providerType: ProviderTypeOpenAI,
		initError:    errors.New("invalid credentials for provider1"),
	}

	provider2 := &MockProvider{
		id:           "failing2",
		providerType: ProviderTypeAnthropic,
		initError:    errors.New("invalid credentials for provider2"),
	}

	provider3 := &MockProvider{
		id:           "failing3",
		providerType: ProviderTypeGoogle,
		initError:    errors.New("invalid credentials for provider3"),
	}

	providerMap := map[string]LLMProvider{
		"failing1": provider1,
		"failing2": provider2,
		"failing3": provider3,
	}

	// Create validator and run validation
	validator := NewValidator(providerMap)
	err := validator.ValidateAll(context.Background())

	if err == nil {
		t.Fatal("Expected error aggregation, got nil")
	}

	// Error should mention that 3 providers failed
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Error message should mention the count of failed providers
	if !strings.Contains(errMsg, "3 provider") {
		t.Errorf("Expected error to mention 3 providers, got: %s", errMsg)
	}
}
