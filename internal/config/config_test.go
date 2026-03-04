// Package config provides comprehensive configuration management for GOLLM.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultConfiguration(t *testing.T) {
	if os.Getenv("CI_SANDBOX") == "1" {
		t.Skip("skipping environment-sensitive test in sandbox (CI_SANDBOX=1)")
	}
	// Save original environment
	originalEnv := os.Environ()
	defer func() {
		// Restore environment
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	// Clear all GOLLM environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GOLLM_") {
			key := strings.Split(env, "=")[0]
			os.Unsetenv(key)
		}
	}

	// Load with validation skipped since we're testing default loading behavior
	config, err := LoadWithOptions(LoadOptions{
		SkipValidation: true,
	})
	require.NoError(t, err, "Default configuration should load without error")
	require.NotNil(t, config, "Configuration should not be nil")

	// Check default values are applied
	assert.Equal(t, 2048, config.Settings.MaxTokens, "Default max_tokens should be 2048")
	assert.Equal(t, 0.7, config.Settings.Temperature, "Default temperature should be 0.7")
	assert.Equal(t, 30*time.Second, config.Settings.Timeout, "Default timeout should be 30s")

	// Check that providers map is initialized (empty but not nil)
	assert.NotNil(t, config.Providers, "Providers should not be nil")
}

func TestLoad_FromConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected func(*testing.T, *Config)
		wantErr  string
	}{
		{
			name: "valid YAML config",
			content: `
default_provider: openai
providers:
  openai:
    type: openai
    api_key: sk-test123
    base_url: https://api.openai.com/v1
settings:
  max_tokens: 4000
  temperature: 0.8
  timeout: 45s
features:
  streaming: true
  caching: false
`,
			expected: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "openai", cfg.DefaultProvider)
				assert.Equal(t, "openai", cfg.Providers["openai"].Type)
				assert.Equal(t, "***REDACTED***", cfg.Providers["openai"].APIKey.String())
				assert.Equal(t, 4000, cfg.Settings.MaxTokens)
				assert.Equal(t, 0.8, cfg.Settings.Temperature)
				assert.True(t, cfg.Features.Streaming)
			},
		},
		{
			name: "valid JSON config",
			content: `{
  "default_provider": "anthropic",
  "providers": {
    "anthropic": {
      "type": "anthropic",
      "api_key": "ant-test456",
      "base_url": "https://api.anthropic.com"
    }
  },
  "settings": {
    "max_tokens": 1000,
    "temperature": 0.3,
    "timeout": "60s"
  }
}`,
			expected: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "anthropic", cfg.DefaultProvider)
				assert.Equal(t, "anthropic", cfg.Providers["anthropic"].Type)
				assert.Equal(t, 1000, cfg.Settings.MaxTokens)
				assert.Equal(t, 0.3, cfg.Settings.Temperature)
			},
		},
		{
			name:    "invalid YAML syntax",
			content: "invalid: yaml: syntax [",
			wantErr: "failed to read configuration file",
		},
		{
			name: "missing required fields",
			content: `
settings:
  max_tokens: 1000
`,
			wantErr: "validation failed",
		},
		{
			name: "invalid timeout format",
			content: `
default_provider: openai
providers:
  openai:
    type: openai
    api_key: sk-test
    timeout: "invalid-duration"
`,
			wantErr: "failed to unmarshal config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configFile, []byte(tt.content), 0644)
			require.NoError(t, err)

			config, err := LoadFromFile(configFile)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			if tt.expected != nil {
				tt.expected(t, config)
			}
		})
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	if os.Getenv("CI_SANDBOX") == "1" {
		t.Skip("skipping environment-sensitive test in sandbox (CI_SANDBOX=1)")
	}

	// Isolate test from host's config files like ~/.gollm/config.yaml
	tmpDir := t.TempDir()
	setConfigPathsForTesting([]string{tmpDir})
	defer setConfigPathsForTesting(nil)

	tests := []struct {
		name     string
		envVars  map[string]string
		expected func(*testing.T, *Config)
		wantErr  string
	}{
		{
			name: "environment override",
			envVars: map[string]string{
				"GOLLM_DEFAULT_PROVIDER":         "openai",
				"GOLLM_PROVIDERS_OPENAI_TYPE":    "openai",
				"GOLLM_PROVIDERS_OPENAI_API_KEY": "sk-env-key",
				"GOLLM_SETTINGS_MAX_TOKENS":      "8000",
				"GOLLM_SETTINGS_TEMPERATURE":     "0.9",
			},
			expected: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "openai", cfg.DefaultProvider)
				assert.Equal(t, "openai", cfg.Providers["openai"].Type)
				assert.Equal(t, 8000, cfg.Settings.MaxTokens)
				assert.Equal(t, 0.9, cfg.Settings.Temperature)
			},
		},
		{
			name: "partial environment override",
			envVars: map[string]string{
				"GOLLM_PROVIDERS_OPENAI_TYPE":    "openai",
				"GOLLM_PROVIDERS_OPENAI_API_KEY": "sk-partial",
				"GOLLM_SETTINGS_MAX_TOKENS":      "3000",
			},
			expected: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "openai", cfg.Providers["openai"].Type)
				assert.Equal(t, 3000, cfg.Settings.MaxTokens)
			},
		},
		{
			name: "invalid environment values",
			envVars: map[string]string{
				"GOLLM_PROVIDERS_OPENAI_TYPE":    "openai",
				"GOLLM_PROVIDERS_OPENAI_API_KEY": "sk-test",
				"GOLLM_SETTINGS_MAX_TOKENS":      "-100", // Invalid
				"GOLLM_SETTINGS_TEMPERATURE":     "3.0",  // Invalid
			},
			wantErr: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any polluting environment API keys before doing mock env tests.
			os.Unsetenv("GEMINI_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("OPENROUTER_API_KEY")
			os.Unsetenv("DEEPSEEK_API_KEY")

			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Restore environment after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
			}()

			config, err := Load()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			if tt.expected != nil {
				tt.expected(t, config)
			}
		})
	}
}

