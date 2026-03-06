package commands

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/m4rba4s/Nexus-LLM/internal/config"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
	"github.com/m4rba4s/Nexus-LLM/internal/providers/mock"
)

// TestCommand represents a CLI command for testing.
type TestCommand struct {
	Name        string
	Command     *cobra.Command
	Args        []string
	Input       string
	ExpectedOut []string
	ExpectedErr string
	ExitCode    int
	Timeout     time.Duration
}

// TestExecutor executes CLI commands for testing.
type TestExecutor struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	stdin  *bytes.Buffer
}

// NewTestExecutor creates a new test executor.
func NewTestExecutor() *TestExecutor {
	return &TestExecutor{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdin:  &bytes.Buffer{},
	}
}

// Execute executes a command and captures output.
func (te *TestExecutor) Execute(cmd *cobra.Command, args []string, input string) error {
	// Reset buffers
	te.stdout.Reset()
	te.stderr.Reset()
	te.stdin.Reset()

	// Set input if provided
	if input != "" {
		te.stdin.WriteString(input)
	}

	// Set command IO
	cmd.SetOut(te.stdout)
	cmd.SetErr(te.stderr)
	cmd.SetIn(te.stdin)

	// Set args
	cmd.SetArgs(args)

	// Execute command
	return cmd.Execute()
}

// GetOutput returns captured output.
func (te *TestExecutor) GetOutput() (string, string) {
	return te.stdout.String(), te.stderr.String()
}

func TestChatCommand_Basic(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		input       string
		setupConfig func() *config.Config
		expectError bool
		errorMsg    string
		expectOut   []string
	}{
		{
			name:  "simple chat message",
			args:  []string{"Hello, world!"},
			input: "",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Hello there!"},
		},
		{
			name:  "chat with specific model",
			args:  []string{"--model", "test-model", "What is Go?"},
			input: "",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Go is a programming language"},
		},
		{
			name:  "chat with streaming",
			args:  []string{"--stream", "Tell me a story"},
			input: "",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Once upon a time"},
		},
		{
			name:  "chat with system message",
			args:  []string{"--system", "You are a helpful assistant", "Hello"},
			input: "",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Hello there!"},
		},
		{
			name:        "missing message argument",
			args:        []string{},
			input:       "",
			setupConfig: func() *config.Config { return createTestConfig("mock") },
			expectError: true,
			errorMsg:    "message is required",
		},
		{
			name:        "invalid provider",
			args:        []string{"--provider", "invalid", "Hello"},
			input:       "",
			setupConfig: func() *config.Config { return createTestConfig("mock") },
			expectError: true,
			errorMsg:    "provider not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			executor := NewTestExecutor()
			cfg := tt.setupConfig()

			// Create chat command
			cmd := NewChatCommand()
			require.NotNil(t, cmd)

			// Setup mock context for command
			setupMockContext(cfg)

			// Execute command
			err := executor.Execute(cmd, tt.args, tt.input)

			stdout, stderr := executor.GetOutput()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				// Check expected output
				for _, expected := range tt.expectOut {
					assert.Contains(t, stdout, expected, "Expected output not found in stdout")
				}
			}

			// Debug output on failure
			if t.Failed() {
				t.Logf("STDOUT: %s", stdout)
				t.Logf("STDERR: %s", stderr)
			}
		})
	}
}

func TestInteractiveCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		input       string
		setupConfig func() *config.Config
		expectError bool
		expectOut   []string
	}{
		{
			name:  "interactive session",
			args:  []string{},
			input: "Hello\n/quit\n",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Interactive mode", "Hello there!"},
		},
		{
			name:  "interactive with model",
			args:  []string{"--model", "test-model"},
			input: "What is Go?\n/exit\n",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Go is a programming language"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor()
			cfg := tt.setupConfig()

			cmd := NewInteractiveCommand()
			require.NotNil(t, cmd)

			setupMockContext(cfg)

			err := executor.Execute(cmd, tt.args, tt.input)

			stdout, stderr := executor.GetOutput()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				for _, expected := range tt.expectOut {
					assert.Contains(t, stdout, expected)
				}
			}

			if t.Failed() {
				t.Logf("STDOUT: %s", stdout)
				t.Logf("STDERR: %s", stderr)
			}
		})
	}
}

func TestCompleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		input       string
		setupConfig func() *config.Config
		expectError bool
		expectOut   []string
	}{
		{
			name:  "code completion from argument",
			args:  []string{"def fibonacci(n):"},
			input: "",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"if n <= 1:", "return n"},
		},
		{
			name:  "code completion from stdin",
			args:  []string{},
			input: "func factorial(n int) int {\n",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"if n <= 1 {", "return 1"},
		},
		{
			name:  "completion with language hint",
			args:  []string{"--language", "python", "class Calculator:"},
			input: "",
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"def __init__(self):"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor()
			cfg := tt.setupConfig()

			cmd := NewCompleteCommand()
			require.NotNil(t, cmd)

			setupMockContext(cfg)

			err := executor.Execute(cmd, tt.args, tt.input)

			stdout, stderr := executor.GetOutput()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				for _, expected := range tt.expectOut {
					assert.Contains(t, stdout, expected)
				}
			}

			if t.Failed() {
				t.Logf("STDOUT: %s", stdout)
				t.Logf("STDERR: %s", stderr)
			}
		})
	}
}

func TestModelsCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupConfig func() *config.Config
		expectError bool
		expectOut   []string
	}{
		{
			name: "list all models",
			args: []string{"list"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Available models:", "test-model-1", "test-model-2"},
		},
		{
			name: "list models for specific provider",
			args: []string{"list", "--provider", "mock"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"test-model-1", "test-model-2"},
		},
		{
			name: "model info",
			args: []string{"info", "test-model-1"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Model: test-model-1", "Provider: mock"},
		},
		{
			name:        "invalid subcommand",
			args:        []string{"invalid"},
			setupConfig: func() *config.Config { return createTestConfig("mock") },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor()
			cfg := tt.setupConfig()

			cmd := NewModelsCommand()
			require.NotNil(t, cmd)

			setupMockContext(cfg)

			err := executor.Execute(cmd, tt.args, "")

			stdout, stderr := executor.GetOutput()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				for _, expected := range tt.expectOut {
					assert.Contains(t, stdout, expected)
				}
			}

			if t.Failed() {
				t.Logf("STDOUT: %s", stdout)
				t.Logf("STDERR: %s", stderr)
			}
		})
	}
}

func TestConfigCommand(t *testing.T) {
	// Create temporary config file for testing
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	tests := []struct {
		name        string
		args        []string
		setupConfig func() *config.Config
		expectError bool
		expectOut   []string
	}{
		{
			name: "config init",
			args: []string{"init", "--config-file", configFile},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Configuration initialized"},
		},
		{
			name: "config get all",
			args: []string{"get"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"default_provider:", "mock"},
		},
		{
			name: "config get specific key",
			args: []string{"get", "default_provider"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"mock"},
		},
		{
			name: "config set key",
			args: []string{"set", "settings.temperature", "0.8"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Configuration updated"},
		},
		{
			name: "config validate",
			args: []string{"validate"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: false,
			expectOut:   []string{"Configuration is valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor()
			cfg := tt.setupConfig()

			cmd := NewConfigCommand()
			require.NotNil(t, cmd)

			setupMockContext(cfg)

			err := executor.Execute(cmd, tt.args, "")

			stdout, stderr := executor.GetOutput()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				for _, expected := range tt.expectOut {
					assert.Contains(t, stdout, expected)
				}
			}

			if t.Failed() {
				t.Logf("STDOUT: %s", stdout)
				t.Logf("STDERR: %s", stderr)
			}
		})
	}
}

// Test utilities and helpers

// createTestConfig creates a test configuration with mock provider.
func createTestConfig(providerType string) *config.Config {
	cfg := &config.Config{
		DefaultProvider: providerType,
		Providers: map[string]config.ProviderConfig{
			providerType: {
				Type:    providerType,
				APIKey:  config.NewSecureString("test-api-key"),
				BaseURL: "http://localhost:8080",
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
			Plugins:   false,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Apply defaults manually
	cfg.Settings.ApplyDefaults()

	return cfg
}

// setupMockContext sets up a mock context for commands.
func setupMockContext(cfg *config.Config) {
	// Inject in-memory config so commands use it instead of loading from disk
	SetInjectedConfig(cfg)
	// This would normally be set by root command initialization
	// For testing, we'll create a minimal mock setup

	// Create mock provider with proper config
	mockConfig := mock.DefaultConfig()
	mockConfig.Name = "mock"
	mockConfig.SupportedModels = []string{"test-model-1", "test-model-2"}

	mockProvider := mock.New(mockConfig)

	// Set up default responses
	mockProvider.SetGlobalResponse("Hello there!")
	mockProvider.SetResponse("test-model-1", "Go is a programming language developed by Google")
	mockProvider.SetResponse("test-model-2", "Go is a programming language developed by Google")
	mockProvider.SetResponse("test-model", "Go is a programming language developed by Google")
	// Ensure streaming test outputs a known prefix
	mockProvider.SetStreamChunks("mock-gpt-3.5-turbo", []string{"Once upon a time ", "there was a ", "blazing fast Go CLI named GOLLM..."})

	// Inject provider factory so commands use our configured mock
	SetInjectedProviderFactory(func(_ string, _ core.ProviderConfig) (core.Provider, error) {
		return mockProvider, nil
	})

	// Set up mock context (this would normally be done by CLI root)
	// For these tests, we're focusing on command logic rather than full integration
}

// Test edge cases and error conditions

func TestChatCommand_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupConfig func() *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "no providers configured",
			args: []string{"Hello"},
			setupConfig: func() *config.Config {
				return &config.Config{
					Providers: map[string]config.ProviderConfig{},
				}
			},
			expectError: true,
			errorMsg:    "no providers configured",
		},
		{
			name: "invalid temperature",
			args: []string{"--temperature", "5.0", "Hello"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: true,
			errorMsg:    "temperature must be between 0 and 2",
		},
		{
			name: "invalid max tokens",
			args: []string{"--max-tokens", "-100", "Hello"},
			setupConfig: func() *config.Config {
				return createTestConfig("mock")
			},
			expectError: true,
			errorMsg:    "max tokens must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewTestExecutor()
			cfg := tt.setupConfig()

			cmd := NewChatCommand()
			require.NotNil(t, cmd)

			setupMockContext(cfg)

			err := executor.Execute(cmd, tt.args, "")

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Benchmark tests for performance validation

func BenchmarkChatCommand_Execute(b *testing.B) {
	executor := NewTestExecutor()
	cfg := createTestConfig("mock")
	cmd := NewChatCommand()
	setupMockContext(cfg)

	args := []string{"Hello, world!"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := executor.Execute(cmd, args, "")
		if err != nil {
			b.Fatal(err)
		}

		// Reset for next iteration
		executor.stdout.Reset()
		executor.stderr.Reset()
	}
}

func BenchmarkModelsCommand_List(b *testing.B) {
	executor := NewTestExecutor()
	cfg := createTestConfig("mock")
	cmd := NewModelsCommand()
	setupMockContext(cfg)

	args := []string{"list"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := executor.Execute(cmd, args, "")
		if err != nil {
			b.Fatal(err)
		}

		executor.stdout.Reset()
		executor.stderr.Reset()
	}
}

// Integration test with real-like scenarios

func TestCommandIntegration_RealWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test simulates a real user workflow:
	// 1. Initialize config
	// 2. List models
	// 3. Send chat message
	// 4. Validate configuration

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "workflow-config.yaml")

	// Step 1: Initialize config
	t.Run("init_config", func(t *testing.T) {
		executor := NewTestExecutor()
		cfg := createTestConfig("mock")

		cmd := NewConfigCommand()
		setupMockContext(cfg)

		err := executor.Execute(cmd, []string{"init", "--config-file", configFile}, "")
		assert.NoError(t, err)

		stdout, _ := executor.GetOutput()
		assert.Contains(t, stdout, "Configuration initialized")
	})

	// Step 2: List models
	t.Run("list_models", func(t *testing.T) {
		executor := NewTestExecutor()
		cfg := createTestConfig("mock")

		cmd := NewModelsCommand()
		setupMockContext(cfg)

		err := executor.Execute(cmd, []string{"list"}, "")
		assert.NoError(t, err)

		stdout, _ := executor.GetOutput()
		assert.Contains(t, stdout, "test-model")
	})

	// Step 3: Send chat message
	t.Run("send_chat", func(t *testing.T) {
		executor := NewTestExecutor()
		cfg := createTestConfig("mock")

		cmd := NewChatCommand()
		setupMockContext(cfg)

		err := executor.Execute(cmd, []string{"What is the capital of France?"}, "")
		assert.NoError(t, err)

		stdout, _ := executor.GetOutput()
		assert.NotEmpty(t, stdout)
	})

	// Step 4: Validate config
	t.Run("validate_config", func(t *testing.T) {
		executor := NewTestExecutor()
		cfg := createTestConfig("mock")

		cmd := NewConfigCommand()
		setupMockContext(cfg)

		err := executor.Execute(cmd, []string{"validate"}, "")
		assert.NoError(t, err)

		stdout, _ := executor.GetOutput()
		assert.Contains(t, stdout, "valid")
	})
}

// Timeout and cancellation tests

func TestCommand_Timeout(t *testing.T) {
	executor := NewTestExecutor()
	cfg := createTestConfig("mock")

	cmd := NewChatCommand()
	setupMockContext(cfg)

	// Set a very short timeout to force timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	cmd.SetContext(ctx)

	err := executor.Execute(cmd, []string{"Hello"}, "")

	// Should get context deadline exceeded or similar
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

// Output format tests

func TestChatCommand_OutputFormats(t *testing.T) {
	formats := []struct {
		format    string
		expectOut string
	}{
		{"text", "Hello there!"},
		{"json", `"content"`},
		{"yaml", "content:"},
	}

	for _, tt := range formats {
		t.Run(fmt.Sprintf("format_%s", tt.format), func(t *testing.T) {
			executor := NewTestExecutor()
			cfg := createTestConfig("mock")

			cmd := NewChatCommand()
			setupMockContext(cfg)

			args := []string{"--output", tt.format, "Hello"}
			err := executor.Execute(cmd, args, "")
			assert.NoError(t, err)

			stdout, _ := executor.GetOutput()
			assert.Contains(t, stdout, tt.expectOut)
		})
	}
}
