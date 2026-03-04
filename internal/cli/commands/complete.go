// Package commands provides code completion functionality for GOLLM.
package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/display"

	// Import providers to trigger their init() registration
	_ "github.com/yourusername/gollm/internal/providers/anthropic"
	_ "github.com/yourusername/gollm/internal/providers/deepseek"
	_ "github.com/yourusername/gollm/internal/providers/gemini"
	_ "github.com/yourusername/gollm/internal/providers/mock"
	_ "github.com/yourusername/gollm/internal/providers/openai"
	_ "github.com/yourusername/gollm/internal/providers/openrouter"
)

// Writer for completion output; set per command execution
var compOut io.Writer = os.Stdout

// CompleteFlags contains flags specific to the complete command.
type CompleteFlags struct {
	Model            string
	Provider         string
	Temperature      float64
	MaxTokens        int
	TopP             float64
	FrequencyPenalty float64
	PresencePenalty  float64
	Language         string
	Context          string
	Style            string
	InputFile        string
	OutputFile       string
	Stream           bool
	NoStream         bool
	ShowContext      bool
	ShowExplanation  bool
	MultipleOptions  int
	Quiet            bool
	Raw              bool
	Stop             []string
}

// CompletionStyle represents different styles of code completion.
type CompletionStyle struct {
	Name        string
	Description string
	SystemMsg   string
}

// predefined completion styles
var completionStyles = map[string]CompletionStyle{
	"function": {
		Name:        "function",
		Description: "Complete function implementations",
		SystemMsg:   "You are an expert programmer. Complete the given function implementation. Provide clean, efficient, and well-documented code. Only return the code implementation without explanations unless requested.",
	},
	"class": {
		Name:        "class",
		Description: "Complete class definitions and methods",
		SystemMsg:   "You are an expert programmer. Complete the given class definition with proper methods and documentation. Follow best practices and design patterns. Only return the code implementation without explanations unless requested.",
	},
	"comment": {
		Name:        "comment",
		Description: "Generate documentation and comments",
		SystemMsg:   "You are a technical writer and expert programmer. Generate comprehensive documentation, comments, and docstrings for the given code. Explain the purpose, parameters, return values, and any important implementation details.",
	},
	"fix": {
		Name:        "fix",
		Description: "Fix and improve existing code",
		SystemMsg:   "You are an expert code reviewer and programmer. Analyze the given code for bugs, issues, and improvements. Fix any problems and optimize the code while maintaining its original functionality. Explain what was fixed if improvements are made.",
	},
	"test": {
		Name:        "test",
		Description: "Generate unit tests for code",
		SystemMsg:   "You are an expert in software testing. Generate comprehensive unit tests for the given code. Include edge cases, error conditions, and positive/negative test scenarios. Follow testing best practices for the specific language and framework.",
	},
	"refactor": {
		Name:        "refactor",
		Description: "Refactor and optimize code",
		SystemMsg:   "You are an expert programmer specializing in code refactoring. Improve the given code by making it more readable, efficient, and maintainable. Apply design patterns, remove code smells, and optimize performance while preserving functionality.",
	},
	"explain": {
		Name:        "explain",
		Description: "Explain how code works",
		SystemMsg:   "You are a programming teacher and expert developer. Explain how the given code works in detail. Break down complex concepts, describe the logic flow, and help the reader understand the implementation. Use clear, educational language.",
	},
}

// NewCompleteCommand creates the complete command.
func NewCompleteCommand() *cobra.Command {
	flags := &CompleteFlags{}

	cmd := &cobra.Command{
		Use:   "complete [code-prompt]",
		Short: "Generate code completions and implementations",
		Long: `Generate code completions, implementations, and improvements using LLMs.

The complete command is designed for various code-related tasks including:
• Function and class implementation
• Code documentation and comments
• Bug fixes and optimizations
• Unit test generation
• Code refactoring
• Code explanation and analysis

Examples:
  # Complete a function implementation
  gollm complete "def fibonacci(n):" --language python --style function

  # Complete code from file
  gollm complete --input-file incomplete.py --style function

  # Generate tests for existing code
  gollm complete --input-file mycode.py --style test --output-file test_mycode.py

  # Fix and improve code
  gollm complete "buggy code here" --style fix --show-explanation

  # Generate documentation
  gollm complete --input-file mycode.py --style comment

  # Multiple completion options
  gollm complete "def sort_array(arr):" --multiple 3 --language python

  # Pipe code from stdin
  cat mycode.py | gollm complete --style refactor

  # Complete with specific context
  gollm complete "handle_request" --context "Flask web application" --language python`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			compOut = cmd.OutOrStdout()
			return runCompleteCommand(cmd, args, flags)
		},
	}

	// Add command-specific flags
	addCompleteFlags(cmd, flags)

	return cmd
}

