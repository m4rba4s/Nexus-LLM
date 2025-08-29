// Package commands provides configuration management commands for GOLLM.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/yourusername/gollm/internal/config"
)

// ConfigFlags contains flags specific to config commands.
type ConfigFlags struct {
	OutputFormat string
	ConfigFile   string
	Force        bool
	Global       bool
}

// NewConfigCommand creates the config command with subcommands.
func NewConfigCommand() *cobra.Command {
	flags := &ConfigFlags{}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage GOLLM configuration",
		Long: `Manage GOLLM configuration files and settings.

The config command allows you to initialize, view, and modify GOLLM configuration.
Configuration is loaded from the first file found in this order:
  1. --config flag or CONFIG_FILE environment variable
  2. ./config.yaml
  3. ~/.gollm/config.yaml
  4. /etc/gollm/config.yaml

Examples:
  # Initialize a new config file
  gollm config init

  # Show current configuration
  gollm config show

  # Set a configuration value
  gollm config set providers.openai.api_key "sk-..."

  # Get a configuration value
  gollm config get providers.openai.api_key

  # Validate current configuration
  gollm config validate

  # List configured providers
  gollm config providers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add global flags for config commands
	addConfigFlags(cmd, flags)

	// Add subcommands
	cmd.AddCommand(newConfigInitCommand(flags))
	cmd.AddCommand(newConfigShowCommand(flags))
	cmd.AddCommand(newConfigSetCommand(flags))
	cmd.AddCommand(newConfigGetCommand(flags))
	cmd.AddCommand(newConfigValidateCommand(flags))
	cmd.AddCommand(newConfigProvidersCommand(flags))

	return cmd
}

// addConfigFlags adds common flags for config commands.
func addConfigFlags(cmd *cobra.Command, flags *ConfigFlags) {
	f := cmd.PersistentFlags()

	f.StringVarP(&flags.OutputFormat, "output", "o", "yaml",
		"output format (yaml, json, text)")
	f.StringVar(&flags.ConfigFile, "config-file", "",
		"config file path (default: auto-detect)")
	f.BoolVar(&flags.Force, "force", false,
		"overwrite existing configuration")
	f.BoolVar(&flags.Global, "global", false,
		"use global configuration file")
}

// newConfigInitCommand creates the 'config init' subcommand.
func newConfigInitCommand(flags *ConfigFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long: `Initialize a new GOLLM configuration file with default settings.

This command creates a new configuration file with sensible defaults and
example provider configurations. You can then edit the file to add your
API keys and customize settings.

Examples:
  # Initialize config in current directory
  gollm config init

  # Initialize global config
  gollm config init --global

  # Initialize with specific file path
  gollm config init --config-file ~/my-gollm-config.yaml

  # Force overwrite existing config
  gollm config init --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(flags)
		},
	}
}

// newConfigShowCommand creates the 'config show' subcommand.
func newConfigShowCommand(flags *ConfigFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Display the current GOLLM configuration.

This command loads and displays the current configuration, showing which
file was loaded and all configured settings. Sensitive values like API
keys are masked for security.

Examples:
  # Show config in YAML format
  gollm config show

  # Show config in JSON format
  gollm config show --output json

  # Show config from specific file
  gollm config show --config-file ~/my-config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(flags)
		},
	}
}

// newConfigSetCommand creates the 'config set' subcommand.
func newConfigSetCommand(flags *ConfigFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value using dot notation for nested keys.

Examples:
  # Set default provider
  gollm config set default_provider openai

  # Set OpenAI API key
  gollm config set providers.openai.api_key "sk-..."

  # Set global temperature
  gollm config set settings.temperature 0.7

  # Set logging level
  gollm config set logging.level debug`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1], flags)
		},
	}
}

// newConfigGetCommand creates the 'config get' subcommand.
func newConfigGetCommand(flags *ConfigFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value using dot notation for nested keys.

Examples:
  # Get default provider
  gollm config get default_provider

  # Get OpenAI base URL
  gollm config get providers.openai.base_url

  # Get all provider settings
  gollm config get providers`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(args[0], flags)
		},
	}
}

// newConfigValidateCommand creates the 'config validate' subcommand.
func newConfigValidateCommand(flags *ConfigFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long: `Validate the current configuration for syntax errors and required fields.

This command checks the configuration file for:
  - Valid YAML/JSON syntax
  - Required fields
  - Valid provider configurations
  - Proper data types and ranges
  - API key format validation

Examples:
  # Validate current config
  gollm config validate

  # Validate specific config file
  gollm config validate --config-file ~/test-config.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigValidate(flags)
		},
	}
}

// newConfigProvidersCommand creates the 'config providers' subcommand.
func newConfigProvidersCommand(flags *ConfigFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List configured providers",
		Long: `List all configured LLM providers and their status.

This command shows all configured providers, their types, and
basic connectivity status.

Examples:
  # List all providers
  gollm config providers

  # List providers in JSON format
  gollm config providers --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigProviders(flags)
		},
	}
}

