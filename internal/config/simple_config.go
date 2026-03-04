package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// Save saves the configuration to the default location
func Save(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	// Get config directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".gollm")
	configFile := filepath.Join(configDir, "config.yaml")

	// Create directory if not exists
    if err := os.MkdirAll(configDir, 0700); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

	// Marshal config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
    if err := os.WriteFile(configFile, data, 0600); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    return nil
}