// addCompleteFlags adds flags specific to the complete command.
func addCompleteFlags(cmd *cobra.Command, flags *CompleteFlags) {
	f := cmd.Flags()

	// Model and provider selection
	f.StringVar(&flags.Model, "model", "",
		"model to use for completion (overrides config default)")
	f.StringVar(&flags.Provider, "provider", "",
		"provider to use for completion (overrides config default)")

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

	// Code completion specific
	f.StringVar(&flags.Language, "language", "",
		"programming language (auto-detected from file extension if not specified)")
	f.StringVar(&flags.Context, "context", "",
		"additional context about the code or project")
	f.StringVar(&flags.Style, "style", "function",
		fmt.Sprintf("completion style (%s)", getStyleList()))
	f.IntVar(&flags.MultipleOptions, "multiple", 1,
		"generate multiple completion options (1-5)")

	// Input/Output
	f.StringVar(&flags.InputFile, "input-file", "",
		"read code from file instead of argument")
	f.StringVar(&flags.OutputFile, "output-file", "",
		"save completion to file")

	// Output behavior
	f.BoolVar(&flags.Stream, "stream", false,
		"stream response tokens")
	f.BoolVar(&flags.NoStream, "no-stream", false,
		"disable streaming (default for code completion)")
	f.BoolVar(&flags.ShowContext, "show-context", false,
		"show the original code context in output")
	f.BoolVar(&flags.ShowExplanation, "show-explanation", false,
		"include explanation of the completion")
	f.BoolVar(&flags.Quiet, "quiet", false,
		"only output the completion code")
	f.BoolVar(&flags.Raw, "raw", false,
		"output raw response without formatting")

	// Validation handled above with other flags

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("stream", "no-stream")

	// Add completion for flags
	cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	cmd.RegisterFlagCompletionFunc("model", completeModels)
	cmd.RegisterFlagCompletionFunc("language", completeLanguages)
	cmd.RegisterFlagCompletionFunc("style", completeStyles)
}

// runCompleteCommand executes the complete command.
func runCompleteCommand(cmd *cobra.Command, args []string, flags *CompleteFlags) error {
	ctx, cancel := setupContext(cmd)
	defer cancel()

	// Get code input from args, file, or stdin (cobra input)
	codeInput, language, err := getCodeInputFrom(cmd, args, flags)
	if err != nil {
		return fmt.Errorf("failed to get code input: %w", err)
	}

	if strings.TrimSpace(codeInput) == "" {
		return fmt.Errorf("no code input provided (use argument, --input-file, or pipe from stdin)")
	}

	// Validate flags
	if err := validateCompleteFlags(flags); err != nil {
		return err
	}

	// Load configuration (or injected during tests)
	cfg, err := getInjectedOrLoad()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve provider and model
	providerName, model, err := resolveProviderAndModel(&ChatFlags{
		Provider: flags.Provider,
		Model:    flags.Model,
	}, cfg)
	if err != nil {
		return err
	}

	// Create provider instance
	provider, err := createProvider(providerName, cfg)
	if err != nil {
		return fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	// Generate completions
	if flags.MultipleOptions > 1 {
		return generateMultipleCompletions(ctx, provider, codeInput, language, model, flags)
	}

	return generateSingleCompletion(ctx, provider, codeInput, language, model, flags)
}

// getCodeInput extracts code input from various sources.
func getCodeInput(args []string, flags *CompleteFlags) (string, string, error) {
	var codeInput string
	var language string

	// Priority: input file > args > stdin
	if flags.InputFile != "" {
		// Read from file
		data, err := os.ReadFile(flags.InputFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read input file: %w", err)
		}
		codeInput = string(data)

		// Auto-detect language from file extension
		if flags.Language == "" {
			language = detectLanguageFromExtension(flags.InputFile)
		} else {
			language = flags.Language
		}
	} else if len(args) > 0 {
		// From command line argument
		codeInput = args[0]
		language = flags.Language
	} else {
		// Check stdin
		stat, err := os.Stdin.Stat()
		if err != nil {
			return "", "", fmt.Errorf("failed to stat stdin: %w", err)
		}

		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Read from stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", "", fmt.Errorf("failed to read from stdin: %w", err)
			}
			codeInput = string(data)
			language = flags.Language
		}
	}

	return codeInput, language, nil
}

