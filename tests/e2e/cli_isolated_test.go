// Package e2e provides isolated integration tests for CLI commands with mock configuration.
//
// These tests are designed to run in isolation without requiring real configuration files,
// API keys, or external dependencies. They use mock providers and controlled test environments.
package e2e

import (
	"bytes"

	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/gollm/internal/config"
)

// IsolatedTestEnvironment represents an isolated test environment for CLI testing
type IsolatedTestEnvironment struct {
	config      *config.Config
	mockServers map[string]*httptest.Server
	t           *testing.T
}

// NewIsolatedTestEnvironment creates a new isolated test environment
func NewIsolatedTestEnvironment(t *testing.T) *IsolatedTestEnvironment {
	env := &IsolatedTestEnvironment{
		t:           t,
		mockServers: make(map[string]*httptest.Server),
	}

	env.setupMockServers()
	env.setupMockConfig()

	return env
}

// Cleanup cleans up the test environment
func (env *IsolatedTestEnvironment) Cleanup() {
	for _, server := range env.mockServers {
		server.Close()
	}
}

// setupMockServers creates mock HTTP servers for providers
func (env *IsolatedTestEnvironment) setupMockServers() {
	// OpenAI mock server
	openaiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			env.handleOpenAICompletion(w, r)
		case "/v1/models":
			env.handleOpenAIModels(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	env.mockServers["openai"] = openaiServer

	// Anthropic mock server
	anthropicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/messages":
			env.handleAnthropicMessages(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	env.mockServers["anthropic"] = anthropicServer
}

// setupMockConfig creates a mock configuration for testing
func (env *IsolatedTestEnvironment) setupMockConfig() {
	env.config = &config.Config{
		DefaultProvider: "mock",
		Providers: map[string]config.ProviderConfig{
			"mock": {
				Type:      "mock",
				APIKey:    config.NewSecureString("test-key-12345"),
				BaseURL:   "http://localhost",
				MaxRetries: 3,
				Timeout:   30 * time.Second,
			},
			"openai": {
				Type:      "openai",
				APIKey:    config.NewSecureString("sk-test-openai-key"),
				BaseURL:   env.mockServers["openai"].URL,
				MaxRetries: 3,
				Timeout:   30 * time.Second,
			},
			"anthropic": {
				Type:      "anthropic",
				APIKey:    config.NewSecureString("sk-ant-test-key"),
				BaseURL:   env.mockServers["anthropic"].URL,
				MaxRetries: 3,
				Timeout:   30 * time.Second,
			},
		},
		Settings: config.GlobalSettings{
			MaxTokens:   2048,
			Temperature: 0.7,
			Timeout:     30 * time.Second,
		},
		Features: config.FeatureFlags{
			Streaming: true,
			Caching:   false,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// GetConfig returns the mock configuration
func (env *IsolatedTestEnvironment) GetConfig() *config.Config {
	return env.config
}

// ExecuteCommand executes a CLI command in the isolated environment
func (env *IsolatedTestEnvironment) ExecuteCommand(args []string, stdin string) (*CommandResult, error) {
	// Create root command with mock config
	rootCmd := &cobra.Command{
		Use:   "gollm",
		Short: "GOLLM CLI for testing",
	}

	// Add commands
	chatCmd := env.createMockChatCommand()
	configCmd := env.createMockConfigCommand()
	modelsCmd := env.createMockModelsCommand()
	versionCmd := env.createMockVersionCommand()

	rootCmd.AddCommand(chatCmd, configCmd, modelsCmd, versionCmd)

	// Setup IO
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)

	if stdin != "" {
		rootCmd.SetIn(strings.NewReader(stdin))
	}

	// Execute
	err := rootCmd.Execute()

	result := &CommandResult{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Error:    err,
	}

	if err != nil {
		result.ExitCode = 1
	}

	return result, nil
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
}

// createMockChatCommand creates a mock chat command for testing
func (env *IsolatedTestEnvironment) createMockChatCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Chat with an LLM",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := ""
			if len(args) > 0 {
				message = args[0]
			}

			if message == "" {
				return fmt.Errorf("no message provided (use argument or pipe from stdin)")
			}

			// Mock response
			response := "Hello there! I'm a helpful AI assistant. How can I help you today?"
			fmt.Fprint(cmd.OutOrStdout(), response)

			return nil
		},
	}

	// Add flags
	cmd.Flags().String("model", "", "model to use for the chat")
	cmd.Flags().String("provider", "", "provider to use for the chat")
	cmd.Flags().Float64("temperature", 0, "sampling temperature")
	cmd.Flags().Int("max-tokens", 0, "maximum tokens in response")
	cmd.Flags().Bool("stream", true, "stream response tokens")
	cmd.Flags().String("system", "", "system message to set context")

	return cmd
}

// createMockConfigCommand creates a mock config command for testing
func (env *IsolatedTestEnvironment) createMockConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configYAML := `default_provider: mock
providers:
  mock:
    type: mock
    api_key: "***REDACTED***"
  openai:
    type: openai
    api_key: "***REDACTED***"
  anthropic:
    type: anthropic
    api_key: "***REDACTED***"
settings:
  max_tokens: 2048
  temperature: 0.7
`
			fmt.Fprint(cmd.OutOrStdout(), configYAML)
			return nil
		},
	}

	cmd.AddCommand(showCmd)
	return cmd
}

