// Package core provides provider registry and base implementations.
//
// The provider registry manages multiple LLM providers and handles
// provider discovery, health checking, and lifecycle management.
//
// Example usage:
//
//	registry := core.NewProviderRegistry()
//
//	openaiProvider := openai.New(openai.Config{
//		APIKey: "sk-...",
//	})
//
//	err := registry.Register("openai", openaiProvider)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	provider, err := registry.Get("openai")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	resp, err := provider.CreateCompletion(ctx, req)
package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// Provider registry errors
	ErrProviderNotFound      = errors.New("provider not found")
	ErrProviderAlreadyExists = errors.New("provider already exists")
	ErrProviderNotHealthy    = errors.New("provider is not healthy")
	ErrNoProvidersAvailable  = errors.New("no providers available")
)

// ProviderFactory is a function that creates a provider instance.
type ProviderFactory func(config ProviderConfig) (Provider, error)

// ProviderRegistry manages multiple LLM providers and handles provider
// discovery, health checking, and lifecycle management.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	factories map[string]ProviderFactory
	health    map[string]*ProviderHealth
	config    RegistryConfig
}

// RegistryConfig configures the provider registry behavior.
type RegistryConfig struct {
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	EnableMetrics       bool          `json:"enable_metrics"`
	DefaultProvider     string        `json:"default_provider,omitempty"`
}

// DefaultRegistryConfig returns sensible defaults for registry configuration.
func DefaultRegistryConfig() RegistryConfig {
	return RegistryConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		EnableMetrics:       true,
	}
}