// runConfigInit initializes a new configuration file.
func runConfigInit(flags *ConfigFlags) error {
	configPath, err := determineConfigPath(flags)
	if err != nil {
		return fmt.Errorf("failed to determine config path: %w", err)
	}

	// Check if file exists and force flag is not set
	if _, err := os.Stat(configPath); err == nil && !flags.Force {
		return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate default configuration
	defaultConfig := generateDefaultConfig()

	// Write configuration to file
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration file created at: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the configuration file to add your API keys")
	fmt.Println("2. Run 'gollm config validate' to check your configuration")
	fmt.Println("3. Run 'gollm config show' to view your current settings")
	fmt.Println("4. Start using GOLLM with 'gollm chat \"Hello, world!\"'")

	return nil
}

// runConfigShow displays the current configuration.
func runConfigShow(flags *ConfigFlags) error {
	cfg, err := loadConfigWithOptions(flags)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Mask sensitive values
	maskedConfig := maskSensitiveValues(cfg)

	switch flags.OutputFormat {
	case "json":
		return outputJSON(maskedConfig)
	case "yaml":
		return outputYAML(maskedConfig)
	case "text":
		return outputText(maskedConfig)
	default:
		return fmt.Errorf("unsupported output format: %s", flags.OutputFormat)
	}
}

// runConfigSet sets a configuration value.
func runConfigSet(key, value string, flags *ConfigFlags) error {
	// This is a simplified implementation
	// In a full implementation, you'd load the config, modify it, and save it
	fmt.Printf("Setting %s = %s\n", key, value)
	fmt.Println("Note: config set functionality is not yet implemented")
	fmt.Println("Please edit the configuration file directly")
	return nil
}

// runConfigGet gets a configuration value.
func runConfigGet(key string, flags *ConfigFlags) error {
	cfg, err := loadConfigWithOptions(flags)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// This is a simplified implementation
	// In a full implementation, you'd use reflection or a library like mapstructure
	// to navigate the nested structure
	fmt.Printf("Getting value for key: %s\n", key)
	fmt.Printf("Config loaded successfully, but get functionality is not yet implemented\n")
	fmt.Printf("Current default provider: %s\n", cfg.DefaultProvider)

	return nil
}

// runConfigValidate validates the current configuration.
func runConfigValidate(flags *ConfigFlags) error {
	cfg, err := loadConfigWithOptions(flags)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	fmt.Println("✓ Configuration is valid")
	fmt.Printf("✓ Found %d configured provider(s)\n", len(cfg.Providers))

	// Additional validation checks
	if cfg.DefaultProvider != "" {
		if cfg.HasProvider(cfg.DefaultProvider) {
			fmt.Printf("✓ Default provider '%s' is configured\n", cfg.DefaultProvider)
		} else {
			fmt.Printf("⚠ Default provider '%s' is not configured\n", cfg.DefaultProvider)
		}
	} else {
		fmt.Println("⚠ No default provider set")
	}

	return nil
}

// runConfigProviders lists configured providers.
func runConfigProviders(flags *ConfigFlags) error {
	cfg, err := loadConfigWithOptions(flags)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	providers := cfg.ListProviders()
	if len(providers) == 0 {
		fmt.Println("No providers configured")
		return nil
	}

	fmt.Printf("Configured providers (%d):\n\n", len(providers))

	for _, name := range providers {
		providerConfig, _, err := cfg.GetProvider(name)
		if err != nil {
			continue
		}

		status := "configured"
		if providerConfig.APIKey.IsEmpty() {
			status = "missing API key"
		}

		defaultMark := ""
		if name == cfg.DefaultProvider {
			defaultMark = " (default)"
		}

		fmt.Printf("  %s%s\n", name, defaultMark)
		fmt.Printf("    Type: %s\n", providerConfig.Type)
		fmt.Printf("    Status: %s\n", status)
		if providerConfig.BaseURL != "" {
			fmt.Printf("    Base URL: %s\n", providerConfig.BaseURL)
		}
		if providerConfig.DefaultModel != "" {
			fmt.Printf("    Default Model: %s\n", providerConfig.DefaultModel)
		}
		fmt.Println()
	}

	return nil
}

// Helper functions

// determineConfigPath determines where to create/load the config file.
func determineConfigPath(flags *ConfigFlags) (string, error) {
	if flags.ConfigFile != "" {
		return flags.ConfigFile, nil
	}

	if flags.Global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, ".gollm", "config.yaml"), nil
	}

	// Default to current directory
	return "config.yaml", nil
}

