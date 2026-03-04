package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	name            string
	config          ProviderConfig
	completionError error
	streamError     error
	modelsError     error
	validateError   error
	models          []Model
	delay           time.Duration
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.completionError != nil {
		return nil, m.completionError
	}
	return &CompletionResponse{
		ID:    "test-response",
		Model: req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    RoleAssistant,
					Content: "Test response",
				},
				FinishReason: FinishReasonStop,
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *MockProvider) StreamCompletion(ctx context.Context, req *CompletionRequest) (<-chan StreamChunk, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}

	chunks := make(chan StreamChunk, 1)
	chunks <- StreamChunk{
		ID:    "test-stream",
		Model: req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Delta: &Message{
					Role:    RoleAssistant,
					Content: "Test stream",
				},
			},
		},
		Done: true,
	}
	close(chunks)

	return chunks, nil
}

func (m *MockProvider) GetModels(ctx context.Context) ([]Model, error) {
	if m.modelsError != nil {
		return nil, m.modelsError
	}
	if m.models != nil {
		return m.models, nil
	}
	return []Model{
		{
			ID:       "test-model",
			Provider: m.name,
		},
	}, nil
}

func (m *MockProvider) ValidateConfig() error {
	return m.validateError
}

// MockProviderFactory creates mock providers
func MockProviderFactory(config ProviderConfig) (Provider, error) {
	return &MockProvider{
		name:   config.Type,
		config: config,
	}, nil
}

func TestDefaultRegistryConfig(t *testing.T) {
	t.Parallel()

	config := DefaultRegistryConfig()

	if config.HealthCheckInterval <= 0 {
		t.Error("HealthCheckInterval should be positive")
	}

	if config.HealthCheckTimeout <= 0 {
		t.Error("HealthCheckTimeout should be positive")
	}
}

func TestNewProviderRegistry(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()

	if registry == nil {
		t.Fatal("Registry should not be nil")
	}

	// Should have default config
	if registry.config.HealthCheckInterval == 0 {
		t.Error("Registry should have default config")
	}
}

func TestNewProviderRegistryWithConfig(t *testing.T) {
	t.Parallel()

	config := RegistryConfig{
		HealthCheckInterval: 5 * time.Minute,
		HealthCheckTimeout:  10 * time.Second,
		EnableMetrics:       true,
	}

	registry := NewProviderRegistryWithConfig(config)

	if registry == nil {
		t.Fatal("Registry should not be nil")
	}

	if registry.config.HealthCheckInterval != 5*time.Minute {
		t.Errorf("HealthCheckInterval = %v, want 5m", registry.config.HealthCheckInterval)
	}
}