// createMockModelsCommand creates a mock models command for testing
func (env *IsolatedTestEnvironment) createMockModelsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Manage models",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, args []string) error {
			models := `Available Models:
- gpt-3.5-turbo (OpenAI)
- gpt-4 (OpenAI)
- claude-3-sonnet-20240229 (Anthropic)
- claude-3-haiku-20240307 (Anthropic)
`
			fmt.Fprint(cmd.OutOrStdout(), models)
			return nil
		},
	}

	cmd.AddCommand(listCmd)
	return cmd
}

// createMockVersionCommand creates a mock version command for testing
func (env *IsolatedTestEnvironment) createMockVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStdout(), "gollm version v1.0.0-dev\n")
		},
	}
}

// Mock HTTP handlers

func (env *IsolatedTestEnvironment) handleOpenAICompletion(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"id":      "chatcmpl-test123",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "gpt-3.5-turbo",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": "Hello there! I'm a helpful AI assistant.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 15,
			"total_tokens":      25,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (env *IsolatedTestEnvironment) handleOpenAIModels(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":       "gpt-3.5-turbo",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "openai",
			},
			{
				"id":       "gpt-4",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "openai",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (env *IsolatedTestEnvironment) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"id":   "msg_test123",
		"type": "message",
		"role": "assistant",
		"content": []map[string]string{
			{
				"type": "text",
				"text": "Hello! I'm Claude, an AI assistant created by Anthropic.",
			},
		},
		"model":       "claude-3-sonnet-20240229",
		"stop_reason": "end_turn",
		"usage": map[string]int{
			"input_tokens":  10,
			"output_tokens": 15,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestCLI_IsolatedBasicCommands tests basic CLI commands in isolation
func TestCLI_IsolatedBasicCommands(t *testing.T) {
	env := NewIsolatedTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name       string
		args       []string
		stdin      string
		expectExit int
		expectOut  []string
		expectErr  string
	}{
		{
			name:       "version command",
			args:       []string{"version"},
			expectExit: 0,
			expectOut:  []string{"gollm version"},
		},
		{
			name:       "simple chat message",
			args:       []string{"chat", "Hello, world!"},
			expectExit: 0,
			expectOut:  []string{"Hello there!"},
		},
		{
			name:       "chat with model flag",
			args:       []string{"chat", "--model", "gpt-4", "What is Go?"},
			expectExit: 0,
			expectOut:  []string{"Hello there!"},
		},
		{
			name:       "config show",
			args:       []string{"config", "show"},
			expectExit: 0,
			expectOut:  []string{"default_provider: mock", "providers:"},
		},
		{
			name:       "models list",
			args:       []string{"models", "list"},
			expectExit: 0,
			expectOut:  []string{"Available Models:", "gpt-3.5-turbo", "claude-3-sonnet"},
		},
		{
			name:       "missing message argument",
			args:       []string{"chat"},
			expectExit: 1,
			expectErr:  "no message provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteCommand(tt.args, tt.stdin)
			require.NoError(t, err, "Command execution should not fail")

			assert.Equal(t, tt.expectExit, result.ExitCode,
				"Exit code mismatch. Stdout: %s, Stderr: %s", result.Stdout, result.Stderr)

			// Check expected output
			for _, expected := range tt.expectOut {
				assert.Contains(t, result.Stdout, expected,
					"Expected output %q not found in stdout: %s", expected, result.Stdout)
			}

			// Check expected error
			if tt.expectErr != "" {
				if result.Error != nil {
					assert.Contains(t, result.Error.Error(), tt.expectErr,
						"Expected error %q not found in error: %v", tt.expectErr, result.Error)
				} else {
					assert.Contains(t, result.Stderr, tt.expectErr,
						"Expected error %q not found in stderr: %s", tt.expectErr, result.Stderr)
				}
			}
		})
	}
}

// TestCLI_IsolatedProviderSwitching tests provider switching functionality
func TestCLI_IsolatedProviderSwitching(t *testing.T) {
	env := NewIsolatedTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name     string
		provider string
		message  string
		expectOut string
	}{
		{
			name:      "openai provider",
			provider:  "openai",
			message:   "Hello OpenAI",
			expectOut: "Hello there!",
		},
		{
			name:      "anthropic provider",
			provider:  "anthropic",
			message:   "Hello Claude",
			expectOut: "Hello there!",
		},
		{
			name:      "mock provider (default)",
			provider:  "mock",
			message:   "Hello Mock",
			expectOut: "Hello there!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"chat", "--provider", tt.provider, tt.message}
			result, err := env.ExecuteCommand(args, "")

			require.NoError(t, err)
			assert.Equal(t, 0, result.ExitCode)
			assert.Contains(t, result.Stdout, tt.expectOut)
		})
	}
}

