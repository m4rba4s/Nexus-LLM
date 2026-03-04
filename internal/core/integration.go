// Package core provides integration utilities for dynamic provider registration
// and discovery. This allows the CLI to automatically register and create
// providers based on configuration without hard-coding provider types.
//
// Example usage:
//
//	// Register all built-in providers
//	core.RegisterBuiltinProviders()
//
//	// Create provider from config
//	provider, err := core.CreateProviderFromConfig("openai", providerConfig)
//	if err != nil {
//		log.Fatal(err)
//	}
package core

import (
	"fmt"
	"strings"
)

// Using the global provider factories from global_registry.go

// CreateProviderFromConfig creates a provider instance from configuration
// using registered factories.
func CreateProviderFromConfig(providerType string, config ProviderConfig) (Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider config: %w", err)
	}

	factoriesMu.RLock()
	factory, exists := globalFactories[strings.ToLower(providerType)]
	if !exists {
		factory, exists = stableFactories[strings.ToLower(providerType)]
	}
	factoriesMu.RUnlock()

	if !exists {
		available := GetRegisteredProviderTypes()
		return nil, fmt.Errorf("no factory registered for provider type %q; available types: %s",
			providerType, strings.Join(available, ", "))
	}

	provider, err := factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %q: %w", providerType, err)
	}

	// Validate the created provider
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	return provider, nil
}

// GetRegisteredProviderTypes returns all registered provider types.
func GetRegisteredProviderTypes() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()

	types := make([]string, 0, len(globalFactories))
	for providerType := range globalFactories {
		types = append(types, providerType)
	}

	return types
}

// IsProviderTypeRegistered checks if a provider type is registered.
func IsProviderTypeRegistered(providerType string) bool {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()

	_, exists := globalFactories[strings.ToLower(providerType)]
	return exists
}

// ProviderInfo contains metadata about a registered provider type.
type ProviderTypeInfo struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Features    []string `json:"features,omitempty"`
}

// GetProviderTypeInfo returns information about registered provider types.
func GetProviderTypeInfo() map[string]ProviderTypeInfo {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()

	info := make(map[string]ProviderTypeInfo)

	// Add basic info for registered types
	// In a more advanced implementation, providers could register
	// their own metadata during registration
	for providerType := range globalFactories {
		info[providerType] = ProviderTypeInfo{
			Type:     providerType,
			Features: []string{"completion"}, // Basic features all providers support
		}
	}
	// Also include stable factories to avoid test interference
	for providerType := range stableFactories {
		if _, exists := info[providerType]; !exists {
			info[providerType] = ProviderTypeInfo{
				Type:     providerType,
				Features: []string{"completion"},
			}
		}
	}

	// Add specific metadata for known providers
	if _, exists := info["openai"]; exists {
		info["openai"] = ProviderTypeInfo{
			Type:        "openai",
			Description: "OpenAI GPT models including GPT-3.5, GPT-4, and variants",
			Features:    []string{"completion", "streaming", "function_calling", "vision"},
		}
	}

	if _, exists := info["anthropic"]; exists {
		info["anthropic"] = ProviderTypeInfo{
			Type:        "anthropic",
			Description: "Anthropic Claude models for advanced reasoning and analysis",
			Features:    []string{"completion", "streaming", "long_context"},
		}
	}

	if _, exists := info["ollama"]; exists {
		info["ollama"] = ProviderTypeInfo{
			Type:        "ollama",
			Description: "Local LLM execution via Ollama",
			Features:    []string{"completion", "streaming", "local_models"},
		}
	}

	return info
}

// RegisterBuiltinProviders registers all built-in provider factories.
// This should be called during application initialization.
func RegisterBuiltinProviders() error {
	// Note: The actual provider imports and registrations would happen here
	// For now, we'll register them when the provider packages are imported
	// via init() functions in each provider package.

	// This function serves as a central place to ensure all built-in
	// providers are registered. The actual registration happens in
	// init() functions of provider packages to avoid import cycles.

	return nil
}

// Enhanced factory creation helpers

// CreateProviderWithRegistry creates a provider and registers it with the given registry.
func CreateProviderWithRegistry(registry *ProviderRegistry, name, providerType string, config ProviderConfig) (Provider, error) {
	provider, err := CreateProviderFromConfig(providerType, config)
	if err != nil {
		return nil, err
	}

	if err := registry.Register(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register provider: %w", err)
	}

	return provider, nil
}

// CreateProvidersFromConfigMap creates multiple providers from a configuration map.
func CreateProvidersFromConfigMap(configs map[string]ProviderConfig) (map[string]Provider, error) {
	providers := make(map[string]Provider)

	for name, config := range configs {
		provider, err := CreateProviderFromConfig(config.Type, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %q: %w", name, err)
		}
		providers[name] = provider
	}

	return providers, nil
}

// PopulateRegistryFromConfig populates a registry with providers from configuration.
func PopulateRegistryFromConfig(registry *ProviderRegistry, configs map[string]ProviderConfig) error {
	for name, config := range configs {
		_, err := CreateProviderWithRegistry(registry, name, config.Type, config)
		if err != nil {
			return fmt.Errorf("failed to create and register provider %q: %w", name, err)
		}
	}

	return nil
}

// ValidationHelper provides validation utilities for provider configurations.
type ValidationHelper struct {
	registry *ProviderRegistry
}

// NewValidationHelper creates a new validation helper.
func NewValidationHelper(registry *ProviderRegistry) *ValidationHelper {
	return &ValidationHelper{registry: registry}
}

// ValidateProviderConfig validates a provider configuration without creating the provider.
func (vh *ValidationHelper) ValidateProviderConfig(providerType string, config ProviderConfig) error {
	// Check if provider type is registered
	if !IsProviderTypeRegistered(providerType) {
		available := GetRegisteredProviderTypes()
		return fmt.Errorf("unsupported provider type %q; available types: %s",
			providerType, strings.Join(available, ", "))
	}

	// Validate the configuration structure
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// For more thorough validation, we could create a temporary provider
	// and validate its configuration, but that might be expensive for some providers
	return nil
}

// ValidateAllConfigs validates all provider configurations in a map.
func (vh *ValidationHelper) ValidateAllConfigs(configs map[string]ProviderConfig) map[string]error {
	errors := make(map[string]error)

	for name, config := range configs {
		if err := vh.ValidateProviderConfig(config.Type, config); err != nil {
			errors[name] = err
		}
	}

	return errors
}

// GetRecommendedConfig returns recommended configuration for a provider type.
func GetRecommendedConfig(providerType string) (ProviderConfig, error) {
	tlsVerify := true
	config := ProviderConfig{
		Type:       providerType,
		MaxRetries: 3,
		Timeout:    DefaultSettings.Timeout,
		TLSVerify:  &tlsVerify,
	}

	switch strings.ToLower(providerType) {
	case "openai":
		config.BaseURL = "https://api.openai.com/v1"
		config.Extra = map[string]interface{}{
			"enable_function_calling": true,
			"enable_vision":           true,
		}
	case "anthropic":
		config.BaseURL = "https://api.anthropic.com"
		config.Extra = map[string]interface{}{
			"max_tokens": 4096,
		}
	case "ollama":
		config.BaseURL = "http://localhost:11434"
		tlsVerifyFalse := false
		config.TLSVerify = &tlsVerifyFalse // Ollama typically runs locally without TLS
	default:
		return config, fmt.Errorf("no recommended configuration available for provider type %q", providerType)
	}

	return config, nil
}