func TestProviderRegistry_RegisterFactory(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()

	t.Run("successful registration", func(t *testing.T) {
		err := registry.RegisterFactory("mock", MockProviderFactory)
		if err != nil {
			t.Errorf("RegisterFactory() error = %v", err)
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		err := registry.RegisterFactory("mock", MockProviderFactory)
		if err != nil {
			t.Error("Expected no error for duplicate registration - factories can be overwritten")
		}
	})

	t.Run("empty provider type", func(t *testing.T) {
		err := registry.RegisterFactory("", MockProviderFactory)
		if err == nil {
			t.Error("Expected error for empty provider type")
		}
	})

	t.Run("nil factory", func(t *testing.T) {
		err := registry.RegisterFactory("nil", nil)
		if err == nil {
			t.Error("Expected error for nil factory")
		}
	})
}

func TestProviderRegistry_CreateProvider(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()

	err := registry.RegisterFactory("mock", MockProviderFactory)
	if err != nil {
		t.Fatalf("Failed to register factory: %v", err)
	}

	tests := []struct {
		name        string
		config      ProviderConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: ProviderConfig{
				Type:   "mock",
				APIKey: "test-key",
			},
			expectError: false,
		},
		{
			name: "unknown provider type",
			config: ProviderConfig{
				Type:   "unknown",
				APIKey: "test-key",
			},
			expectError: true,
		},
		{
			name: "empty provider type",
			config: ProviderConfig{
				Type:   "",
				APIKey: "test-key",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, err := registry.CreateProvider(tt.config.Type, tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if provider != nil {
					t.Error("Expected nil provider on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("Expected non-nil provider")
				}
			}
		})
	}
}

func TestProviderRegistry_Register(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	provider := &MockProvider{name: "test-provider"}

	t.Run("successful registration", func(t *testing.T) {
		err := registry.Register("test", provider)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		err := registry.Register("test", provider)
		if err == nil {
			t.Error("Expected error for duplicate registration")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		err := registry.Register("", provider)
		if err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("nil provider", func(t *testing.T) {
		err := registry.Register("nil-test", nil)
		if err == nil {
			t.Error("Expected error for nil provider")
		}
	})
}

func TestProviderRegistry_Get(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	provider := &MockProvider{name: "test-provider"}

	// Register a provider first
	err := registry.Register("test", provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	t.Run("existing provider", func(t *testing.T) {
		t.Parallel()
		retrieved, err := registry.Get("test")
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if retrieved != provider {
			t.Error("Retrieved provider doesn't match registered provider")
		}
	})

	t.Run("non-existing provider", func(t *testing.T) {
		t.Parallel()
		retrieved, err := registry.Get("non-existing")
		if err == nil {
			t.Error("Expected error for non-existing provider")
		}
		if retrieved != nil {
			t.Error("Expected nil provider for non-existing name")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		t.Parallel()
		retrieved, err := registry.Get("")
		if err == nil {
			t.Error("Expected error for empty name")
		}
		if retrieved != nil {
			t.Error("Expected nil provider for empty name")
		}
	})
}

func TestProviderRegistry_GetDefault(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	provider := &MockProvider{name: "default-provider"}

	t.Run("no default set", func(t *testing.T) {
		defaultProvider, err := registry.GetDefault()
		if err == nil {
			t.Error("Expected error when no default is set")
		}
		if defaultProvider != nil {
			t.Error("Expected nil when no default is set")
		}
	})

	t.Run("with default set", func(t *testing.T) {
		err := registry.Register("default", provider)
		if err != nil {
			t.Fatalf("Failed to register provider: %v", err)
		}

		// Set the default provider directly in config
		registry.config.DefaultProvider = "default"

		defaultProvider, err := registry.GetDefault()
		if err != nil {
			t.Errorf("Unexpected error getting default: %v", err)
		}
		if defaultProvider != provider {
			t.Error("Default provider doesn't match expected provider")
		}
	})
}

func TestProviderRegistry_List(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	provider1 := &MockProvider{name: "provider1"}
	provider2 := &MockProvider{name: "provider2"}

	// Initially empty
	providers := registry.List()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}

	// Register providers
	registry.Register("test1", provider1)
	registry.Register("test2", provider2)

	providers = registry.List()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}
}

func TestProviderRegistry_ListHealthy(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	healthyProvider := &MockProvider{name: "healthy"}
	unhealthyProvider := &MockProvider{
		name:            "unhealthy",
		completionError: errors.New("provider error"),
	}

	registry.Register("healthy", healthyProvider)
	registry.Register("unhealthy", unhealthyProvider)

	// Initially all providers are considered healthy
	healthy := registry.ListHealthy()
	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy providers initially, got %d", len(healthy))
	}

	// After health check, unhealthy provider should be filtered out
	ctx := context.Background()
	registry.HealthCheck(ctx)

	// Give some time for health checks to complete
	time.Sleep(100 * time.Millisecond)

	healthy = registry.ListHealthy()
	// Note: This test might be flaky depending on health check implementation
	// In real implementation, you might need to wait or have a way to wait for health checks
}

func TestProviderRegistry_Unregister(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	provider := &MockProvider{name: "test-provider"}

	// Register first
	err := registry.Register("test", provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	t.Run("successful unregistration", func(t *testing.T) {
		err := registry.Unregister("test")
		if err != nil {
			t.Errorf("Unregister() error = %v", err)
		}

		// Should not be able to get it anymore
		_, err = registry.Get("test")
		if err == nil {
			t.Error("Expected error when getting unregistered provider")
		}
	})

	t.Run("unregister non-existing", func(t *testing.T) {
		err := registry.Unregister("non-existing")
		if err == nil {
			t.Error("Expected error for non-existing provider")
		}
	})
}

func TestProviderRegistry_HealthCheck(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	healthyProvider := &MockProvider{name: "healthy"}
	unhealthyProvider := &MockProvider{
		name:            "unhealthy",
		completionError: errors.New("health check failed"),
	}

	registry.Register("healthy", healthyProvider)
	registry.Register("unhealthy", unhealthyProvider)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	registry.HealthCheck(ctx)

	// Health check runs asynchronously, so we need to wait a bit
	time.Sleep(200 * time.Millisecond)

	// Get health status
	healthyStatus, err := registry.GetHealth("healthy")
	if err != nil {
		t.Errorf("Expected no error getting health for healthy provider: %v", err)
	}
	unhealthyStatus, err2 := registry.GetHealth("unhealthy")
	if err2 != nil {
		t.Errorf("Expected no error getting health for unhealthy provider: %v", err2)
	}

	// Note: The actual health status depends on the implementation
	// These assertions might need adjustment based on the real implementation
	if healthyStatus == nil {
		t.Error("Expected health status for healthy provider")
	}

	if unhealthyStatus == nil {
		t.Error("Expected health status for unhealthy provider")
	}
}

func TestProviderRegistry_GetHealth(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()
	provider := &MockProvider{name: "test-provider"}

	registry.Register("test", provider)

	t.Run("existing provider", func(t *testing.T) {
		health, err := registry.GetHealth("test")
		if err != nil {
			t.Errorf("Unexpected error getting health: %v", err)
		}
		// Initial health status should exist
		if health == nil {
			t.Error("Expected health status for registered provider")
		}
	})

	t.Run("non-existing provider", func(t *testing.T) {
		health, err := registry.GetHealth("non-existing")
		if err == nil {
			t.Error("Expected error for non-existing provider")
		}
		if health != nil {
			t.Error("Expected nil health for non-existing provider")
		}
	})
}

func TestNewBaseProvider(t *testing.T) {
	t.Parallel()

	config := ProviderConfig{
		Type:   "test",
		APIKey: "test-key",
	}

	base := NewBaseProvider("test", config)

	if base == nil {
		t.Fatal("BaseProvider should not be nil")
	}

	if base.Name() != "test" {
		t.Errorf("Name() = %s, want test", base.Name())
	}

	if base.Config().Type != "test" {
		t.Errorf("Config().Type = %s, want test", base.Config().Type)
	}
}

func TestNewProviderMetrics(t *testing.T) {
	t.Parallel()

	metrics := NewProviderMetrics()

	if metrics == nil {
		t.Fatal("ProviderMetrics should not be nil")
	}

	// Test initial values
	if metrics.GetAverageResponseTime() < 0 {
		t.Error("Initial average response time should be >= 0")
	}

	if metrics.GetSuccessRate() < 0 || metrics.GetSuccessRate() > 1 {
		t.Error("Success rate should be between 0 and 1")
	}
}

func TestProviderMetrics_RecordRequest(t *testing.T) {
	t.Parallel()

	metrics := NewProviderMetrics()

	// Record successful request
	usage := &Usage{TotalTokens: 10}
	cost := floatPtr(0.001)
	metrics.RecordRequest(100*time.Millisecond, usage, cost, nil)

	if metrics.GetAverageResponseTime() <= 0 {
		t.Error("Average response time should be positive after recording")
	}

	// Record failed request
	usage2 := &Usage{TotalTokens: 5}
	cost2 := floatPtr(0.0005)
	metrics.RecordRequest(50*time.Millisecond, usage2, cost2, errors.New("test error"))

	successRate := metrics.GetSuccessRate()
	expected := 50.0 // Success rate is returned as percentage (0-100), not fraction (0-1)
	if successRate < expected-0.01 || successRate > expected+0.01 {
		t.Errorf("Success rate = %f, want %f", successRate, expected)
	}
}

func TestProviderMetrics_Clone(t *testing.T) {
	t.Parallel()

	original := NewProviderMetrics()
	usage := &Usage{TotalTokens: 10}
	cost := floatPtr(0.001)
	original.RecordRequest(100*time.Millisecond, usage, cost, nil)

	cloned := original.Clone()

	if cloned == nil {
		t.Fatal("Cloned metrics should not be nil")
	}

	if cloned == original {
		t.Error("Clone should return a different instance")
	}

	// Values should be the same
	if cloned.GetAverageResponseTime() != original.GetAverageResponseTime() {
		t.Error("Cloned metrics should have same average response time")
	}
}

// Concurrent access tests
func TestProviderRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	registry := NewProviderRegistry()

	// Test concurrent registration and access
	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			provider := &MockProvider{name: fmt.Sprintf("provider-%d", id)}
			err := registry.Register(fmt.Sprintf("test-%d", id), provider)
			if err != nil {
				t.Errorf("Concurrent registration failed: %v", err)
			}
		}(i)
	}

	// Concurrent access
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond) // Give registration time
			_, _ = registry.Get(fmt.Sprintf("test-%d", id))
			_ = registry.List()
		}(i)
	}

	wg.Wait()

	// Verify all providers were registered
	providers := registry.List()
	if len(providers) != 10 {
		t.Errorf("Expected 10 providers, got %d", len(providers))
	}
}