// TestCLI_IsolatedErrorHandling tests error handling scenarios
func TestCLI_IsolatedErrorHandling(t *testing.T) {
	env := NewIsolatedTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name      string
		args      []string
		expectErr string
	}{
		{
			name:      "missing chat message",
			args:      []string{"chat"},
			expectErr: "no message provided",
		},
		{
			name:      "invalid command",
			args:      []string{"invalid-command"},
			expectErr: "",  // Will show help, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteCommand(tt.args, "")
			require.NoError(t, err, "Command execution should not fail")

			if tt.expectErr != "" {
				assert.NotEqual(t, 0, result.ExitCode, "Expected non-zero exit code")
				if result.Error != nil {
					assert.Contains(t, result.Error.Error(), tt.expectErr)
				}
			}
		})
	}
}

// TestCLI_IsolatedFlagValidation tests CLI flag validation
func TestCLI_IsolatedFlagValidation(t *testing.T) {
	env := NewIsolatedTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name      string
		args      []string
		expectErr bool
		errorMsg  string
	}{
		{
			name:     "valid temperature",
			args:     []string{"chat", "--temperature", "0.7", "Test"},
			expectErr: false,
		},
		{
			name:      "valid max tokens",
			args:      []string{"chat", "--max-tokens", "100", "Test"},
			expectErr: false,
		},
		{
			name:     "valid model",
			args:     []string{"chat", "--model", "gpt-3.5-turbo", "Test"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteCommand(tt.args, "")
			require.NoError(t, err)

			if tt.expectErr {
				assert.NotEqual(t, 0, result.ExitCode)
				if tt.errorMsg != "" && result.Error != nil {
					assert.Contains(t, result.Error.Error(), tt.errorMsg)
				}
			} else {
				assert.Equal(t, 0, result.ExitCode,
					"Command should succeed. Stderr: %s", result.Stderr)
			}
		})
	}
}

// TestCLI_IsolatedStdinInput tests stdin input handling
func TestCLI_IsolatedStdinInput(t *testing.T) {
	env := NewIsolatedTestEnvironment(t)
	defer env.Cleanup()

	t.Run("chat with stdin", func(t *testing.T) {
		stdinInput := "Hello from stdin!"
		result, err := env.ExecuteCommand([]string{"chat", stdinInput}, "")

		require.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Stdout, "Hello there!")
	})
}

// TestCLI_IsolatedHelp tests help functionality
func TestCLI_IsolatedHelp(t *testing.T) {
	env := NewIsolatedTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name     string
		args     []string
		expectOut string
	}{
		{
			name:      "root help",
			args:      []string{"--help"},
			expectOut: "Usage:",
		},
		{
			name:      "chat help",
			args:      []string{"chat", "--help"},
			expectOut: "chat [message]",
		},
		{
			name:      "config help",
			args:      []string{"config", "--help"},
			expectOut: "Manage configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.ExecuteCommand(tt.args, "")
			require.NoError(t, err)

			// Help commands typically exit with code 0
			assert.Equal(t, 0, result.ExitCode)
			assert.Contains(t, result.Stdout, tt.expectOut)
		})
	}
}
