// Package config provides comprehensive configuration management for GOLLM.
//
// The configuration system supports hierarchical loading with the following precedence:
//  1. Command-line flags
//  2. Environment variables
//  3. Configuration file
//  4. Default values
//
// Configuration files are searched in the following locations:
//   - ./config.yaml
//   - ~/.gollm/config.yaml
//   - /etc/gollm/config.yaml
//
// Example usage:
//
//	config, err := config.Load()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	provider, err := config.GetProvider("openai")
//	if err != nil {
//		log.Fatal(err)
//	}
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/yourusername/gollm/internal/core"
)

// Config represents the complete application configuration.
type Config struct {
	// Default provider to use when none is specified
	DefaultProvider string `mapstructure:"default_provider" json:"default_provider" yaml:"default_provider" validate:"omitempty"`

	// Default model (for simplified menu)
	Model string `mapstructure:"model" json:"model" yaml:"model" validate:"omitempty"`

	// Provider configurations
	Providers map[string]ProviderConfig `mapstructure:"providers" json:"providers" yaml:"providers" validate:"omitempty"`

	// Global settings
	Settings GlobalSettings `mapstructure:"settings" json:"settings" yaml:"settings"`

	// Feature flags
	Features FeatureFlags `mapstructure:"features" json:"features" yaml:"features"`

	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging" json:"logging" yaml:"logging"`

	// Security settings
	Security SecurityConfig `mapstructure:"security" json:"security" yaml:"security"`

	// Cache configuration
	Cache CacheConfig `mapstructure:"cache" json:"cache" yaml:"cache"`

	// Plugin configuration
	Plugins PluginConfig `mapstructure:"plugins" json:"plugins" yaml:"plugins"`

	// MCP (Model Context Protocol) configuration
	MCP MCPConfig `mapstructure:"mcp" json:"mcp" yaml:"mcp"`

	// Extra configuration (for menu-specific settings)
	Extra map[string]interface{} `mapstructure:"extra" json:"extra" yaml:"extra"`
}

// ProviderConfig represents configuration for a single LLM provider.
type ProviderConfig struct {
	Type          string            `mapstructure:"type" json:"type" yaml:"type" validate:"required,oneof=openai anthropic ollama gemini deepseek openrouter custom"`
	APIKey        SecureString      `mapstructure:"api_key" json:"api_key,omitempty" yaml:"api_key"`
	BaseURL       string            `mapstructure:"base_url" json:"base_url,omitempty" yaml:"base_url" validate:"omitempty,url"`
	Organization  string            `mapstructure:"organization" json:"organization,omitempty" yaml:"organization"`
	MaxRetries    int               `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries" validate:"min=0,max=10"`
	Timeout       time.Duration     `mapstructure:"timeout" json:"timeout" yaml:"timeout" validate:"min=1s,max=300s"`
	RateLimit     string            `mapstructure:"rate_limit" json:"rate_limit,omitempty" yaml:"rate_limit" validate:"omitempty"`
	CustomHeaders map[string]string `mapstructure:"custom_headers" json:"custom_headers,omitempty" yaml:"custom_headers"`
	TLSVerify     bool              `mapstructure:"tls_verify" json:"tls_verify" yaml:"tls_verify"`
	Models        []string          `mapstructure:"models" json:"models,omitempty" yaml:"models"`
	DefaultModel  string            `mapstructure:"default_model" json:"default_model,omitempty" yaml:"default_model"`

	// Provider-specific settings
	Extra map[string]interface{} `mapstructure:"extra" json:"extra,omitempty" yaml:"extra"`
}

// Validate validates the provider configuration.
func (pc *ProviderConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(pc); err != nil {
		return fmt.Errorf("provider validation failed: %w", err)
	}
	return nil
}

// GlobalSettings contains global application settings.
type GlobalSettings struct {
	MaxTokens        int           `mapstructure:"max_tokens" json:"max_tokens" yaml:"maxtokens" validate:"min=1,max=32768"`
	Temperature      float64       `mapstructure:"temperature" json:"temperature" yaml:"temperature" validate:"min=0,max=2"`
	TopP             float64       `mapstructure:"top_p" json:"top_p" yaml:"topp" validate:"min=0,max=1"`
	FrequencyPenalty float64       `mapstructure:"frequency_penalty" json:"frequency_penalty" yaml:"frequencypenalty" validate:"min=-2,max=2"`
	PresencePenalty  float64       `mapstructure:"presence_penalty" json:"presence_penalty" yaml:"presencepenalty" validate:"min=-2,max=2"`
	Timeout          time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout" validate:"min=1s,max=300s"`
	RetryAttempts    int           `mapstructure:"retry_attempts" json:"retry_attempts" yaml:"retryattempts" validate:"min=0,max=10"`
	RetryDelay       time.Duration `mapstructure:"retry_delay" json:"retry_delay" yaml:"retrydelay" validate:"min=100ms,max=60s"`
	UserAgent        string        `mapstructure:"user_agent" json:"user_agent" yaml:"useragent"`
	OutputFormat     string        `mapstructure:"output_format" json:"output_format" yaml:"outputformat" validate:"oneof=text json yaml"`
}