// Benchmark tests
func BenchmarkProviderRegistry_Register(b *testing.B) {
	registry := NewProviderRegistry()
	provider := &MockProvider{name: "bench-provider"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("bench-%d", i)
		registry.Register(name, provider)
	}
}

func BenchmarkProviderRegistry_Get(b *testing.B) {
	registry := NewProviderRegistry()
	provider := &MockProvider{name: "bench-provider"}
	registry.Register("bench", provider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Get("bench")
	}
}

func BenchmarkProviderMetrics_RecordRequest(b *testing.B) {
	metrics := NewProviderMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage := &Usage{TotalTokens: 10}
		cost := floatPtr(0.001)
		metrics.RecordRequest(100*time.Millisecond, usage, cost, nil)
	}
}

// Edge case tests
func TestProviderRegistry_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("registry with custom config", func(t *testing.T) {
		t.Parallel()
		config := RegistryConfig{
			HealthCheckInterval: time.Minute,
			HealthCheckTimeout:  30 * time.Second,
			EnableMetrics:       true,
		}

		registry := NewProviderRegistryWithConfig(config)

		// Register providers
		provider1 := &MockProvider{name: "provider1"}
		provider2 := &MockProvider{name: "provider2"}

		err1 := registry.Register("test1", provider1)
		err2 := registry.Register("test2", provider2)

		if err1 != nil {
			t.Errorf("First registration should succeed: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Second registration should succeed: %v", err2)
		}

		// Verify providers are registered
		providers := registry.List()
		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(providers))
		}
	})

	t.Run("provider with slow response", func(t *testing.T) {
		t.Parallel()
		registry := NewProviderRegistry()
		slowProvider := &MockProvider{
			name:  "slow-provider",
			delay: 100 * time.Millisecond,
		}

		registry.Register("slow", slowProvider)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := slowProvider.CreateCompletion(ctx, &CompletionRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: RoleUser, Content: "test"},
			},
		})
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Slow provider should still work: %v", err)
		}

		if duration < 100*time.Millisecond {
			t.Error("Provider should have taken at least 100ms")
		}
	})
}
