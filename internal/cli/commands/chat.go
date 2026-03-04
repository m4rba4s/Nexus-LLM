// Package commands provides CLI command implementations for GOLLM.
//
// This package contains the implementation of various CLI commands including
// chat, interactive, complete, models, and configuration management commands.
//
// Each command follows a consistent pattern:
//   - Validation of inputs and flags
//   - Provider initialization and selection
//   - Request construction with proper error handling
//   - Response processing with streaming support
//   - Formatted output with color and formatting options
package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"

	// Import providers to trigger their init() registration
	_ "github.com/yourusername/gollm/internal/providers/anthropic"
	_ "github.com/yourusername/gollm/internal/providers/deepseek"
	_ "github.com/yourusername/gollm/internal/providers/gemini"
	_ "github.com/yourusername/gollm/internal/providers/mock"
	_ "github.com/yourusername/gollm/internal/providers/openai"
	_ "github.com/yourusername/gollm/internal/providers/openrouter"
)

// ChatFlags contains flags specific to the chat command.
type ChatFlags struct {
	Model            string
	Provider         string
	Temperature      float64
	MaxTokens        int
	TopP             float64
	FrequencyPenalty float64
	PresencePenalty  float64
	SystemMessage    string
	Stream           bool
	NoStream         bool
	OutputFile       string
	Format           string
	Interactive      bool
	Quiet            bool
	Raw              bool
	Stop             []string
}

// CommandContext contains the shared context needed by commands.
type CommandContext struct {
	Config  *config.Config
	Timeout time.Duration
}

// NewChatCommand creates the chat command.
func NewChatCommand() *cobra.Command {
	flags := &ChatFlags{}

	cmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Send a chat message to an LLM",
		Long: `Send a chat message to a Large Language Model and receive a response.

The message can be provided as an argument or piped from stdin.
Supports streaming responses and various output formats.

Examples:
  # Simple chat message
  gollm chat "What is Go programming language?"

  # Chat with specific model and provider
  gollm chat "Explain quantum computing" --provider openai --model gpt-4

  # Chat with custom parameters
  gollm chat "Write a haiku" --temperature 1.2 --max-tokens 100

  # Chat with system message
  gollm chat "Hello" --system "You are a helpful assistant"

  # Streaming response (default)
  gollm chat "Tell me a story" --stream

  # Non-streaming response
  gollm chat "What is 2+2?" --no-stream

  # Read from stdin
  echo "Explain this code" | gollm chat

  # Save response to file
  gollm chat "Write documentation" --output response.md

  # Raw output without formatting
  gollm chat "Hello" --raw`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatCommand(cmd, args, flags)
		},
	}

	// Add command-specific flags
	addChatFlags(cmd, flags)

	return cmd
}

// addChatFlags adds flags specific to the chat command.
func addChatFlags(cmd *cobra.Command, flags *ChatFlags) {
	f := cmd.Flags()

	// Model and provider selection
	f.StringVar(&flags.Model, "model", "",
		"model to use for the chat (overrides config default)")
	f.StringVar(&flags.Provider, "provider", "",
		"provider to use for the chat (overrides config default)")

	// Generation parameters
	f.Float64Var(&flags.Temperature, "temperature", 0,
		"sampling temperature 0.0-2.0 (0=use default)")
	f.IntVar(&flags.MaxTokens, "max-tokens", 0,
		"maximum tokens in response (0=use default)")
	f.Float64Var(&flags.TopP, "top-p", 0,
		"nucleus sampling parameter 0.0-1.0 (0=use default)")
	f.Float64Var(&flags.FrequencyPenalty, "frequency-penalty", 0,
		"frequency penalty -2.0 to 2.0 (0=use default)")
	f.Float64Var(&flags.PresencePenalty, "presence-penalty", 0,
		"presence penalty -2.0 to 2.0 (0=use default)")
	f.StringSliceVar(&flags.Stop, "stop", nil,
		"stop sequences to end generation")

	// Message configuration
	f.StringVar(&flags.SystemMessage, "system", "",
		"system message to set context")

	// Output and behavior flags
	f.BoolVar(&flags.Stream, "stream", false,
		"stream response tokens")
	f.BoolVar(&flags.NoStream, "no-stream", false,
		"disable streaming and wait for complete response")
	f.StringVar(&flags.OutputFile, "output-file", "",
		"save response to file")
	f.StringVar(&flags.Format, "output", "",
		"output format (text, json, yaml)")
	f.BoolVar(&flags.Interactive, "interactive", false,
		"enter interactive mode after response")
	f.BoolVar(&flags.Quiet, "quiet", false,
		"only output the response content")
	f.BoolVar(&flags.Raw, "raw", false,
		"output raw response without formatting")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("stream", "no-stream")

	// Add completion for provider and model flags
	cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	cmd.RegisterFlagCompletionFunc("model", completeModels)
}