// ApplyDefaults applies default values to GlobalSettings.
func (gs *GlobalSettings) ApplyDefaults() {
	if gs.MaxTokens == 0 {
		gs.MaxTokens = 2048
	}
	if gs.Temperature == 0 {
		gs.Temperature = 0.7
	}
	if gs.TopP == 0 {
		gs.TopP = 1.0
	}
	if gs.Timeout == 0 {
		gs.Timeout = 30 * time.Second
	}
	if gs.RetryAttempts == 0 {
		gs.RetryAttempts = 3
	}
	if gs.RetryDelay == 0 {
		gs.RetryDelay = time.Second
	}
	if gs.UserAgent == "" {
		gs.UserAgent = "gollm/1.0.0"
	}
	if gs.OutputFormat == "" {
		gs.OutputFormat = "text"
	}
}

// Merge merges another GlobalSettings into this one.
func (gs GlobalSettings) Merge(other GlobalSettings) GlobalSettings {
	result := gs

	if other.MaxTokens != 0 {
		result.MaxTokens = other.MaxTokens
	}
	if other.Temperature != 0 {
		result.Temperature = other.Temperature
	}
	if other.TopP != 0 {
		result.TopP = other.TopP
	}
	if other.FrequencyPenalty != 0 {
		result.FrequencyPenalty = other.FrequencyPenalty
	}
	if other.PresencePenalty != 0 {
		result.PresencePenalty = other.PresencePenalty
	}
	if other.Timeout != 0 {
		result.Timeout = other.Timeout
	}
	if other.RetryAttempts != 0 {
		result.RetryAttempts = other.RetryAttempts
	}
	if other.RetryDelay != 0 {
		result.RetryDelay = other.RetryDelay
	}
	if other.UserAgent != "" {
		result.UserAgent = other.UserAgent
	}
	if other.OutputFormat != "" {
		result.OutputFormat = other.OutputFormat
	}

	return result
}

// FeatureFlags controls optional features.
type FeatureFlags struct {
	Streaming    bool `mapstructure:"streaming" json:"streaming" yaml:"streaming"`
	Caching      bool `mapstructure:"caching" json:"caching" yaml:"caching"`
	Plugins      bool `mapstructure:"plugins" json:"plugins" yaml:"plugins"`
	MCP          bool `mapstructure:"mcp" json:"mcp" yaml:"mcp"`
	Metrics      bool `mapstructure:"metrics" json:"metrics" yaml:"metrics"`
	HealthChecks bool `mapstructure:"health_checks" json:"health_checks" yaml:"healthchecks"`
	AutoRetry    bool `mapstructure:"auto_retry" json:"auto_retry" yaml:"autoretry"`
	Compression  bool `mapstructure:"compression" json:"compression" yaml:"compression"`
}

// ApplyDefaults applies default values to FeatureFlags.
func (ff *FeatureFlags) ApplyDefaults() {
	// Feature flags default to false, so no changes needed
	// This method exists for consistency with other config structs
}

// Merge merges another FeatureFlags into this one.
func (ff FeatureFlags) Merge(other FeatureFlags) FeatureFlags {
	result := ff

	// For boolean flags, other takes precedence if true
	if other.Streaming {
		result.Streaming = other.Streaming
	}
	if other.Caching {
		result.Caching = other.Caching
	}
	if other.Plugins {
		result.Plugins = other.Plugins
	}
	if other.MCP {
		result.MCP = other.MCP
	}
	if other.Metrics {
		result.Metrics = other.Metrics
	}
	if other.HealthChecks {
		result.HealthChecks = other.HealthChecks
	}
	if other.AutoRetry {
		result.AutoRetry = other.AutoRetry
	}
	if other.Compression {
		result.Compression = other.Compression
	}

	return result
}

// LoggingConfig configures application logging.
type LoggingConfig struct {
	Level        string `mapstructure:"level" json:"level" yaml:"level" validate:"oneof=trace debug info warn error fatal panic"`
	Format       string `mapstructure:"format" json:"format" yaml:"format" validate:"oneof=text json"`
	Output       string `mapstructure:"output" json:"output" yaml:"output" validate:"oneof=stdout stderr file"`
	File         string `mapstructure:"file" json:"file,omitempty" yaml:"file"`
	MaxSize      int    `mapstructure:"max_size" json:"max_size" yaml:"maxsize" validate:"min=1,max=1000"` // MB
	MaxBackups   int    `mapstructure:"max_backups" json:"max_backups" yaml:"maxbackups" validate:"min=0,max=100"`
	MaxAge       int    `mapstructure:"max_age" json:"max_age" yaml:"maxage" validate:"min=1,max=365"` // Days
	AuditEnabled bool   `mapstructure:"audit_enabled" json:"audit_enabled" yaml:"auditenabled"`
}

// ApplyDefaults applies default values to LoggingConfig.
func (lc *LoggingConfig) ApplyDefaults() {
	if lc.Level == "" {
		lc.Level = "info"
	}
	if lc.Format == "" {
		lc.Format = "json"
	}
	if lc.Output == "" {
		lc.Output = "stdout"
	}
	if lc.MaxSize == 0 {
		lc.MaxSize = 100
	}
	if lc.MaxAge == 0 {
		lc.MaxAge = 30
	}
}

