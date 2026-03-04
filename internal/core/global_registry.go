package core

import (
	"strings"
	"sync"
)

var (
	globalRegistry *ProviderRegistry
	registryOnce   sync.Once
)

// Provider factory registry (global for the package). Tests manipulate
// globalFactories directly; core code should also use this map.
var (
	globalFactories = make(map[string]ProviderFactory)
	stableFactories = make(map[string]ProviderFactory)
	factoriesMu     sync.RWMutex
)

// GetGlobalRegistry returns the global provider registry instance
func GetGlobalRegistry() *ProviderRegistry {
    registryOnce.Do(func() {
        globalRegistry = NewProviderRegistry()

		// Register factory functions for each provider type
		// These will be called when providers are created from config
		globalRegistry.RegisterFactory("openai", func(config ProviderConfig) (Provider, error) {
			if factory, exists := globalFactories["openai"]; exists {
				return factory(config)
			}
			return nil, ErrProviderNotFound
		})

		globalRegistry.RegisterFactory("anthropic", func(config ProviderConfig) (Provider, error) {
			if factory, exists := globalFactories["anthropic"]; exists {
				return factory(config)
			}
			return nil, ErrProviderNotFound
		})

		globalRegistry.RegisterFactory("deepseek", func(config ProviderConfig) (Provider, error) {
			if factory, exists := globalFactories["deepseek"]; exists {
				return factory(config)
			}
			return nil, ErrProviderNotFound
		})

		globalRegistry.RegisterFactory("gemini", func(config ProviderConfig) (Provider, error) {
			if factory, exists := globalFactories["gemini"]; exists {
				return factory(config)
			}
			return nil, ErrProviderNotFound
		})

		globalRegistry.RegisterFactory("openrouter", func(config ProviderConfig) (Provider, error) {
			if factory, exists := globalFactories["openrouter"]; exists {
				return factory(config)
			}
			return nil, ErrProviderNotFound
		})

        globalRegistry.RegisterFactory("ollama", func(config ProviderConfig) (Provider, error) {
            if factory, exists := globalFactories["ollama"]; exists {
                return factory(config)
            }
            return nil, ErrProviderNotFound
        })

        // Register mock provider (useful for offline/testing and graceful fallbacks)
        globalRegistry.RegisterFactory("mock", func(config ProviderConfig) (Provider, error) {
            if factory, exists := globalFactories["mock"]; exists {
                return factory(config)
            }
            return nil, ErrProviderNotFound
        })
    })

    return globalRegistry
}

// RegisterProviderFactory registers a factory function for a provider type.
// This is typically called from provider packages' init() functions.
// It returns an error for invalid inputs; duplicate registrations overwrite.
func RegisterProviderFactory(providerType string, factory ProviderFactory) error {
	providerType = strings.ToLower(strings.TrimSpace(providerType))
	if providerType == "" {
		return &ValidationError{Field: "providerType", Value: providerType, Rule: "required", Message: "provider type cannot be empty"}
	}
	if factory == nil {
		return &ValidationError{Field: "factory", Value: nil, Rule: "required", Message: "factory function cannot be nil"}
	}

	factoriesMu.Lock()
	globalFactories[providerType] = factory
	stableFactories[providerType] = factory
	factoriesMu.Unlock()

	// If global registry is already initialized, register there too
	if globalRegistry != nil {
		_ = globalRegistry.RegisterFactory(providerType, factory)
	}
	return nil
}