// getCodeInputFrom reads from cmd.InOrStdin for testability
func getCodeInputFrom(cmd *cobra.Command, args []string, flags *CompleteFlags) (string, string, error) {
	if flags.InputFile != "" {
		data, err := os.ReadFile(flags.InputFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to read input file: %w", err)
		}
		if flags.Language == "" {
			return string(data), detectLanguageFromExtension(flags.InputFile), nil
		}
		return string(data), flags.Language, nil
	}

	if len(args) > 0 {
		return args[0], flags.Language, nil
	}

	in := cmd.InOrStdin()
	// peek if char device
	if f, ok := in.(*os.File); ok {
		if fi, err := f.Stat(); err == nil {
			if (fi.Mode() & os.ModeCharDevice) != 0 {
				return "", "", nil
			}
		}
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", "", fmt.Errorf("failed to read from stdin: %w", err)
	}
	return string(data), flags.Language, nil
}

// validateCompleteFlags validates completion flags.
func validateCompleteFlags(flags *CompleteFlags) error {
	// Treat 0 as default (1)
	if flags.MultipleOptions == 0 {
		flags.MultipleOptions = 1
	}
	if flags.MultipleOptions < 1 || flags.MultipleOptions > 5 {
		return fmt.Errorf("multiple options must be between 1 and 5")
	}

	if flags.Temperature < 0 || flags.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if flags.MaxTokens < 0 {
		return fmt.Errorf("max tokens must be positive")
	}

	if flags.Style != "" {
		if _, exists := completionStyles[flags.Style]; !exists {
			return fmt.Errorf("invalid style %q; available styles: %s", flags.Style, getStyleList())
		}
	}

	return nil
}

// generateSingleCompletion generates a single code completion.
func generateSingleCompletion(ctx context.Context, provider core.Provider, codeInput, language, model string, flags *CompleteFlags) error {
	// Build the prompt
	prompt := buildCompletionPrompt(codeInput, language, flags)

	// Build request
	request, err := buildCompleteRequest(prompt, model, flags)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	// Determine streaming behavior (default to non-streaming for code)
	shouldStream := flags.Stream && !flags.NoStream

	var response string
	var usage *core.Usage

	if shouldStream {
		response, usage, err = executeStreamingComplete(ctx, provider, request, flags)
	} else {
		response, usage, err = executeNonStreamingComplete(ctx, provider, request, flags)
	}

	if err != nil {
		return err
	}

	// Process and output response
	return outputCompletion(response, usage, codeInput, flags)
}

// generateMultipleCompletions generates multiple completion options.
func generateMultipleCompletions(ctx context.Context, provider core.Provider, codeInput, language, model string, flags *CompleteFlags) error {
	if !flags.Quiet {
		fmt.Printf("Generating %d completion options...\n\n", flags.MultipleOptions)
	}

	for i := 0; i < flags.MultipleOptions; i++ {
		if !flags.Quiet {
			fmt.Printf("Option %d:\n", i+1)
			fmt.Printf("─────────\n")
		}

		// Generate completion
		err := generateSingleCompletion(ctx, provider, codeInput, language, model, flags)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate option %d: %v\n", i+1, err)
			continue
		}

		if i < flags.MultipleOptions-1 && !flags.Quiet {
			fmt.Print("\n" + strings.Repeat("═", 50) + "\n\n")
		}
	}

	return nil
}

// buildCompletionPrompt constructs the completion prompt.
func buildCompletionPrompt(codeInput, language string, flags *CompleteFlags) string {
	var parts []string

	// Add context if provided
	if flags.Context != "" {
		parts = append(parts, fmt.Sprintf("Context: %s", flags.Context))
	}

	// Add language information
	if language != "" {
		parts = append(parts, fmt.Sprintf("Language: %s", language))
	}

	// Add the code input
	if flags.Style == "explain" || flags.Style == "comment" {
		parts = append(parts, fmt.Sprintf("Code to %s:", flags.Style))
	} else {
		parts = append(parts, "Code to complete:")
	}

	parts = append(parts, codeInput)

	return strings.Join(parts, "\n\n")
}

// buildCompleteRequest constructs a completion request.
func buildCompleteRequest(prompt, model string, flags *CompleteFlags) (*core.CompletionRequest, error) {
	// Get system message for the style
	var systemMessage string
	if style, exists := completionStyles[flags.Style]; exists {
		systemMessage = style.SystemMsg

		// Add explanation request to system message if requested
		if flags.ShowExplanation {
			systemMessage += " Include a brief explanation of your completion."
		}
	}

	messages := []core.Message{}

	// Add system message
	if systemMessage != "" {
		messages = append(messages, core.Message{
			Role:    core.RoleSystem,
			Content: systemMessage,
		})
	}

	// Add user prompt
	messages = append(messages, core.Message{
		Role:    core.RoleUser,
		Content: prompt,
	})

	request := &core.CompletionRequest{
		Model:     model,
		Messages:  messages,
		CreatedAt: time.Now(),
	}

	// Set optional parameters
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

	request.Stream = flags.Stream && !flags.NoStream

	return request, nil
}