// Merge merges another LoggingConfig into this one.
func (lc LoggingConfig) Merge(other LoggingConfig) LoggingConfig {
	result := lc

	if other.Level != "" {
		result.Level = other.Level
	}
	if other.Format != "" {
		result.Format = other.Format
	}
	if other.Output != "" {
		result.Output = other.Output
	}
	if other.MaxSize != 0 {
		result.MaxSize = other.MaxSize
	}
	if other.MaxAge != 0 {
		result.MaxAge = other.MaxAge
	}
	if other.File != "" {
		result.File = other.File
	}

	return result
}

// SecurityConfig contains security settings.
type SecurityConfig struct {
	TLSMinVersion      string        `mapstructure:"tls_min_version" json:"tls_min_version" yaml:"tlsminversion" validate:"oneof=1.0 1.1 1.2 1.3"`
	CertificatePinning bool          `mapstructure:"certificate_pinning" json:"certificate_pinning" yaml:"certificatepinning"`
	TokenRotation      bool          `mapstructure:"token_rotation" json:"token_rotation" yaml:"tokenrotation"`
	EncryptConfig      bool          `mapstructure:"encrypt_config" json:"encrypt_config" yaml:"encryptconfig"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout" json:"session_timeout" yaml:"sessiontimeout" validate:"min=1m,max=24h"`
	MaxRequestSize     int64         `mapstructure:"max_request_size" json:"max_request_size" yaml:"maxrequestsize" validate:"min=1024,max=104857600"` // 1KB to 100MB
}

// ApplyDefaults applies default values to SecurityConfig.
func (sc *SecurityConfig) ApplyDefaults() {
	if sc.TLSMinVersion == "" {
		sc.TLSMinVersion = "1.3"
	}
	if sc.SessionTimeout == 0 {
		sc.SessionTimeout = 2 * time.Hour
	}
	if sc.MaxRequestSize == 0 {
		sc.MaxRequestSize = 10 * 1024 * 1024 // 10MB
	}
}

// Merge merges another SecurityConfig into this one.
func (sc SecurityConfig) Merge(other SecurityConfig) SecurityConfig {
	result := sc

	if other.TLSMinVersion != "" {
		result.TLSMinVersion = other.TLSMinVersion
	}
	if other.CertificatePinning {
		result.CertificatePinning = other.CertificatePinning
	}
	if other.TokenRotation {
		result.TokenRotation = other.TokenRotation
	}
	if other.EncryptConfig {
		result.EncryptConfig = other.EncryptConfig
	}
	if other.SessionTimeout != 0 {
		result.SessionTimeout = other.SessionTimeout
	}
	if other.MaxRequestSize != 0 {
		result.MaxRequestSize = other.MaxRequestSize
	}

	return result
}

// CacheConfig configures the response caching system.
type CacheConfig struct {
	Enabled     bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Type        string        `mapstructure:"type" json:"type" yaml:"type" validate:"oneof=memory redis file"`
	TTL         time.Duration `mapstructure:"ttl" json:"ttl" yaml:"ttl" validate:"min=1m,max=24h"`
	MaxSize     int64         `mapstructure:"max_size" json:"max_size" yaml:"maxsize" validate:"min=1048576,max=10737418240"` // 1MB to 10GB
	Directory   string        `mapstructure:"directory" json:"directory,omitempty" yaml:"directory"`
	RedisURL    string        `mapstructure:"redis_url" json:"redis_url,omitempty" yaml:"redisurl"`
	Compression bool          `mapstructure:"compression" json:"compression" yaml:"compression"`
}

// ApplyDefaults applies default values to CacheConfig.
func (cc *CacheConfig) ApplyDefaults() {
	if cc.Type == "" {
		cc.Type = "memory"
	}
	if cc.TTL == 0 {
		cc.TTL = time.Hour
	}
	if cc.MaxSize == 0 {
		cc.MaxSize = 100 * 1024 * 1024 // 100MB
	}
}

// Merge merges another CacheConfig into this one.
func (cc CacheConfig) Merge(other CacheConfig) CacheConfig {
	result := cc

	if other.Enabled {
		result.Enabled = other.Enabled
	}
	if other.Type != "" {
		result.Type = other.Type
	}
	if other.TTL != 0 {
		result.TTL = other.TTL
	}
	if other.MaxSize != 0 {
		result.MaxSize = other.MaxSize
	}
	if other.Directory != "" {
		result.Directory = other.Directory
	}
	if other.RedisURL != "" {
		result.RedisURL = other.RedisURL
	}
	if other.Compression {
		result.Compression = other.Compression
	}

	return result
}

// PluginConfig configures the plugin system.
type PluginConfig struct {
	Enabled   bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Directory string        `mapstructure:"directory" json:"directory,omitempty" yaml:"directory"`
	AutoLoad  bool          `mapstructure:"auto_load" json:"auto_load" yaml:"autoload"`
	Whitelist []string      `mapstructure:"whitelist" json:"whitelist,omitempty" yaml:"whitelist"`
	Blacklist []string      `mapstructure:"blacklist" json:"blacklist,omitempty" yaml:"blacklist"`
	Timeout   time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout" validate:"min=1s,max=300s"`
	MaxMemory int64         `mapstructure:"max_memory" json:"max_memory" yaml:"maxmemory" validate:"min=1048576,max=1073741824"` // 1MB to 1GB
}