// loadConfigWithOptions loads config with specified options.
func loadConfigWithOptions(flags *ConfigFlags) (*config.Config, error) {
	opts := config.LoadOptions{}
	if flags.ConfigFile != "" {
		opts.ConfigFile = flags.ConfigFile
	}

	return config.LoadWithOptions(opts)
}

// generateDefaultConfig creates a default configuration structure.
func generateDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"default_provider": "openai",
		"providers": map[string]interface{}{
			"openai": map[string]interface{}{
				"type":     "openai",
				"api_key":  "your-openai-api-key-here",
				"base_url": "https://api.openai.com/v1",
				"max_retries": 3,
				"timeout": "30s",
				"default_model": "gpt-3.5-turbo",
			},
			"anthropic": map[string]interface{}{
				"type":     "anthropic",
				"api_key":  "your-anthropic-api-key-here",
				"base_url": "https://api.anthropic.com",
				"max_retries": 3,
				"timeout": "30s",
				"default_model": "claude-3-sonnet-20240229",
			},
			"mock": map[string]interface{}{
				"type":     "mock",
				"api_key":  "mock-api-key",
				"timeout": "1s",
				"default_model": "mock-gpt-3.5-turbo",
				"extra": map[string]interface{}{
					"latency": "100ms",
					"default_response": "Hello! This is a mock response for testing GOLLM.",
				},
			},
		},
		"settings": map[string]interface{}{
			"max_tokens":   2048,
			"temperature":  0.7,
			"timeout":      "60s",
			"output_format": "text",
		},
		"features": map[string]interface{}{
			"streaming": true,
			"caching":   false,
			"metrics":   true,
		},
		"logging": map[string]interface{}{
			"level":  "info",
			"format": "text",
		},
	}
}

// maskSensitiveValues masks sensitive configuration values for display.
func maskSensitiveValues(cfg *config.Config) *config.Config {
	// Create a copy and mask API keys
	masked := *cfg
	maskedProviders := make(map[string]config.ProviderConfig)

	for name, provider := range cfg.Providers {
		maskedProvider := provider
		if !provider.APIKey.IsEmpty() {
			// Create a masked version of the API key
			apiKey := provider.APIKey.Value()
			if len(apiKey) > 8 {
				maskedProvider.APIKey = config.NewSecureString(apiKey[:4] + "..." + apiKey[len(apiKey)-4:])
			} else {
				maskedProvider.APIKey = config.NewSecureString("***")
			}
		}
		maskedProviders[name] = maskedProvider
	}

	masked.Providers = maskedProviders
	return &masked
}

// outputJSON outputs configuration in JSON format.
func outputJSON(cfg *config.Config) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(cfg)
}

// outputYAML outputs configuration in YAML format.
func outputYAML(cfg *config.Config) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(cfg)
}

// outputText outputs configuration in human-readable text format.
func outputText(cfg *config.Config) error {
	fmt.Printf("GOLLM Configuration\n")
	fmt.Printf("==================\n\n")

	fmt.Printf("Default Provider: %s\n\n", cfg.DefaultProvider)

	fmt.Printf("Providers (%d configured):\n", len(cfg.Providers))
	for name, provider := range cfg.Providers {
		defaultMark := ""
		if name == cfg.DefaultProvider {
			defaultMark = " (default)"
		}

		fmt.Printf("  %s%s:\n", name, defaultMark)
		fmt.Printf("    Type: %s\n", provider.Type)
		if !provider.APIKey.IsEmpty() {
			fmt.Printf("    API Key: %s\n", provider.APIKey.String())
		}
		if provider.BaseURL != "" {
			fmt.Printf("    Base URL: %s\n", provider.BaseURL)
		}
		if provider.DefaultModel != "" {
			fmt.Printf("    Default Model: %s\n", provider.DefaultModel)
		}
		fmt.Printf("    Max Retries: %d\n", provider.MaxRetries)
		fmt.Printf("    Timeout: %s\n", provider.Timeout)
		fmt.Println()
	}

	fmt.Printf("Global Settings:\n")
	fmt.Printf("  Max Tokens: %d\n", cfg.Settings.MaxTokens)
	fmt.Printf("  Temperature: %.1f\n", cfg.Settings.Temperature)
	fmt.Printf("  Timeout: %s\n", cfg.Settings.Timeout)
	fmt.Printf("  Output Format: %s\n", cfg.Settings.OutputFormat)
	fmt.Println()

	fmt.Printf("Features:\n")
	fmt.Printf("  Streaming: %t\n", cfg.Features.Streaming)
	fmt.Printf("  Caching: %t\n", cfg.Features.Caching)
	fmt.Printf("  Metrics: %t\n", cfg.Features.Metrics)
	fmt.Println()

	fmt.Printf("Logging:\n")
	fmt.Printf("  Level: %s\n", cfg.Logging.Level)
	fmt.Printf("  Format: %s\n", cfg.Logging.Format)

	return nil
}