// runChatCommand executes the chat command.
func runChatCommand(cmd *cobra.Command, args []string, flags *ChatFlags) error {
	ctx, cancel := setupContext(cmd)
	defer cancel()

	// Get message from args or stdin
	message, err := getChatMessage(args)
	if err != nil {
		return fmt.Errorf("failed to get chat message: %w", err)
	}

	if message == "" {
		return fmt.Errorf("message is required")
	}

	// Validate flags
	if err := validateChatFlagsRuntime(flags); err != nil {
		return err
	}

	// Load configuration (or injected config during tests)
	cfg, err := getInjectedOrLoad()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve provider and model
	providerName, model, err := resolveProviderAndModel(flags, cfg)
	if err != nil {
		return err
	}

	// Validate provider exists
	if err := validateProvider(providerName, cfg); err != nil {
		return err
	}

	// Create provider instance
	provider, err := createProvider(providerName, cfg)
	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Build request
	request, err := buildChatRequest(message, model, flags)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	// Determine output format from global flag or config
	outputFormat := flags.Format
	if outputFormat == "" {
		outputFormat = getOutputFormat(cmd)
	}
	if outputFormat == "" && cfg != nil {
		outputFormat = cfg.Settings.OutputFormat
	}
	if outputFormat == "" {
		outputFormat = "text"
	}

	// Determine if we should stream (only for text output)
	shouldStream := flags.Stream && !flags.NoStream && outputFormat == "text"

	// Resolve writers from Cobra (respect tests and redirection)
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	// Execute request
	if shouldStream {
		return executeStreamingChat(ctx, provider, request, flags, out, errOut)
	} else {
		return executeNonStreamingChat(ctx, provider, request, flags, out, errOut, outputFormat)
	}
}

// getChatMessage extracts the chat message from args or stdin.
func getChatMessage(args []string) (string, error) {
	// If message provided as argument
	if len(args) > 0 {
		return strings.TrimSpace(args[0]), nil
	}

	// Check if stdin has data
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}

	// If data is piped to stdin
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		var lines []string

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					if line != "" {
						lines = append(lines, line)
					}
					break
				}
				return "", fmt.Errorf("failed to read from stdin: %w", err)
			}
			lines = append(lines, line)
		}

		return strings.TrimSpace(strings.Join(lines, "")), nil
	}

	// No message provided
	return "", nil
}

// resolveProviderAndModel determines which provider and model to use.
func resolveProviderAndModel(flags *ChatFlags, cfg *config.Config) (string, string, error) {
	var providerName string
	var model string

	// Use provider from flag or config default
	if flags.Provider != "" {
		providerName = flags.Provider
	} else {
		var err error
		providerName, _, err = cfg.GetDefaultProvider()
		if err != nil {
			return "", "", fmt.Errorf("no provider specified and no default configured: %w", err)
		}
	}

	// Validate provider exists
	if !cfg.HasProvider(providerName) {
		return "", "", fmt.Errorf("provider not configured; available providers: %s",
			strings.Join(cfg.ListProviders(), ", "))
	}

	// Use model from flag, provider default, or config default
	if flags.Model != "" {
		model = flags.Model
	} else {
		// Get provider config to check for default model
		providerConfig, _, err := cfg.GetProvider(providerName)
		if err != nil {
			return "", "", fmt.Errorf("failed to get provider config: %w", err)
		}
		if providerConfig.DefaultModel != "" {
			model = providerConfig.DefaultModel
		} else {
			// Fallback: pick a reasonable default model based on provider type/name
			switch strings.ToLower(providerConfig.Type) {
			case "mock":
				model = "mock-gpt-3.5-turbo"
			case "openai":
				model = "gpt-3.5-turbo"
			case "anthropic":
				model = "claude-3-haiku-20240307"
			default:
				model = "default"
			}
		}
	}

	return providerName, model, nil
}