// ApplyDefaults applies default values to PluginConfig.
func (pc *PluginConfig) ApplyDefaults() {
	if pc.Timeout == 0 {
		pc.Timeout = 30 * time.Second
	}
	if pc.MaxMemory == 0 {
		pc.MaxMemory = 64 * 1024 * 1024 // 64MB
	}
}

// Merge merges another PluginConfig into this one.
func (pc PluginConfig) Merge(other PluginConfig) PluginConfig {
	result := pc

	if other.Enabled {
		result.Enabled = other.Enabled
	}
	if other.Directory != "" {
		result.Directory = other.Directory
	}
	if other.Timeout != 0 {
		result.Timeout = other.Timeout
	}
	if other.MaxMemory != 0 {
		result.MaxMemory = other.MaxMemory
	}
	if len(other.Whitelist) > 0 {
		result.Whitelist = other.Whitelist
	}

	return result
}

// MCPConfig contains Model Context Protocol configuration.
type MCPConfig struct {
	Enabled      bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Servers      []MCPServer   `mapstructure:"servers" json:"servers" yaml:"servers"`
	Timeout      time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout" validate:"min=1s,max=300s"`
	MaxRetries   int           `mapstructure:"max_retries" json:"max_retries" yaml:"maxretries" validate:"min=0,max=10"`
	HealthChecks bool          `mapstructure:"health_checks" json:"health_checks" yaml:"healthchecks"`
}

// MCPServer represents an MCP server configuration.
type MCPServer struct {
	Name    string `mapstructure:"name" json:"name" yaml:"name" validate:"required"`
	URL     string `mapstructure:"url" json:"url" yaml:"url" validate:"required,url"`
	Type    string `mapstructure:"type" json:"type" yaml:"type" validate:"oneof=websocket http"`
	Enabled bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// ApplyDefaults applies default values to MCPConfig.
func (mc *MCPConfig) ApplyDefaults() {
	if mc.Timeout == 0 {
		mc.Timeout = 30 * time.Second
	}
}

// Merge merges another MCPConfig into this one.
func (mc MCPConfig) Merge(other MCPConfig) MCPConfig {
	result := mc

	if other.Enabled {
		result.Enabled = other.Enabled
	}
	if other.Timeout != 0 {
		result.Timeout = other.Timeout
	}
	if len(other.Servers) > 0 {
		result.Servers = other.Servers
	}

	return result
}

// SecureString represents a string that should not be logged or displayed.
type SecureString struct {
	data []byte
}

// NewSecureString creates a new SecureString from a regular string.
func NewSecureString(value string) SecureString {
	return SecureString{
		data: []byte(value),
	}
}

// String implements fmt.Stringer to prevent accidental logging.
func (s SecureString) String() string {
	if len(s.data) == 0 {
		return ""
	}
	return "***REDACTED***"
}

// Value returns the actual string value.
func (s SecureString) Value() string {
	return string(s.data)
}

// IsEmpty returns true if the secure string is empty.
func (s SecureString) IsEmpty() bool {
	return len(s.data) == 0
}

// Clear zeroes out the string value for security.
func (s *SecureString) Clear() {
	for i := range s.data {
		s.data[i] = 0
	}
	s.data = s.data[:0]
}

// UnmarshalText implements encoding.TextUnmarshaler for config loading.
func (s *SecureString) UnmarshalText(text []byte) error {
	s.data = make([]byte, len(text))
	copy(s.data, text)
	return nil
}

// MarshalText implements encoding.TextMarshaler for config saving.
func (s SecureString) MarshalText() ([]byte, error) {
	// Return the actual data for serialization
	return s.data, nil
}

// MarshalYAML implements yaml.Marshaler for YAML serialization.
func (s SecureString) MarshalYAML() (interface{}, error) {
	return string(s.data), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for YAML deserialization.
func (s *SecureString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	s.data = []byte(str)
	return nil
}

// MarshalJSON implements json.Marshaler for JSON serialization.
func (s SecureString) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(s.data) + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler for JSON deserialization.
func (s *SecureString) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	s.data = make([]byte, len(data))
	copy(s.data, data)
	return nil
}

var (
	// Configuration errors
	ErrConfigNotFound   = errors.New("configuration file not found")
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrProviderNotFound = errors.New("provider not found in configuration")
	ErrValidationFailed = errors.New("configuration validation failed")
)

// configPaths defines the search paths for configuration files.
var configPaths = []string{
	".",
	"$HOME/.gollm",
	"/etc/gollm",
}

// testConfigPaths can be set by tests to override default config paths.
var testConfigPaths []string

// getConfigPaths returns the search paths for configuration files.
// This function can be overridden in tests by using testConfigPaths.
func getConfigPaths() []string {
	if testConfigPaths != nil {
		return testConfigPaths
	}
	return configPaths
}

// setConfigPathsForTesting sets custom config paths for testing.
// This is used internally by tests.
func setConfigPathsForTesting(paths []string) {
	testConfigPaths = paths
}

// configNames defines the possible configuration file names.
var configNames = []string{
	"config",
	"gollm",
	".gollm",
}

// Load loads the configuration from various sources with proper precedence.
func Load() (*Config, error) {
	return LoadWithOptions(LoadOptions{})
}

// LoadOptions provides options for configuration loading.
type LoadOptions struct {
	ConfigFile     string
	SkipEnvVars    bool
	SkipValidation bool
	SkipDefaults   bool
}

// LoadWithOptions loads configuration with custom options.
func LoadWithOptions(opts LoadOptions) (*Config, error) {
	v := viper.New()

	// Set defaults if not skipped
	if !opts.SkipDefaults {
		setDefaults(v)
	}

	// Configure file searching
	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// Add search paths
		for _, path := range getConfigPaths() {
			v.AddConfigPath(os.ExpandEnv(path))
		}
	}

	// Configure environment variables if not skipped
	if !opts.SkipEnvVars {
		setupEnvironmentVariables(v)
	}

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
		// Config file not found is acceptable, we'll use defaults and env vars
	}

	// Unmarshal into struct with decode hooks
	var config Config
	if err := v.Unmarshal(&config, viper.DecodeHook(composedDecodeHook())); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate configuration if not skipped
	if !opts.SkipValidation {
		if err := validateConfig(&config); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Post-process configuration
	if err := postProcessConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration post-processing failed: %w", err)
	}

	return &config, nil
}