// executeStreamingComplete handles streaming completion.
func executeStreamingComplete(ctx context.Context, provider core.Provider, request *core.CompletionRequest, flags *CompleteFlags) (string, *core.Usage, error) {
	request.Stream = true

	streamer, ok := provider.(core.Streamer)
	if !ok {
		// Fall back to non-streaming
		return executeNonStreamingComplete(ctx, provider, request, flags)
	}

	chunks, err := streamer.StreamCompletion(ctx, request)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start streaming: %w", err)
	}

	var responseBuilder strings.Builder
	var finalUsage *core.Usage

	// Create smart highlighter for streaming output
	highlighter := display.NewSyntaxHighlighter()
	highlighter.SetColorEnabled(display.HasColorSupport() && !flags.Raw)

	// Buffer for intelligent highlighting during streaming
	var streamBuffer strings.Builder
	const bufferFlushThreshold = 50 // Apply highlighting every N characters

	for chunk := range chunks {
		if chunk.Error != nil {
			return "", nil, fmt.Errorf("streaming error: %w", chunk.Error)
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				responseBuilder.WriteString(content)

				if !flags.Quiet && !flags.Raw {
					// Smart streaming output with syntax highlighting
					streamBuffer.WriteString(content)

					// Apply highlighting when we have enough content or at line breaks
					if streamBuffer.Len() > bufferFlushThreshold || strings.Contains(content, "\n") {
						bufferedContent := streamBuffer.String()

						// Try to apply syntax highlighting to accumulated content
						if highlighter.IsCodeBlock(bufferedContent) {
							highlighted, err := highlighter.FormatResponse(bufferedContent)
							if err == nil {
								// Clear previous output and rewrite with highlighting
								fmt.Print("\r" + strings.Repeat(" ", len(bufferedContent)) + "\r")
								fmt.Print(highlighted)
							} else {
								fmt.Print(content) // Fallback to raw content
							}
						} else {
							fmt.Print(content) // Regular text, no highlighting needed
						}

						streamBuffer.Reset()
					} else {
						// Just print raw content for now, will be enhanced when buffer flushes
						fmt.Print(content)
					}
				}
			}
		}

		if chunk.Usage != nil {
			finalUsage = chunk.Usage
		}

		if chunk.Done {
			// Flush any remaining buffered content with final highlighting
			if streamBuffer.Len() > 0 && !flags.Quiet && !flags.Raw {
				remaining := streamBuffer.String()
				highlighted, err := highlighter.FormatResponse(remaining)
				if err == nil {
					fmt.Print("\r" + strings.Repeat(" ", len(remaining)) + "\r")
					fmt.Print(highlighted)
				}
			}
			break
		}
	}

	return responseBuilder.String(), finalUsage, nil
}

// executeNonStreamingComplete handles non-streaming completion.
func executeNonStreamingComplete(ctx context.Context, provider core.Provider, request *core.CompletionRequest, flags *CompleteFlags) (string, *core.Usage, error) {
	request.Stream = false

	response, err := provider.CreateCompletion(ctx, request)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create completion: %w", err)
	}

	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	}

	return responseContent, &response.Usage, nil
}