func TestLoad_ConfigurationPrecedence(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
default_provider: anthropic
providers:
  anthropic:
    type: anthropic
    api_key: sk-file-key
    max_retries: 3
    timeout: 30s
  openai:
    type: openai
    api_key: sk-openai-file
    timeout: 30s
settings:
  max_tokens: 2000
  temperature: 0.5
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set config path
	originalConfigPaths := getConfigPaths()
	setConfigPathsForTesting([]string{tmpDir})
	defer setConfigPathsForTesting(originalConfigPaths)

	// Set environment variables (should override file)
	originalEnv := map[string]string{
		"GOLLM_DEFAULT_PROVIDER":                os.Getenv("GOLLM_DEFAULT_PROVIDER"),
		"GOLLM_PROVIDERS_ANTHROPIC_MAX_RETRIES": os.Getenv("GOLLM_PROVIDERS_ANTHROPIC_MAX_RETRIES"),
		"GOLLM_SETTINGS_TEMPERATURE":            os.Getenv("GOLLM_SETTINGS_TEMPERATURE"),
	}

	os.Setenv("GOLLM_DEFAULT_PROVIDER", "openai")
	os.Setenv("GOLLM_PROVIDERS_ANTHROPIC_MAX_RETRIES", "5")
	os.Setenv("GOLLM_SETTINGS_TEMPERATURE", "0.8")

	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check precedence: ENV > File > Defaults
	assert.Equal(t, "openai", config.DefaultProvider, "Environment should override file")
	assert.Equal(t, 5, config.Providers["anthropic"].MaxRetries, "Environment should override file")
	assert.Equal(t, 0.8, config.Settings.Temperature, "Environment should override file")
	assert.Equal(t, 2000, config.Settings.MaxTokens, "File should override defaults")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name:   "valid complete config",
			config: createValidTestConfig(),
		},
		{
			name: "missing provider type",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"openai": {
						APIKey: NewSecureString("sk-test123"),
					},
				},
			},
			wantErr: "type",
		},
		{
			name: "empty API key",
			config: func() *Config {
				cfg := createBaseTestConfig()
				provider := cfg.Providers["openai"]
				provider.APIKey = NewSecureString("")
				cfg.Providers["openai"] = provider
				return cfg
			}(),
		},
		{
			name: "invalid temperature range",
			config: func() *Config {
				cfg := createBaseTestConfig()
				cfg.Settings.Temperature = 5.0 // Invalid: > 2.0
				return cfg
			}(),
			wantErr: "temperature",
		},
		{
			name: "invalid max tokens",
			config: func() *Config {
				cfg := createBaseTestConfig()
				cfg.Settings.MaxTokens = -100 // Invalid: < 1
				return cfg
			}(),
			wantErr: "maxtokens",
		},
		{
			name: "nonexistent default provider",
			config: func() *Config {
				cfg := createBaseTestConfig()
				cfg.DefaultProvider = "nonexistent"
				return cfg
			}(),
			wantErr: "provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetProvider(t *testing.T) {
	config := &Config{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:   "openai",
				APIKey: NewSecureString("sk-openai-key"),
			},
			"anthropic": {
				Type:   "anthropic",
				APIKey: NewSecureString("sk-anthropic-key"),
			},
		},
	}

	tests := []struct {
		name         string
		provider     string
		expectedType string
		wantErr      string
	}{
		{
			name:         "get existing provider",
			provider:     "openai",
			expectedType: "openai",
		},
		{
			name:         "get another existing provider",
			provider:     "anthropic",
			expectedType: "anthropic",
		},
		{
			name:     "get nonexistent provider",
			provider: "nonexistent",
			wantErr:  "provider not found",
		},
		{
			name:         "get default provider with empty string",
			provider:     "",
			expectedType: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerConfig, actualProvider, err := config.GetProvider(tt.provider)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, providerConfig.Type)

			if tt.provider == "" {
				assert.Equal(t, config.DefaultProvider, actualProvider)
			} else {
				assert.Equal(t, tt.provider, actualProvider)
			}
		})
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	config := &Config{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:       "openai",
				APIKey:     NewSecureString("sk-test123"),
				BaseURL:    "https://api.openai.com/v1",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				TLSVerify:  true,
			},
		},
		Settings: GlobalSettings{
			MaxTokens:     2048,
			Temperature:   0.7,
			Timeout:       30 * time.Second,
			RetryAttempts: 3,
			RetryDelay:    time.Second,
			OutputFormat:  "json",
		},
	}

	// Apply defaults to ensure validation passes
	config.Settings.ApplyDefaults()
	config.Logging.ApplyDefaults()
	config.Security.ApplyDefaults()
	config.Cache.ApplyDefaults()
	config.Plugins.ApplyDefaults()
	config.MCP.ApplyDefaults()

	err := config.Save(configFile)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configFile)

	// Load and verify content
	loadedConfig, err := LoadFromFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, config.DefaultProvider, loadedConfig.DefaultProvider)
	assert.Equal(t, config.Providers["openai"].Type, loadedConfig.Providers["openai"].Type)
}

func TestSecureStringHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular API key",
			input:    "sk-1234567890abcdef",
			expected: "***REDACTED***",
		},
		{
			name:     "empty API key",
			input:    "",
			expected: "",
		},
		{
			name:     "long API key",
			input:    strings.Repeat("x", 100),
			expected: "***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secureStr := NewSecureString(tt.input)

			// Test String() method redacts non-empty keys
			result := secureStr.String()
			if tt.input == "" {
				assert.Equal(t, "", result)
			} else {
				assert.Equal(t, "***REDACTED***", result)
			}

			// Test that original value can still be retrieved internally
			if tt.input != "" {
				assert.True(t, len(secureStr.data) > 0)
			}
		})
	}
}

func TestConfigurationFileDiscovery(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(string) string // Returns config file path or empty string if none created
		expectFound bool
	}{
		{
			name: "current directory config",
			setupFiles: func(tmpDir string) string {
				configPath := filepath.Join(tmpDir, "config.yaml")
				content := `
default_provider: openai
providers:
  openai:
    type: openai
    api_key: sk-test
`
				os.WriteFile(configPath, []byte(content), 0644)
				return configPath
			},
			expectFound: true,
		},
		{
			name: "home directory config",
			setupFiles: func(tmpDir string) string {
				gollmDir := filepath.Join(tmpDir, ".gollm")
				os.MkdirAll(gollmDir, 0755)
				configPath := filepath.Join(gollmDir, "config.yaml")
				content := `
default_provider: anthropic
providers:
  anthropic:
    type: anthropic
    api_key: sk-test
`
				os.WriteFile(configPath, []byte(content), 0644)
				return configPath
			},
			expectFound: true,
		},
		{
			name: "no config file found",
			setupFiles: func(tmpDir string) string {
				// Don't create any config files
				return ""
			},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := tt.setupFiles(tmpDir)

			var config *Config
			var err error

			if configPath != "" {
				// Load from specific file
				config, err = LoadFromFile(configPath)
			} else {
				// Test loading with no config file (should use defaults)
				originalConfigPaths := getConfigPaths()
				setConfigPathsForTesting([]string{tmpDir}) // Empty directory
				defer setConfigPathsForTesting(originalConfigPaths)

				config, err = Load()
			}

			if tt.expectFound {
				require.NoError(t, err)
				require.NotNil(t, config)
			} else {
				// When no config file is found, validation should fail because no providers are configured
				require.Error(t, err)
				assert.Contains(t, err.Error(), "at least one provider must be configured")
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	base := &Config{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:   "openai",
				APIKey: NewSecureString("sk-base"),
			},
		},
		Settings: GlobalSettings{
			MaxTokens:   1000,
			Temperature: 0.5,
		},
	}

	override := &Config{
		DefaultProvider: "anthropic",
		Providers: map[string]ProviderConfig{
			"anthropic": {
				Type:   "anthropic",
				APIKey: NewSecureString("sk-override"),
			},
		},
		Settings: GlobalSettings{
			Temperature: 0.8,
			TopP:        0.9,
		},
	}

	merged := base.Merge(override)

	// Check that override values take precedence
	assert.Equal(t, "anthropic", merged.DefaultProvider)
	assert.Equal(t, 0.8, merged.Settings.Temperature)
	assert.Equal(t, 0.9, merged.Settings.TopP)

	// Check that base values are preserved when not overridden
	assert.Equal(t, 1000, merged.Settings.MaxTokens)

	// Check that both providers exist
	assert.Contains(t, merged.Providers, "openai")
	assert.Contains(t, merged.Providers, "anthropic")
}