// LoadFromFile loads configuration from a specific file.
func LoadFromFile(filename string) (*Config, error) {
	return LoadWithOptions(LoadOptions{
		ConfigFile: filename,
	})
}

// GetProvider returns the configuration for a specific provider.
// If providerName is empty, returns the default provider.
func (c *Config) GetProvider(providerName string) (ProviderConfig, string, error) {
	// Use default provider if name is empty
	if providerName == "" {
		providerName = c.DefaultProvider
	}

	// Check if provider exists
	providerConfig, exists := c.Providers[providerName]
	if !exists {
		return ProviderConfig{}, "", fmt.Errorf("provider %q: %w", providerName, ErrProviderNotFound)
	}

	return providerConfig, providerName, nil
}

// Save saves the configuration to a file.
func (c *Config) Save(filename string) error {
	// Ensure all defaults are applied
	c.Settings.ApplyDefaults()
	c.Logging.ApplyDefaults()
	c.Security.ApplyDefaults()
	c.Cache.ApplyDefaults()
	c.Plugins.ApplyDefaults()

	// Validate before saving
	if err := c.Validate(); err != nil {
		return fmt.Errorf("cannot save invalid configuration: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(filename)
	v.SetConfigType("yaml")

	// Set all values in viper
	if err := setViperFromConfig(v, c); err != nil {
		return fmt.Errorf("failed to prepare config for saving: %w", err)
	}

    // Create directory if it doesn't exist
    dir := filepath.Dir(filename)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

	// Write config file
    if err := v.WriteConfig(); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    // Restrict file permissions to owner-only (0600)
    if err := os.Chmod(filename, 0600); err != nil {
        return fmt.Errorf("failed to set secure permissions on config file: %w", err)
    }

    return nil
}

// Validate validates the entire configuration.
func (c *Config) Validate() error {
	validate := validator.New()

	// Register custom validators
	validate.RegisterValidation("exists", func(fl validator.FieldLevel) bool {
		// For default_provider validation
		if fl.FieldName() == "DefaultProvider" {
			defaultProvider := fl.Field().String()
			if defaultProvider == "" {
				return true // Empty is valid
			}
			config := fl.Parent().Interface().(*Config)
			_, exists := config.Providers[defaultProvider]
			return exists
		}
		return true
	})

	// Validate struct
	if err := validate.Struct(c); err != nil {
		return core.NewValidationError("config", c, "struct_validation", err.Error())
	}

	// Validate individual provider configs
	for name, provider := range c.Providers {
		if err := provider.Validate(); err != nil {
			return fmt.Errorf("provider %s: %w", name, err)
		}
	}

	// Additional business logic validation
	if c.DefaultProvider != "" {
		if _, exists := c.Providers[c.DefaultProvider]; !exists {
			return core.NewValidationError("default_provider", c.DefaultProvider, "exists", "default provider must reference an existing provider")
		}
	}

	return nil
}

// Merge merges another config into this one, with the other config taking precedence.
func (c *Config) Merge(other *Config) *Config {
	merged := &Config{}

	// Copy base config
	*merged = *c

	// Override with other's values
	if other.DefaultProvider != "" {
		merged.DefaultProvider = other.DefaultProvider
	}

	// Merge providers
	merged.Providers = make(map[string]ProviderConfig)
	for name, provider := range c.Providers {
		merged.Providers[name] = provider
	}
	for name, provider := range other.Providers {
		merged.Providers[name] = provider
	}

	// Merge settings
	merged.Settings = c.Settings.Merge(other.Settings)
	merged.Features = c.Features.Merge(other.Features)
	merged.Logging = c.Logging.Merge(other.Logging)
	merged.Security = c.Security.Merge(other.Security)
	merged.Cache = c.Cache.Merge(other.Cache)
	merged.Plugins = c.Plugins.Merge(other.Plugins)
	merged.MCP = c.MCP.Merge(other.MCP)

	return merged
}

// setViperFromConfig sets viper values from config struct for saving.
func setViperFromConfig(v *viper.Viper, c *Config) error {
	// This would marshal the config to a map and set in viper
	// For now, we'll implement a simplified version
	v.Set("default_provider", c.DefaultProvider)
	v.Set("providers", c.Providers)
	v.Set("settings", c.Settings)
	v.Set("features", c.Features)
	v.Set("logging", c.Logging)
	v.Set("security", c.Security)
	v.Set("cache", c.Cache)
	v.Set("plugins", c.Plugins)
	v.Set("mcp", c.MCP)

	return nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Global settings defaults
	v.SetDefault("settings.max_tokens", core.DefaultSettings.MaxTokens)
	v.SetDefault("settings.temperature", core.DefaultSettings.Temperature)
	v.SetDefault("settings.top_p", core.DefaultSettings.TopP)
	v.SetDefault("settings.frequency_penalty", core.DefaultSettings.FrequencyPenalty)
	v.SetDefault("settings.presence_penalty", core.DefaultSettings.PresencePenalty)
	v.SetDefault("settings.timeout", core.DefaultSettings.Timeout)
	v.SetDefault("settings.retry_attempts", core.DefaultSettings.MaxRetries)
	v.SetDefault("settings.retry_delay", 1*time.Second)
	v.SetDefault("settings.user_agent", "gollm/1.0")
	v.SetDefault("settings.output_format", "text")

	// Feature flags defaults
	v.SetDefault("features.streaming", true)
	v.SetDefault("features.caching", false)
	v.SetDefault("features.plugins", false)
	v.SetDefault("features.mcp", false)
	v.SetDefault("features.metrics", true)
	v.SetDefault("features.health_checks", true)
	v.SetDefault("features.auto_retry", true)
	v.SetDefault("features.compression", true)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_backups", 3)
	v.SetDefault("logging.max_age", 7)
	v.SetDefault("logging.audit_enabled", false)

	// Security defaults
	v.SetDefault("security.tls_min_version", "1.3")
	v.SetDefault("security.certificate_pinning", false)
	v.SetDefault("security.token_rotation", false)
	v.SetDefault("security.encrypt_config", false)
	v.SetDefault("security.session_timeout", 24*time.Hour)
	v.SetDefault("security.max_request_size", 10*1024*1024) // 10MB

	// Cache defaults
	v.SetDefault("cache.enabled", false)
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.ttl", 1*time.Hour)
	v.SetDefault("cache.max_size", 100*1024*1024) // 100MB
	v.SetDefault("cache.compression", true)

	// Plugin defaults
	v.SetDefault("plugins.enabled", false)
	v.SetDefault("plugins.auto_load", false)
	v.SetDefault("plugins.timeout", 30*time.Second)
	v.SetDefault("plugins.max_memory", 64*1024*1024) // 64MB

	// Providers default - empty map to satisfy required validation
	v.SetDefault("providers", map[string]interface{}{})

	// MCP defaults
	v.SetDefault("mcp.enabled", false)
	v.SetDefault("mcp.timeout", 10*time.Second)
	v.SetDefault("mcp.max_retries", 3)
	v.SetDefault("mcp.health_checks", true)

	// Provider defaults
	v.SetDefault("providers", map[string]interface{}{})
}

// setupEnvironmentVariables configures environment variable handling.
func setupEnvironmentVariables(v *viper.Viper) {
	v.AutomaticEnv()
	v.SetEnvPrefix("GOLLM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicitly bind important environment variables
	envBindings := map[string]string{
		"default_provider": "GOLLM_DEFAULT_PROVIDER",

		// OpenAI provider bindings
		"providers.openai.type":         "GOLLM_PROVIDERS_OPENAI_TYPE",
		"providers.openai.api_key":      "GOLLM_PROVIDERS_OPENAI_API_KEY",
		"providers.openai.base_url":     "GOLLM_PROVIDERS_OPENAI_BASE_URL",
		"providers.openai.organization": "GOLLM_PROVIDERS_OPENAI_ORGANIZATION",
		"providers.openai.max_retries":  "GOLLM_PROVIDERS_OPENAI_MAX_RETRIES",
		"providers.openai.timeout":      "GOLLM_PROVIDERS_OPENAI_TIMEOUT",
		"providers.openai.tls_verify":   "GOLLM_PROVIDERS_OPENAI_TLS_VERIFY",

		// Anthropic provider bindings
		"providers.anthropic.type":        "GOLLM_PROVIDERS_ANTHROPIC_TYPE",
		"providers.anthropic.api_key":     "GOLLM_PROVIDERS_ANTHROPIC_API_KEY",
		"providers.anthropic.base_url":    "GOLLM_PROVIDERS_ANTHROPIC_BASE_URL",
		"providers.anthropic.max_retries": "GOLLM_PROVIDERS_ANTHROPIC_MAX_RETRIES",
		"providers.anthropic.timeout":     "GOLLM_PROVIDERS_ANTHROPIC_TIMEOUT",
		"providers.anthropic.tls_verify":  "GOLLM_PROVIDERS_ANTHROPIC_TLS_VERIFY",

		// OpenRouter provider bindings
		"providers.openrouter.type":        "GOLLM_PROVIDERS_OPENROUTER_TYPE",
		"providers.openrouter.api_key":     "GOLLM_PROVIDERS_OPENROUTER_API_KEY",
		"providers.openrouter.base_url":    "GOLLM_PROVIDERS_OPENROUTER_BASE_URL",
		"providers.openrouter.max_retries": "GOLLM_PROVIDERS_OPENROUTER_MAX_RETRIES",
		"providers.openrouter.timeout":     "GOLLM_PROVIDERS_OPENROUTER_TIMEOUT",
		"providers.openrouter.tls_verify":  "GOLLM_PROVIDERS_OPENROUTER_TLS_VERIFY",

		// Gemini provider bindings
		"providers.gemini.type":        "GOLLM_PROVIDERS_GEMINI_TYPE",
		"providers.gemini.api_key":     "GOLLM_PROVIDERS_GEMINI_API_KEY",
		"providers.gemini.base_url":    "GOLLM_PROVIDERS_GEMINI_BASE_URL",
		"providers.gemini.max_retries": "GOLLM_PROVIDERS_GEMINI_MAX_RETRIES",
		"providers.gemini.timeout":     "GOLLM_PROVIDERS_GEMINI_TIMEOUT",
		"providers.gemini.tls_verify":  "GOLLM_PROVIDERS_GEMINI_TLS_VERIFY",

		// DeepSeek provider bindings
		"providers.deepseek.type":        "GOLLM_PROVIDERS_DEEPSEEK_TYPE",
		"providers.deepseek.api_key":     "GOLLM_PROVIDERS_DEEPSEEK_API_KEY",
		"providers.deepseek.base_url":    "GOLLM_PROVIDERS_DEEPSEEK_BASE_URL",
		"providers.deepseek.max_retries": "GOLLM_PROVIDERS_DEEPSEEK_MAX_RETRIES",
		"providers.deepseek.timeout":     "GOLLM_PROVIDERS_DEEPSEEK_TIMEOUT",
		"providers.deepseek.tls_verify":  "GOLLM_PROVIDERS_DEEPSEEK_TLS_VERIFY",

		// Ollama provider bindings
		"providers.ollama.type":        "GOLLM_PROVIDERS_OLLAMA_TYPE",
		"providers.ollama.base_url":    "GOLLM_PROVIDERS_OLLAMA_BASE_URL",
		"providers.ollama.max_retries": "GOLLM_PROVIDERS_OLLAMA_MAX_RETRIES",
		"providers.ollama.timeout":     "GOLLM_PROVIDERS_OLLAMA_TIMEOUT",

		// Settings bindings
		"settings.max_tokens":        "GOLLM_SETTINGS_MAX_TOKENS",
		"settings.temperature":       "GOLLM_SETTINGS_TEMPERATURE",
		"settings.top_p":             "GOLLM_SETTINGS_TOP_P",
		"settings.frequency_penalty": "GOLLM_SETTINGS_FREQUENCY_PENALTY",
		"settings.presence_penalty":  "GOLLM_SETTINGS_PRESENCE_PENALTY",
		"settings.timeout":           "GOLLM_SETTINGS_TIMEOUT",
		"settings.retry_attempts":    "GOLLM_SETTINGS_RETRY_ATTEMPTS",
		"settings.retry_delay":       "GOLLM_SETTINGS_RETRY_DELAY",
		"settings.output_format":     "GOLLM_SETTINGS_OUTPUT_FORMAT",
	}

	for key, env := range envBindings {
		v.BindEnv(key, env)
	}
}

// composedDecodeHook creates a composed decode hook for all custom types.
func composedDecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		durationDecodeHook(),
		secureStringDecodeHook(),
	)
}

// durationDecodeHook creates a decode hook for time.Duration types.
func durationDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		// Check if target type is time.Duration
		if t != reflect.TypeOf(time.Duration(0)) {
			return data, nil
		}

		// Check if source is string
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Convert string to Duration
		str, ok := data.(string)
		if !ok {
			return data, nil
		}

		// Parse duration string
		duration, err := time.ParseDuration(str)
		if err != nil {
			return data, fmt.Errorf("invalid duration format: %w", err)
		}

		return duration, nil
	}
}

