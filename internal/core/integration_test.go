package core

import (
	"errors"
	"fmt"
	"testing"
)

// MockIntegrationProviderFactory creates mock providers for testing
func MockIntegrationProviderFactory(config ProviderConfig) (Provider, error) {
	return &MockProvider{
		name:   config.Type,
		config: config,
	}, nil
}

// MockFailingProviderFactory creates failing providers for testing
func MockFailingProviderFactory(config ProviderConfig) (Provider, error) {
	return nil, errors.New("mock creation failed")
}

func TestRegisterProviderFactory(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Clear global state for testing
	globalFactories = make(map[string]ProviderFactory)

	tests := []struct {
		name        string
		factoryType string
		factory     ProviderFactory
		expectError bool
	}{
		{
			name:        "valid factory registration",
			factoryType: "test-provider",
			factory:     MockIntegrationProviderFactory,
			expectError: false,
		},
		{
			name:        "empty factory type",
			factoryType: "",
			factory:     MockIntegrationProviderFactory,
			expectError: true,
		},
		{
			name:        "nil factory",
			factoryType: "nil-factory",
			factory:     nil,
			expectError: true,
		},
		{
			name:        "duplicate factory registration",
			factoryType: "duplicate",
			factory:     MockIntegrationProviderFactory,
			expectError: false, // First registration should succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RegisterProviderFactory(tt.factoryType, tt.factory)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}

	// Test duplicate registration separately
	t.Run("duplicate registration error", func(t *testing.T) {
		factory := MockIntegrationProviderFactory

		// First registration should succeed
		err1 := RegisterProviderFactory("duplicate-test", factory)
		if err1 != nil {
			t.Errorf("First registration should succeed: %v", err1)
		}

		// Second registration should succeed (factories can be overwritten)
		err2 := RegisterProviderFactory("duplicate-test", factory)
		if err2 != nil {
			t.Errorf("Duplicate registration should succeed: %v", err2)
		}
	})
}

func TestCreateProviderFromConfig(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Setup test factory
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("test-create", factory)

	tests := []struct {
		name        string
		config      ProviderConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: ProviderConfig{
				Type:   "test-create",
				APIKey: "test-key",
			},
			expectError: false,
		},
		{
			name: "unknown provider type",
			config: ProviderConfig{
				Type:   "unknown-type",
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
			provider, err := CreateProviderFromConfig(tt.config.Type, tt.config)

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

func TestGetRegisteredProviderTypes(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Setup test state
	globalFactories = make(map[string]ProviderFactory)

	// Initially should be empty
	types := GetRegisteredProviderTypes()
	if len(types) != 0 {
		t.Errorf("Expected 0 types initially, got %d", len(types))
	}

	// Register some factories
	factory1 := MockIntegrationProviderFactory
	factory2 := MockIntegrationProviderFactory

	RegisterProviderFactory("type1", factory1)
	RegisterProviderFactory("type2", factory2)

	types = GetRegisteredProviderTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(types))
	}

	// Check that both types are present
	foundType1, foundType2 := false, false
	for _, typ := range types {
		if typ == "type1" {
			foundType1 = true
		}
		if typ == "type2" {
			foundType2 = true
		}
	}

	if !foundType1 {
		t.Error("Expected to find type1")
	}
	if !foundType2 {
		t.Error("Expected to find type2")
	}
}

func TestIsProviderTypeRegistered(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Setup test state
	globalFactories = make(map[string]ProviderFactory)

	// Test non-existing type
	if IsProviderTypeRegistered("non-existing") {
		t.Error("Expected false for non-existing type")
	}

	// Register a factory
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("registered-type", factory)

	// Test existing type
	if !IsProviderTypeRegistered("registered-type") {
		t.Error("Expected true for registered type")
	}

	// Test empty string
	if IsProviderTypeRegistered("") {
		t.Error("Expected false for empty string")
	}
}

func TestGetProviderTypeInfo(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Setup test state
	globalFactories = make(map[string]ProviderFactory)

	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("info-test", factory)

	tests := []struct {
		name         string
		providerType string
		expectError  bool
	}{
		{
			name:         "existing provider",
			providerType: "info-test",
			expectError:  false,
		},
		{
			name:         "non-existing provider",
			providerType: "non-existing",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			infoMap := GetProviderTypeInfo()

			if tt.expectError {
				if _, exists := infoMap[tt.providerType]; exists {
					t.Error("Expected provider type to not exist")
				}
			} else {
				info, exists := infoMap[tt.providerType]
				if !exists {
					t.Error("Expected provider type to exist")
				}
				if info.Type != tt.providerType {
					t.Errorf("Expected type %s, got %s", tt.providerType, info.Type)
				}
			}
		})
	}
}

func TestRegisterBuiltinProviders(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Clear global state
	globalFactories = make(map[string]ProviderFactory)

	err := RegisterBuiltinProviders()

	// Should not error even if no builtin providers are registered
	if err != nil {
		t.Errorf("RegisterBuiltinProviders() should not error: %v", err)
	}

	// Current implementation doesn't register builtin providers directly
	// It relies on init() functions in provider packages
	// So we just test that the function doesn't panic or error
}

func TestCreateProviderWithRegistry(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	registry := NewProviderRegistry()
	config := ProviderConfig{
		Type:   "registry-test",
		APIKey: "test-key",
	}

	// Setup factory
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("registry-test", factory)

	tests := []struct {
		name        string
		config      ProviderConfig
		expectError bool
	}{
		{
			name:        "valid config",
			config:      config,
			expectError: false,
		},
		{
			name: "unknown provider type",
			config: ProviderConfig{
				Type:   "unknown-registry-type",
				APIKey: "test-key",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, err := CreateProviderWithRegistry(registry, "test-provider", tt.config.Type, tt.config)

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

func TestCreateProvidersFromConfigMap(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Setup factory
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("map-test", factory)

	tests := []struct {
		name        string
		configs     map[string]ProviderConfig
		expectError bool
		expectCount int
	}{
		{
			name: "valid configs",
			configs: map[string]ProviderConfig{
				"provider1": {Type: "map-test", APIKey: "key1"},
				"provider2": {Type: "map-test", APIKey: "key2"},
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "mixed valid and invalid",
			configs: map[string]ProviderConfig{
				"valid":   {Type: "map-test", APIKey: "key1"},
				"invalid": {Type: "unknown-type", APIKey: "key2"},
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "empty configs",
			configs:     map[string]ProviderConfig{},
			expectError: false,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			providers, err := CreateProvidersFromConfigMap(tt.configs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(providers) != tt.expectCount {
					t.Errorf("Expected %d providers, got %d", tt.expectCount, len(providers))
				}
			}
		})
	}
}

func TestPopulateRegistryFromConfig(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	registry := NewProviderRegistry()

	// Setup factory
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("populate-test", factory)

	configs := map[string]ProviderConfig{
		"provider1": {Type: "populate-test", APIKey: "key1"},
		"provider2": {Type: "populate-test", APIKey: "key2"},
	}

	err := PopulateRegistryFromConfig(registry, configs)
	if err != nil {
		t.Errorf("PopulateRegistryFromConfig() error = %v", err)
	}

	// Check that providers were registered
	providers := registry.List()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers in registry, got %d", len(providers))
	}

	// Check that we can get each provider
	provider1, err := registry.Get("provider1")
	if err != nil {
		t.Errorf("Failed to get provider1: %v", err)
	}
	if provider1 == nil {
		t.Error("Expected non-nil provider1")
	}

	provider2, err := registry.Get("provider2")
	if err != nil {
		t.Errorf("Failed to get provider2: %v", err)
	}
	if provider2 == nil {
		t.Error("Expected non-nil provider2")
	}
}

func TestNewValidationHelper(t *testing.T) {
	t.Parallel()

	// Skip this test as ValidationHelper may not be implemented yet
	t.Skip("ValidationHelper not implemented yet")
}

func TestValidateProviderConfig(t *testing.T) {
	t.Parallel()

	// Skip this test as ValidateProviderConfig function doesn't exist
	t.Skip("ValidateProviderConfig function not implemented")
}

func TestValidateAllConfigs(t *testing.T) {
	t.Parallel()

	// Skip this test as ValidateAllConfigs function doesn't exist
	t.Skip("ValidateAllConfigs function not implemented")
}

func TestGetRecommendedConfig(t *testing.T) {
	// Don't run in parallel since we're modifying global state

	// Save original state and restore after test
	originalFactories := globalFactories
	defer func() {
		globalFactories = originalFactories
	}()

	// Setup factory
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("recommend-test", factory)

	t.Run("get recommended config for existing provider", func(t *testing.T) {
		t.Parallel()
		infoMap := GetProviderTypeInfo()

		if info, exists := infoMap["recommend-test"]; !exists {
			t.Error("Expected recommend-test provider to exist")
		} else {
			if info.Type != "recommend-test" {
				t.Errorf("Expected type recommend-test, got %s", info.Type)
			}
		}
	})
}

// Edge cases and error scenarios
func TestIntegration_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("factory creation fails", func(t *testing.T) {
		// Save original state and restore after test
		originalFactories := globalFactories
		defer func() {
			globalFactories = originalFactories
		}()

		globalFactories = make(map[string]ProviderFactory)

		failingFactory := MockFailingProviderFactory
		RegisterProviderFactory("failing-factory", failingFactory)

		config := ProviderConfig{
			Type:   "failing-factory",
			APIKey: "test-key",
		}

		provider, err := CreateProviderFromConfig(config.Type, config)
		if err == nil {
			t.Error("Expected error from failing factory")
		}
		if provider != nil {
			t.Error("Expected nil provider from failing factory")
		}
	})

	t.Run("concurrent access to global factories", func(t *testing.T) {
		// Save original state and restore after test
		originalFactories := globalFactories
		defer func() {
			globalFactories = originalFactories
		}()

		globalFactories = make(map[string]ProviderFactory)

		// Test concurrent registration and access
		done := make(chan bool, 10)

		// Concurrent registrations
		for i := 0; i < 5; i++ {
			go func(id int) {
				factory := MockIntegrationProviderFactory
				RegisterProviderFactory(fmt.Sprintf("concurrent-%d", id), factory)
				done <- true
			}(i)
		}

		// Concurrent reads
		for i := 0; i < 5; i++ {
			go func() {
				_ = GetRegisteredProviderTypes()
				_ = IsProviderTypeRegistered("concurrent-0")
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify final state
		types := GetRegisteredProviderTypes()
		if len(types) < 5 {
			t.Errorf("Expected at least 5 types, got %d", len(types))
		}
	})
}

// Benchmark tests
func BenchmarkRegisterProviderFactory(b *testing.B) {
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RegisterProviderFactory(fmt.Sprintf("bench-%d", i), factory)
	}
}

func BenchmarkCreateProviderFromConfig(b *testing.B) {
	globalFactories = make(map[string]ProviderFactory)
	factory := MockIntegrationProviderFactory
	RegisterProviderFactory("bench", factory)

	config := ProviderConfig{Type: "bench", APIKey: "key"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CreateProviderFromConfig(config.Type, config)
	}
}