// createProvider creates a provider instance based on the provider name and config.
func createProvider(providerName string, cfg *config.Config) (core.Provider, error) {
	// Validate provider exists before proceeding
	if !cfg.HasProvider(providerName) {
		available := cfg.ListProviders()
		return nil, fmt.Errorf("provider %q not configured; available providers: %s",
			providerName, strings.Join(available, ", "))
	}
	providerConfig, _, err := cfg.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("provider %q not configured", providerName)
	}

	// Convert config.ProviderConfig to core.ProviderConfig
	coreConfig := providerConfig.ToProviderConfig()

	// Use injected factory if provided (tests)
	if injectedProviderFactory != nil {
		return injectedProviderFactory(providerConfig.Type, coreConfig)
	}

	// Use the registry to create the provider
	provider, err := core.CreateProviderFromConfig(providerConfig.Type, coreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	return provider, nil
}

// buildChatRequest constructs a completion request from the message and flags.
func buildChatRequest(message, model string, flags *ChatFlags) (*core.CompletionRequest, error) {
	request := &core.CompletionRequest{
		Model:     model,
		Messages:  []core.Message{{Role: core.RoleUser, Content: message}},
		CreatedAt: time.Now(),
	}

	// Add system message if provided
	if flags.SystemMessage != "" {
		systemMsg := core.Message{Role: core.RoleSystem, Content: flags.SystemMessage}
		request.Messages = append([]core.Message{systemMsg}, request.Messages...)
	}

	// Set optional parameters from flags
	if flags.MaxTokens > 0 {
		request.MaxTokens = &flags.MaxTokens
	}
	if flags.Temperature > 0 {
		request.Temperature = &flags.Temperature
	}
	if flags.TopP > 0 {
		request.TopP = &flags.TopP
	}
	if flags.FrequencyPenalty != 0 {
		request.FrequencyPenalty = &flags.FrequencyPenalty
	}
	if flags.PresencePenalty != 0 {
		request.PresencePenalty = &flags.PresencePenalty
	}
	if len(flags.Stop) > 0 {
		request.Stop = flags.Stop
	}

	// Set streaming flag
	request.Stream = flags.Stream && !flags.NoStream

	return request, nil
}

// executeStreamingChat executes a streaming chat request.
func executeStreamingChat(ctx context.Context, provider core.Provider, request *core.CompletionRequest, flags *ChatFlags, out io.Writer, errOut io.Writer) error {
	// Check if provider supports streaming
	_, supportsStreaming := provider.(core.Streamer)
	if !supportsStreaming {
		fmt.Fprintf(errOut, "Warning: Provider %s does not support streaming, falling back to non-streaming\n", provider.Name())
		return executeNonStreamingChat(ctx, provider, request, flags, out, errOut, "text")
	}

	streamer := provider.(core.Streamer)
	chunks, err := streamer.StreamCompletion(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	var outputWriter io.Writer = out
	var outputFile *os.File

	// Set up output file if specified
	if flags.OutputFile != "" {
		outputFile, err = os.Create(flags.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()
		outputWriter = io.MultiWriter(out, outputFile)
	}

	// Process streaming chunks
	var responseContent strings.Builder
	var totalUsage *core.Usage

	for chunk := range chunks {
		if chunk.Error != nil {
			return fmt.Errorf("streaming error: %w", chunk.Error)
		}

		// Extract content from chunk
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				responseContent.WriteString(content)
				if !flags.Quiet && !flags.Raw {
					fmt.Fprint(outputWriter, content)
				}
			}
		}

		// Store usage information from final chunk
		if chunk.Usage != nil {
			totalUsage = chunk.Usage
		}

		if chunk.Done {
			break
		}
	}

	// Output final newline and statistics
	if !flags.Quiet && !flags.Raw {
		fmt.Fprintln(outputWriter)
	}

	// Output the complete response if in quiet or raw mode
	if flags.Quiet || flags.Raw {
		fmt.Fprint(outputWriter, responseContent.String())
		if !flags.Raw {
			fmt.Fprintln(outputWriter)
		}
	}

	// Show usage statistics if not quiet
	if !flags.Quiet && !flags.Raw && totalUsage != nil {
		fmt.Fprintf(errOut, "\n[Tokens: %d prompt + %d completion = %d total]\n",
			totalUsage.PromptTokens, totalUsage.CompletionTokens, totalUsage.TotalTokens)
	}

	return nil
}