// secureStringDecodeHook creates a decode hook for SecureString types.
func secureStringDecodeHook() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		// Check if target type is SecureString
		if t != reflect.TypeOf(SecureString{}) {
			return data, nil
		}

		// Check if source is string (from environment variables)
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Convert string to SecureString with environment expansion
		str, ok := data.(string)
		if !ok {
			return data, nil
		}
		// Expand ${VAR} placeholders using the current process environment
		expanded := os.ExpandEnv(str)
		return NewSecureString(expanded), nil
	}
}

// validateConfig validates the loaded configuration.
func validateConfig(config *Config) error {
	validate := validator.New()

	// Register custom validators
	validate.RegisterValidation("url", validateURL)

	if err := validate.Struct(config); err != nil {
		var validationErrors []string

		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, validationErr := range validationErrs {
				validationErrors = append(validationErrors, formatValidationError(validationErr))
			}
		} else {
			validationErrors = append(validationErrors, err.Error())
		}

		return &core.ValidationError{
			Field:   "config",
			Rule:    "struct_validation",
			Message: strings.Join(validationErrors, "; "),
		}
	}

	// Custom validation logic
	if err := validateProviders(config.Providers); err != nil {
		return err
	}

	if err := validateDefaultProvider(config); err != nil {
		return err
	}

	return nil
}