// ProviderHealth tracks the health status of a provider.
type ProviderHealth struct {
	Name         string        `json:"name"`
	IsHealthy    bool          `json:"is_healthy"`
	LastCheck    time.Time     `json:"last_check"`
	LastError    error         `json:"last_error,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	CheckCount   int64         `json:"check_count"`
	ErrorCount   int64         `json:"error_count"`
}

// NewProviderRegistry creates a new provider registry with default configuration.
func NewProviderRegistry() *ProviderRegistry {
	return NewProviderRegistryWithConfig(DefaultRegistryConfig())
}

// NewProviderRegistryWithConfig creates a new provider registry with custom configuration.
func NewProviderRegistryWithConfig(config RegistryConfig) *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[string]Provider),
		factories: make(map[string]ProviderFactory),
		health:    make(map[string]*ProviderHealth),
		config:    config,
	}

	// Start health checking if enabled
	if config.HealthCheckInterval > 0 {
		go registry.startHealthChecker()
	}

	return registry
}

// RegisterFactory registers a factory function for creating providers of a specific type.
func (r *ProviderRegistry) RegisterFactory(providerType string, factory ProviderFactory) error {
	if providerType == "" {
		return &ValidationError{
			Field:   "providerType",
			Value:   providerType,
			Rule:    "required",
			Message: "provider type cannot be empty",
		}
	}

	if factory == nil {
		return &ValidationError{
			Field:   "factory",
			Value:   factory,
			Rule:    "required",
			Message: "factory function cannot be nil",
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[providerType] = factory
	return nil
}

// CreateProvider creates a provider instance using a registered factory.
func (r *ProviderRegistry) CreateProvider(providerType string, config ProviderConfig) (Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider config: %w", err)
	}

	r.mu.RLock()
	factory, exists := r.factories[providerType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no factory registered for provider type %q: %w", providerType, ErrProviderNotFound)
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

// Register registers a provider instance with the registry.
func (r *ProviderRegistry) Register(name string, provider Provider) error {
	if name == "" {
		return &ValidationError{
			Field:   "name",
			Value:   name,
			Rule:    "required",
			Message: "provider name cannot be empty",
		}
	}

	if provider == nil {
		return &ValidationError{
			Field:   "provider",
			Value:   provider,
			Rule:    "required",
			Message: "provider cannot be nil",
		}
	}

	// Validate provider configuration
	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q: %w", name, ErrProviderAlreadyExists)
	}

	r.providers[name] = provider
	r.health[name] = &ProviderHealth{
		Name:      name,
		IsHealthy: true,
		LastCheck: time.Now(),
	}

	return nil
}

// Get retrieves a provider by name.
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	if name == "" {
		return nil, &ValidationError{
			Field:   "name",
			Value:   name,
			Rule:    "required",
			Message: "provider name cannot be empty",
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %q: %w", name, ErrProviderNotFound)
	}

	// Check if provider is healthy
	if health, exists := r.health[name]; exists && !health.IsHealthy {
		return nil, fmt.Errorf("provider %q: %w", name, ErrProviderNotHealthy)
	}

	return provider, nil
}

// GetDefault returns the default provider if configured.
func (r *ProviderRegistry) GetDefault() (Provider, error) {
	if r.config.DefaultProvider == "" {
		return nil, errors.New("no default provider configured")
	}

	return r.Get(r.config.DefaultProvider)
}

// List returns all registered provider names.
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

// ListHealthy returns all healthy provider names.
func (r *ProviderRegistry) ListHealthy() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name, health := range r.health {
		if health.IsHealthy {
			names = append(names, name)
		}
	}

	return names
}

// Unregister removes a provider from the registry.
func (r *ProviderRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider %q: %w", name, ErrProviderNotFound)
	}

	delete(r.providers, name)
	delete(r.health, name)

	return nil
}

// HealthCheck performs a health check on all registered providers.
func (r *ProviderRegistry) HealthCheck(ctx context.Context) map[string]*ProviderHealth {
	r.mu.RLock()
	providers := make(map[string]Provider, len(r.providers))
	for name, provider := range r.providers {
		providers[name] = provider
	}
	r.mu.RUnlock()

	results := make(map[string]*ProviderHealth)
	var wg sync.WaitGroup
	var resMu sync.Mutex

	for name, provider := range providers {
		wg.Add(1)
		go func(name string, provider Provider) {
			defer wg.Done()

			health := r.checkProviderHealth(ctx, name, provider)

			// Update shared health map under lock
			r.mu.Lock()
			r.health[name] = health
			r.mu.Unlock()

			// Collect results under local mutex to avoid concurrent map writes
			resMu.Lock()
			results[name] = health
			resMu.Unlock()
		}(name, provider)
	}

	wg.Wait()
	return results
}

// GetHealth returns the health status of a specific provider.
func (r *ProviderRegistry) GetHealth(name string) (*ProviderHealth, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health, exists := r.health[name]
	if !exists {
		return nil, fmt.Errorf("provider %q: %w", name, ErrProviderNotFound)
	}

	// Return a copy to avoid race conditions
	healthCopy := *health
	return &healthCopy, nil
}

// checkProviderHealth performs a health check on a single provider.
func (r *ProviderRegistry) checkProviderHealth(ctx context.Context, name string, provider Provider) *ProviderHealth {
	ctx, cancel := context.WithTimeout(ctx, r.config.HealthCheckTimeout)
	defer cancel()

	start := time.Now()

	// Get current health status
	r.mu.RLock()
	currentHealth := r.health[name]
	r.mu.RUnlock()

	var health *ProviderHealth
	if currentHealth != nil {
		health = &ProviderHealth{
			Name:       currentHealth.Name,
			CheckCount: currentHealth.CheckCount + 1,
			ErrorCount: currentHealth.ErrorCount,
		}
	} else {
		health = &ProviderHealth{
			Name:       name,
			CheckCount: 1,
		}
	}

	health.LastCheck = time.Now()

	// Perform simple health check by validating configuration
	err := provider.ValidateConfig()
	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		health.ErrorCount++
	} else {
		// For providers that implement ModelLister, try to get models as a deeper health check
		if modelLister, ok := provider.(ModelLister); ok {
			_, err = modelLister.GetModels(ctx)
			if err != nil {
				health.IsHealthy = false
				health.LastError = fmt.Errorf("model listing failed: %w", err)
				health.ErrorCount++
			} else {
				health.IsHealthy = true
				health.LastError = nil
			}
		} else {
			// Basic validation passed
			health.IsHealthy = true
			health.LastError = nil
		}
	}

	health.ResponseTime = time.Since(start)
	return health
}

// startHealthChecker starts the background health checking routine.
func (r *ProviderRegistry) startHealthChecker() {
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), r.config.HealthCheckTimeout*2)
		r.HealthCheck(ctx)
		cancel()
	}
}

// BaseProvider provides common functionality for provider implementations.
type BaseProvider struct {
	name   string
	config ProviderConfig
}

// NewBaseProvider creates a new base provider.
func NewBaseProvider(name string, config ProviderConfig) *BaseProvider {
	return &BaseProvider{
		name:   name,
		config: config,
	}
}

// Name returns the provider name.
func (p *BaseProvider) Name() string {
	return p.name
}

// ValidateConfig validates the provider configuration.
func (p *BaseProvider) ValidateConfig() error {
	return p.config.Validate()
}

// Config returns the provider configuration.
func (p *BaseProvider) Config() ProviderConfig {
	return p.config
}

// ProviderMetrics tracks metrics for provider operations.
type ProviderMetrics struct {
	mu sync.RWMutex

	// Request counts
	TotalRequests   int64 `json:"total_requests"`
	SuccessRequests int64 `json:"success_requests"`
	ErrorRequests   int64 `json:"error_requests"`

	// Timing metrics
	TotalResponseTime time.Duration `json:"total_response_time"`
	MinResponseTime   time.Duration `json:"min_response_time"`
	MaxResponseTime   time.Duration `json:"max_response_time"`

	// Token usage
	TotalTokensUsed       int64 `json:"total_tokens_used"`
	TotalPromptTokens     int64 `json:"total_prompt_tokens"`
	TotalCompletionTokens int64 `json:"total_completion_tokens"`

	// Cost tracking
	TotalCost float64 `json:"total_cost"`

	// Last updated
	LastUpdated time.Time `json:"last_updated"`
}

// NewProviderMetrics creates new provider metrics.
func NewProviderMetrics() *ProviderMetrics {
	return &ProviderMetrics{
		LastUpdated: time.Now(),
	}
}

// RecordRequest records a completed request in the metrics.
func (m *ProviderMetrics) RecordRequest(responseTime time.Duration, usage *Usage, cost *float64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	m.LastUpdated = time.Now()

	if err != nil {
		m.ErrorRequests++
		return
	}

	m.SuccessRequests++
	m.TotalResponseTime += responseTime

	if m.MinResponseTime == 0 || responseTime < m.MinResponseTime {
		m.MinResponseTime = responseTime
	}
	if responseTime > m.MaxResponseTime {
		m.MaxResponseTime = responseTime
	}

	if usage != nil {
		m.TotalTokensUsed += int64(usage.TotalTokens)
		m.TotalPromptTokens += int64(usage.PromptTokens)
		m.TotalCompletionTokens += int64(usage.CompletionTokens)
	}

	if cost != nil {
		m.TotalCost += *cost
	}
}

// GetAverageResponseTime calculates the average response time.
func (m *ProviderMetrics) GetAverageResponseTime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.SuccessRequests == 0 {
		return 0
	}

	return m.TotalResponseTime / time.Duration(m.SuccessRequests)
}

// GetSuccessRate calculates the success rate as a percentage.
func (m *ProviderMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalRequests == 0 {
		return 0
	}

	return float64(m.SuccessRequests) / float64(m.TotalRequests) * 100
}

// Clone creates a copy of the metrics for safe reading.
func (m *ProviderMetrics) Clone() *ProviderMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &ProviderMetrics{
		TotalRequests:         m.TotalRequests,
		SuccessRequests:       m.SuccessRequests,
		ErrorRequests:         m.ErrorRequests,
		TotalResponseTime:     m.TotalResponseTime,
		MinResponseTime:       m.MinResponseTime,
		MaxResponseTime:       m.MaxResponseTime,
		TotalTokensUsed:       m.TotalTokensUsed,
		TotalPromptTokens:     m.TotalPromptTokens,
		TotalCompletionTokens: m.TotalCompletionTokens,
		TotalCost:             m.TotalCost,
		LastUpdated:           m.LastUpdated,
	}
}