// executeNonStreamingChat executes a non-streaming chat request.
func executeNonStreamingChat(ctx context.Context, provider core.Provider, request *core.CompletionRequest, flags *ChatFlags, out io.Writer, errOut io.Writer, outputFormat string) error {
	// Disable streaming in request
	request.Stream = false

	response, err := provider.CreateCompletion(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create completion: %w", err)
	}

	var outputWriter io.Writer = out
	var outputFile *os.File

	// Set up output file if specified
	if flags.OutputFile != "" {
		outputFile, err = os.Create(flags.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()
		outputWriter = io.MultiWriter(out, outputFile)
	}

	// Extract response content
	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	}

	// Output response
	if flags.Raw {
		fmt.Fprint(outputWriter, responseContent)
	} else if outputFormat == "json" {
		fmt.Fprintf(outputWriter, "{\"content\":%q}\n", responseContent)
	} else if outputFormat == "yaml" {
		fmt.Fprintf(outputWriter, "content: |\n  %s\n", strings.ReplaceAll(responseContent, "\n", "\n  "))
	} else {
		fmt.Fprintln(outputWriter, responseContent)
	}

	// Show usage statistics if not quiet
	if !flags.Quiet && !flags.Raw {
		fmt.Fprintf(errOut, "\n[Tokens: %d prompt + %d completion = %d total]\n",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
		if response.ResponseTime > 0 {
			fmt.Fprintf(errOut, "[Response time: %v]\n", response.ResponseTime)
		}
	}

	return nil
}

// getOutputFormat reads the persistent/global output format flag if set.
func getOutputFormat(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	if f := cmd.InheritedFlags().Lookup("output"); f != nil && f.Value != nil {
		val := strings.TrimSpace(f.Value.String())
		switch val {
		case "text", "json", "yaml", "markdown":
			return val
		}
	}
	return ""
}

// Test-only: allow injecting a provider factory so tests can supply a mock
var injectedProviderFactory func(providerType string, cfg core.ProviderConfig) (core.Provider, error)

func SetInjectedProviderFactory(f func(string, core.ProviderConfig) (core.Provider, error)) {
	injectedProviderFactory = f
}

// Completion functions

// completeProviders provides shell completion for provider names.
func completeProviders(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	providers := cfg.ListProviders()
	return providers, cobra.ShellCompDirectiveNoFileComp
}

// setupContext sets up command context with timeout.
func setupContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Default timeout for chat commands
	timeout := 60 * time.Second
	return context.WithTimeout(ctx, timeout)
}

// validateProvider validates that a provider name is configured.
func validateProvider(providerName string, cfg *config.Config) error {
	if !cfg.HasProvider(providerName) {
		available := cfg.ListProviders()
		return fmt.Errorf("provider %q not configured; available providers: %s",
			providerName, strings.Join(available, ", "))
	}
	return nil
}

// validateChatFlagsRuntime validates chat flags for common errors at runtime.
func validateChatFlagsRuntime(flags *ChatFlags) error {
	if flags.Temperature < 0 || flags.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if flags.MaxTokens < 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	return nil
}

// completeModels provides shell completion for model names.
func completeModels(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// This would need to be implemented based on the selected provider
	// For now, return common model names
	models := []string{
		"gpt-3.5-turbo",
		"gpt-4",
		"gpt-4-turbo",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
		"claude-3-haiku-20240307",
	}
	return models, cobra.ShellCompDirectiveNoFileComp
}