// validateURL is a custom validator for URLs.
func validateURL(fl validator.FieldLevel) bool {
	url := fl.Field().String()
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://")
}

// validateProviders validates provider configurations.
func validateProviders(providers map[string]ProviderConfig) error {
	if len(providers) == 0 {
		return &core.ValidationError{
			Field:   "providers",
			Rule:    "required",
			Message: "at least one provider must be configured",
		}
	}

	for name, provider := range providers {
		if provider.Type == "" {
			return &core.ValidationError{
				Field:   fmt.Sprintf("providers.%s.type", name),
				Rule:    "required",
				Message: "provider type cannot be empty",
			}
		}

		// Validate provider-specific requirements
		switch provider.Type {
		case "openai", "anthropic":
			if provider.APIKey.IsEmpty() {
				return &core.ValidationError{
					Field:   fmt.Sprintf("providers.%s.api_key", name),
					Rule:    "required",
					Message: "API key is required for this provider type",
				}
			}
		case "ollama":
			if provider.BaseURL == "" {
				return &core.ValidationError{
					Field:   fmt.Sprintf("providers.%s.base_url", name),
					Rule:    "required",
					Message: "base URL is required for Ollama provider",
				}
			}
		case "custom":
			if provider.BaseURL == "" {
				return &core.ValidationError{
					Field:   fmt.Sprintf("providers.%s.base_url", name),
					Rule:    "required",
					Message: "base URL is required for custom provider",
				}
			}
		}
	}

	return nil
}

