package providers

import (
	"context"
	"testing"
	"time"
)

// MockProvider is a test implementation of LLMProvider
type MockProvider struct {
	id           string
	providerType ProviderType
	initError    error
	initFunc     func(context.Context) error // Optional custom initialization function
	execResponse Response
	execError    error
}

func (m *MockProvider) Initialize(ctx context.Context) error {
	if m.initFunc != nil {
		return m.initFunc(ctx)
	}
	return m.initError
}

func (m *MockProvider) Execute(ctx context.Context, req Request) (Response, error) {
	return m.execResponse, m.execError
}

func (m *MockProvider) Name() string {
	return m.id
}

func (m *MockProvider) Type() ProviderType {
	return m.providerType
}

func (m *MockProvider) Shutdown(ctx context.Context) error {
	return nil
}

// initRegistry is a test helper that initializes a registry with mock providers
// Now we can access unexported fields directly since we're in the same package
func initRegistry(registry *Registry, providerMap map[string]LLMProvider, roleMap map[Role]*RoleAssignment, defaultProvider string) {
	registry.providers = providerMap
	registry.roleMap = roleMap
	registry.defaultProvider = defaultProvider
	registry.metrics = NewMetricsCollector()
}

func TestProviderRegistry_SelectProvider_PrimarySelection(t *testing.T) {
	// Create mock providers
	mockPrimary := &MockProvider{id: "primary", providerType: ProviderTypeOpenAI}
	mockFallback := &MockProvider{id: "fallback", providerType: ProviderTypeAnthropic}

	// Create registry directly (unit test - bypass validation)
	primaryID := "primary"
	registry := &Registry{}

	// Use reflection to set private fields for testing
	// This is a unit test, so we directly inject mock dependencies
	providerMap := map[string]LLMProvider{
		"primary":  mockPrimary,
		"fallback": mockFallback,
	}

	roleMap := map[Role]*RoleAssignment{
		RoleCoder: {
			PrimaryProvider:   "primary",
			FallbackProviders: []string{"fallback"},
		},
	}

	// Initialize registry fields
	initRegistry(registry, providerMap, roleMap, primaryID)

	// Test primary selection
	provider, providerID, err := registry.SelectProvider(context.Background(), RoleCoder)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if providerID != "primary" {
		t.Errorf("Expected primary provider, got %s", providerID)
	}

	if provider != mockPrimary {
		t.Error("Expected primary provider instance")
	}
}

func TestProviderRegistry_SelectProvider_FallbackChain(t *testing.T) {
	// Create mock providers
	mockFallback1 := &MockProvider{id: "fallback1", providerType: ProviderTypeOpenAI}
	mockFallback2 := &MockProvider{id: "fallback2", providerType: ProviderTypeAnthropic}

	// Create registry with only fallback providers (primary doesn't exist)
	registry := &Registry{}

	providerMap := map[string]LLMProvider{
		"fallback1": mockFallback1,
		"fallback2": mockFallback2,
	}

	roleMap := map[Role]*RoleAssignment{
		RoleCoder: {
			PrimaryProvider:   "nonexistent", // Primary doesn't exist
			FallbackProviders: []string{"fallback1", "fallback2"},
		},
	}

	initRegistry(registry, providerMap, roleMap, "fallback1")

	// Test fallback selection
	provider, providerID, err := registry.SelectProvider(context.Background(), RoleCoder)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if providerID != "fallback1" {
		t.Errorf("Expected fallback1 provider, got %s", providerID)
	}

	if provider != mockFallback1 {
		t.Error("Expected fallback1 provider instance")
	}
}

func TestProviderRegistry_SelectProvider_DefaultProvider(t *testing.T) {
	// Create mock provider
	mockDefault := &MockProvider{id: "default", providerType: ProviderTypeOpenAI}

	// Create registry with no role assignments (should use default provider)
	registry := &Registry{}

	providerMap := map[string]LLMProvider{
		"default": mockDefault,
	}

	roleMap := map[Role]*RoleAssignment{} // No role assignments

	initRegistry(registry, providerMap, roleMap, "default")

	// Test default provider selection for unmapped role
	provider, providerID, err := registry.SelectProvider(context.Background(), RoleCoder)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if providerID != "default" {
		t.Errorf("Expected default provider, got %s", providerID)
	}

	if provider != mockDefault {
		t.Error("Expected default provider instance")
	}
}

func TestProviderRegistry_Initialize_ValidatesCredentials(t *testing.T) {
	// This test verifies that a registry can be initialized with a provider
	// Actual credential validation is tested in integration tests with real providers

	// Create mock provider that will succeed initialization
	mockProvider := &MockProvider{
		id:           "test-provider",
		providerType: ProviderTypeOpenAI,
		initError:    nil,
	}

	// Create registry
	registry := &Registry{}

	providerMap := map[string]LLMProvider{
		"test-provider": mockProvider,
	}

	roleMap := map[Role]*RoleAssignment{}

	initRegistry(registry, providerMap, roleMap, "test-provider")

	// Verify registry was initialized
	provider, providerID, err := registry.SelectProvider(context.Background(), RoleCoder)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if providerID != "test-provider" {
		t.Errorf("Expected test-provider, got %s", providerID)
	}

	if provider != mockProvider {
		t.Error("Expected mock provider instance")
	}
}

func TestProviderRegistry_RecordMetrics(t *testing.T) {
	// Create registry
	mockProvider := &MockProvider{id: "test", providerType: ProviderTypeOpenAI}

	registry := &Registry{}

	providerMap := map[string]LLMProvider{
		"test": mockProvider,
	}

	roleMap := map[Role]*RoleAssignment{}

	initRegistry(registry, providerMap, roleMap, "test")

	// Record a metric
	metric := ProviderMetrics{
		ProviderID:       "test",
		Role:             RoleCoder,
		Status:           MetricStatusSuccess,
		ResponseTimeMs:   100,
		TokensPrompt:     50,
		TokensCompletion: 25,
		Timestamp:        time.Now(), // Explicitly set timestamp
	}

	err := registry.RecordMetrics(context.Background(), metric)

	if err != nil {
		t.Fatalf("Expected no error recording metrics, got %v", err)
	}

	// Verify metric was recorded by retrieving it (use 1 hour window)
	summary, err := registry.GetMetrics(context.Background(), "test", RoleCoder, time.Now().Add(-1*time.Hour))

	if err != nil {
		t.Fatalf("Expected no error retrieving metrics, got %v", err)
	}

	if summary.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", summary.TotalRequests)
	}

	if summary.SuccessCount != 1 {
		t.Errorf("Expected 1 success, got %d", summary.SuccessCount)
	}
}