func TestGlobalSettings_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    GlobalSettings
		expected GlobalSettings
	}{
		{
			name:  "empty settings should get defaults",
			input: GlobalSettings{},
			expected: GlobalSettings{
				MaxTokens:        2048,
				Temperature:      0.7,
				TopP:             1.0,
				FrequencyPenalty: 0.0,
				PresencePenalty:  0.0,
				Timeout:          30 * time.Second,
				RetryAttempts:    3,
				RetryDelay:       time.Second,
				UserAgent:        "gollm/1.0.0",
				OutputFormat:     "text",
			},
		},
		{
			name: "partial settings should preserve values",
			input: GlobalSettings{
				MaxTokens:   4000,
				Temperature: 0.9,
			},
			expected: GlobalSettings{
				MaxTokens:        4000, // Preserved
				Temperature:      0.9,  // Preserved
				TopP:             1.0,  // Default
				FrequencyPenalty: 0.0,  // Default
				PresencePenalty:  0.0,  // Default
				Timeout:          30 * time.Second,
				RetryAttempts:    3,
				RetryDelay:       time.Second,
				UserAgent:        "gollm/1.0.0",
				OutputFormat:     "text",
			},
		},
		{
			name: "complete settings should remain unchanged",
			input: GlobalSettings{
				MaxTokens:        1000,
				Temperature:      0.3,
				TopP:             0.8,
				FrequencyPenalty: 0.5,
				PresencePenalty:  0.2,
				Timeout:          60 * time.Second,
				RetryAttempts:    5,
				RetryDelay:       2 * time.Second,
				UserAgent:        "custom-agent",
				OutputFormat:     "json",
			},
			expected: GlobalSettings{
				MaxTokens:        1000,
				Temperature:      0.3,
				TopP:             0.8,
				FrequencyPenalty: 0.5,
				PresencePenalty:  0.2,
				Timeout:          60 * time.Second,
				RetryAttempts:    5,
				RetryDelay:       2 * time.Second,
				UserAgent:        "custom-agent",
				OutputFormat:     "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.ApplyDefaults()
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

func TestProviderConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: ProviderConfig{
				Type:       "openai",
				APIKey:     NewSecureString("sk-test123"),
				BaseURL:    "https://api.openai.com/v1",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				TLSVerify:  true,
			},
		},
		{
			name: "missing type",
			config: ProviderConfig{
				APIKey: NewSecureString("sk-test123"),
			},
			wantErr: "type",
		},
		{
			name: "invalid URL",
			config: ProviderConfig{
				Type:    "openai",
				APIKey:  NewSecureString("sk-test123"),
				BaseURL: "not-a-valid-url",
			},
			wantErr: "baseurl",
		},
		{
			name: "invalid max retries",
			config: ProviderConfig{
				Type:       "openai",
				APIKey:     NewSecureString("sk-test123"),
				MaxRetries: -1,
			},
			wantErr: "maxretries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.wantErr))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions for testing

// createValidTestConfig creates a fully valid configuration for testing.
// This ensures all required fields have valid values that pass validation.
func createValidTestConfig() *Config {
	config := &Config{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:       "openai",
				APIKey:     NewSecureString("sk-test123"),
				BaseURL:    "https://api.openai.com/v1",
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				TLSVerify:  true,
			},
		},
		Settings: GlobalSettings{
			MaxTokens:        2048,
			Temperature:      0.7,
			TopP:             1.0,
			FrequencyPenalty: 0.0,
			PresencePenalty:  0.0,
			Timeout:          30 * time.Second,
			RetryAttempts:    3,
			RetryDelay:       time.Second,
			UserAgent:        "gollm/1.0.0",
			OutputFormat:     "text",
		},
		Features: FeatureFlags{
			Streaming:    false,
			Caching:      false,
			Plugins:      false,
			MCP:          false,
			Metrics:      false,
			HealthChecks: false,
			AutoRetry:    false,
			Compression:  false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     30,
		},
		Security: SecurityConfig{
			TLSMinVersion:      "1.3",
			CertificatePinning: false,
			TokenRotation:      false,
			EncryptConfig:      false,
			SessionTimeout:     2 * time.Hour,
			MaxRequestSize:     10 * 1024 * 1024,
		},
		Cache: CacheConfig{
			Enabled:     false,
			Type:        "memory",
			TTL:         time.Hour,
			MaxSize:     100 * 1024 * 1024,
			Compression: false,
		},
		Plugins: PluginConfig{
			Enabled:   false,
			AutoLoad:  false,
			Timeout:   30 * time.Second,
			MaxMemory: 64 * 1024 * 1024,
		},
		MCP: MCPConfig{
			Enabled:      false,
			Timeout:      30 * time.Second,
			MaxRetries:   3,
			HealthChecks: false,
		},
	}
	return config
}

// createBaseTestConfig creates a basic configuration with defaults applied.
// This is useful for tests that need a valid base config but want to override specific fields.
func createBaseTestConfig() *Config {
	config := &Config{
		DefaultProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:       "openai",
				APIKey:     NewSecureString("sk-test123"),
				MaxRetries: 3,
				Timeout:    30 * time.Second,
				TLSVerify:  true,
			},
		},
	}

	// Apply defaults to all sections
	config.Settings.ApplyDefaults()
	config.Features.ApplyDefaults()
	config.Logging.ApplyDefaults()
	config.Security.ApplyDefaults()
	config.Cache.ApplyDefaults()
	config.Plugins.ApplyDefaults()
	config.MCP.ApplyDefaults()

	return config
}