// validateDefaultProvider ensures the default provider exists.
func validateDefaultProvider(config *Config) error {
	if config.DefaultProvider != "" {
		if _, exists := config.Providers[config.DefaultProvider]; !exists {
			return &core.ValidationError{
				Field:   "default_provider",
				Value:   config.DefaultProvider,
				Rule:    "exists",
				Message: "default provider must reference an existing provider",
			}
		}
	}

	return nil
}

// formatValidationError formats a validation error for display.
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("field %s is required", err.Field())
	case "min":
		return fmt.Sprintf("field %s must be at least %s", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("field %s must be at most %s", err.Field(), err.Param())
	case "oneof":
		return fmt.Sprintf("field %s must be one of: %s", err.Field(), err.Param())
	case "url":
		return fmt.Sprintf("field %s must be a valid URL", err.Field())
	default:
		return fmt.Sprintf("field %s failed validation %s", err.Field(), err.Tag())
	}
}

// postProcessConfig performs post-processing on the loaded configuration.
func postProcessConfig(config *Config) error {
	// Initialize empty providers map if nil
	if config.Providers == nil {
		config.Providers = make(map[string]ProviderConfig)
	}

	// Apply defaults to all config sections
	config.Settings.ApplyDefaults()
	config.Features.ApplyDefaults()
	config.Logging.ApplyDefaults()
	config.Security.ApplyDefaults()
	config.Cache.ApplyDefaults()
	config.Plugins.ApplyDefaults()
	config.MCP.ApplyDefaults()

	// Expand environment variables in paths
	if config.Logging.File != "" {
		config.Logging.File = os.ExpandEnv(config.Logging.File)
	}

	if config.Cache.Directory != "" {
		config.Cache.Directory = os.ExpandEnv(config.Cache.Directory)
	}

	if config.Plugins.Directory != "" {
		config.Plugins.Directory = os.ExpandEnv(config.Plugins.Directory)
	}

	// Set default directories if not specified
	if config.Cache.Enabled && config.Cache.Type == "disk" && config.Cache.Directory == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		config.Cache.Directory = filepath.Join(homeDir, ".gollm", "cache")
	}

	if config.Plugins.Enabled && config.Plugins.Directory == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		config.Plugins.Directory = filepath.Join(homeDir, ".gollm", "plugins")
	}

	return nil
}

// GetDefaultProvider returns the default provider configuration.
func (c *Config) GetDefaultProvider() (string, ProviderConfig, error) {
	if c.DefaultProvider == "" {
		// Return the first available provider if no default is set
		for name, provider := range c.Providers {
			return name, provider, nil
		}
		return "", ProviderConfig{}, errors.New("no providers configured")
	}

	provider, _, err := c.GetProvider(c.DefaultProvider)
	if err != nil {
		return "", ProviderConfig{}, err
	}

	return c.DefaultProvider, provider, nil
}

// ListProviders returns all configured provider names.
func (c *Config) ListProviders() []string {
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	return names
}

// HasProvider checks if a provider is configured.
func (c *Config) HasProvider(name string) bool {
	_, exists := c.Providers[name]
	return exists
}

// ToProviderConfig converts to a core.ProviderConfig.
func (p *ProviderConfig) ToProviderConfig() core.ProviderConfig {
	return core.ProviderConfig{
		Type:          p.Type,
		APIKey:        p.APIKey.Value(),
		BaseURL:       p.BaseURL,
		Organization:  p.Organization,
		MaxRetries:    p.MaxRetries,
		Timeout:       p.Timeout,
		RateLimit:     p.RateLimit,
		CustomHeaders: p.CustomHeaders,
		TLSVerify:     &p.TLSVerify,
		Extra:         p.Extra,
	}
}