// outputCompletion processes and outputs the completion result with smart syntax highlighting.
func outputCompletion(response string, usage *core.Usage, originalCode string, flags *CompleteFlags) error {
	var outputWriter io.Writer = compOut
	var outputFile *os.File
	var err error

	// Set up output file if specified
	if flags.OutputFile != "" {
		outputFile, err = os.Create(flags.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outputFile.Close()
		outputWriter = outputFile
	}

	// Create smart formatter with appropriate settings
	formatterConfig := display.DefaultFormatterConfig()

	// Configure based on flags
	if flags.Raw {
		formatterConfig.Format = display.FormatRaw
		formatterConfig.ColorEnabled = false
	} else {
		formatterConfig.Format = display.FormatAuto
		formatterConfig.ColorEnabled = display.HasColorSupport() && outputFile == nil
	}

	if flags.Quiet {
		formatterConfig.Mode = display.ModeQuiet
		formatterConfig.ShowTokens = false
		formatterConfig.ShowTiming = false
	} else {
		formatterConfig.Mode = display.ModeCompact
		formatterConfig.ShowTokens = true
		formatterConfig.ShowTiming = false
	}

	highlighter := display.NewSyntaxHighlighter()

	// Show original context if requested with smart highlighting
	if flags.ShowContext && !flags.Quiet && !flags.Raw {
		separator := "─── Original Code ───────────────────────────────────────────"
		if formatterConfig.ColorEnabled {
			separator = "\033[2m" + separator + "\033[0m"
		}
		fmt.Fprintln(outputWriter, separator)

		// Apply syntax highlighting to original code
		if flags.Language != "" || originalCode != "" {
			language := flags.Language
			if language == "" {
				language = highlighter.DetectLanguage(originalCode)
			}

			highlightedOriginal, err := highlighter.HighlightCode(originalCode, language)
			if err != nil {
				fmt.Fprintln(outputWriter, originalCode)
			} else {
				fmt.Fprintln(outputWriter, highlightedOriginal)
			}
		} else {
			fmt.Fprintln(outputWriter, originalCode)
		}

		fmt.Fprintln(outputWriter)
		separator2 := "─── AI Completion ──────────────────────────────────────────"
		if formatterConfig.ColorEnabled {
			separator2 = "\033[2m" + separator2 + "\033[0m"
		}
		fmt.Fprintln(outputWriter, separator2)
	}

	// Process and output the completion with smart formatting
	if flags.Raw {
		fmt.Fprint(outputWriter, response)
	} else {
		// Use intelligent response formatting with syntax highlighting
		formattedResponse, err := highlighter.FormatResponse(response)
		if err != nil {
			// Fallback to original response if formatting fails
			fmt.Fprintln(outputWriter, response)
		} else {
			fmt.Fprintln(outputWriter, formattedResponse)
		}
	}

	// Show enhanced usage statistics with colors
	if !flags.Quiet && !flags.Raw && usage != nil {
		statsLine := fmt.Sprintf("💾 Tokens: %d prompt + %d completion = %d total",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)

		if usage.TotalCost != nil && *usage.TotalCost > 0 {
			statsLine += fmt.Sprintf(" • 💰 Cost: $%.6f", *usage.TotalCost)
		}

		if formatterConfig.ColorEnabled {
			statsLine = "\033[2m" + statsLine + "\033[0m"
		}
		fmt.Fprintf(os.Stderr, "\n%s\n", statsLine)
	}

	// Save to file message with emoji
	if flags.OutputFile != "" && !flags.Quiet {
		successMsg := fmt.Sprintf("✅ Completion saved to: %s", flags.OutputFile)
		if formatterConfig.ColorEnabled {
			successMsg = "\033[32m" + successMsg + "\033[0m"
		}
		fmt.Fprintf(os.Stderr, "%s\n", successMsg)
	}

	return nil
}

// Helper functions

// detectLanguageFromExtension detects programming language from file extension.
func detectLanguageFromExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	languageMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".java":  "java",
		".cpp":   "cpp",
		".cc":    "cpp",
		".cxx":   "cpp",
		".c":     "c",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".rs":    "rust",
		".scala": "scala",
		".sh":    "bash",
		".bash":  "bash",
		".zsh":   "zsh",
		".ps1":   "powershell",
		".sql":   "sql",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".sass":  "sass",
		".less":  "less",
		".xml":   "xml",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".md":    "markdown",
		".tex":   "latex",
		".r":     "r",
		".m":     "matlab",
		".pl":    "perl",
		".lua":   "lua",
		".vim":   "vim",
		".fish":  "fish",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return ""
}

// getStyleList returns a comma-separated list of available styles.
func getStyleList() string {
	var styles []string
	for name := range completionStyles {
		styles = append(styles, name)
	}
	return strings.Join(styles, ", ")
}

// Completion functions for CLI

// completeLanguages provides shell completion for programming languages.
func completeLanguages(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	languages := []string{
		"go", "python", "javascript", "typescript", "java", "cpp", "c", "csharp",
		"ruby", "php", "swift", "kotlin", "rust", "scala", "bash", "powershell",
		"sql", "html", "css", "json", "yaml", "markdown", "r", "matlab", "perl",
		"lua", "vim", "fish",
	}
	return languages, cobra.ShellCompDirectiveNoFileComp
}

// completeStyles provides shell completion for completion styles.
func completeStyles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var styles []string
	for name, style := range completionStyles {
		styles = append(styles, fmt.Sprintf("%s\t%s", name, style.Description))
	}
	return styles, cobra.ShellCompDirectiveNoFileComp
}
