// Package cli provides the command-line interface for GOLLM.
//
// The CLI is built using the Cobra library and provides a comprehensive
// set of commands for interacting with Large Language Models through
// various providers.
//
// The CLI supports:
//   - Multiple LLM providers (OpenAI, Anthropic, Ollama, etc.)
//   - Interactive and batch processing modes
//   - Streaming responses
//   - Configuration management
//   - Plugin system integration
//   - Model Context Protocol (MCP) support
//
// Example usage:
//
//	cmd, err := cli.NewRootCommand(cli.BuildInfo{
//		Version:   "1.0.0",
//		Commit:    "abc123",
//		BuildTime: "2024-01-01T00:00:00Z",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	err = cmd.Execute()
//	if err != nil {
//		os.Exit(1)
//	}
package cli

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/spf13/cobra"

    "github.com/m4rba4s/Nexus-LLM/internal/cli/commands"
    "github.com/m4rba4s/Nexus-LLM/internal/config"
    "github.com/m4rba4s/Nexus-LLM/internal/core"
    "github.com/m4rba4s/Nexus-LLM/internal/version"
    branding "github.com/m4rba4s/Nexus-LLM/internal/branding"
)

// BuildInfo contains information about the build.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildTime string
}

// GlobalFlags contains global CLI flags that are available to all commands.
type GlobalFlags struct {
	ConfigFile   string
	LogLevel     string
	OutputFormat string
	NoColor      bool
	Quiet        bool
	Verbose      bool
	Provider     string
	Model        string
	Temperature  float64
	MaxTokens    int
	Timeout      time.Duration
	Debug        bool
}

// RootContext contains shared context for all CLI commands.
type RootContext struct {
	Config      *config.Config
	BuildInfo   BuildInfo
	GlobalFlags *GlobalFlags
	Registry    *core.ProviderRegistry
}

var (
	// Global context shared across all commands
	rootContext *RootContext

	// Global flags instance
	globalFlags = &GlobalFlags{}
)

// NewRootCommand creates and returns the root cobra command.
func NewRootCommand(buildInfo BuildInfo) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   "gollm",
		Short: "High-performance CLI for Large Language Models",
        Long: branding.GetLogo(branding.LogoOptions{
            ShowTagline: true,
            Colored:     true,
            Compact:     false,
        }) + `

Built with performance and reliability in mind, GOLLM provides:
• Lightning-fast startup (sub-100ms)
• Multiple provider support (OpenAI, Anthropic, Ollama, etc.)
• Streaming responses with real-time feedback
• Comprehensive configuration management
• Plugin system for extensibility
• Model Context Protocol (MCP) integration
• Enterprise-grade security features

Examples:
  gollm chat "What is Go programming language?"
  gollm interactive --model gpt-4
  gollm complete "def fibonacci(n):" --provider openai
  gollm models list --provider anthropic
  gollm config init

For more information, visit: https://docs.gollm.dev`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip initialization for commands that don't need configuration
			if shouldSkipInitialization(cmd) {
				return nil
			}
			return initializeRootContext(cmd, buildInfo)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
            // If no subcommand is provided, show welcome banner and help
            branding.WelcomeBanner(buildInfo.Version)
            return cmd.Help()
        },
    }

	// Add global flags
	addGlobalFlags(rootCmd)

	// Add subcommands
	if err := addSubcommands(rootCmd); err != nil {
		return nil, fmt.Errorf("failed to add subcommands: %w", err)
	}

	// Set custom help template
	rootCmd.SetHelpTemplate(getHelpTemplate())

	// Set version information
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)",
		buildInfo.Version, buildInfo.Commit, buildInfo.BuildTime)

	return rootCmd, nil
}

// addGlobalFlags adds global flags that are available to all commands.
// addSubcommands adds all subcommands to the root command.
func addSubcommands(rootCmd *cobra.Command) error {
	// Core commands
	rootCmd.AddCommand(commands.NewChatCommand())
	rootCmd.AddCommand(commands.NewCompleteCommand())
	rootCmd.AddCommand(commands.NewInteractiveCommand())
	rootCmd.AddCommand(commands.NewEnhancedInteractiveCommand())
	rootCmd.AddCommand(commands.NewTUICommand())
	rootCmd.AddCommand(commands.NewAdvancedCommand())
	rootCmd.AddCommand(commands.NewUltimateCommand())
	rootCmd.AddCommand(commands.NewUltimateWorkingCommand())  // WORKING VERSION!
	rootCmd.AddCommand(commands.NewUltimateRealCommand())     // REAL API VERSION!
	rootCmd.AddCommand(commands.NewUltimateEnhancedCommand()) // ENHANCED WITH CHAT LOOP & AUTOMATION!
	rootCmd.AddCommand(commands.NewChatLoopFixCommand())      // FIXED CONTINUOUS CHAT!
	rootCmd.AddCommand(commands.NewAIOperatorCommand())       // AI AS SYSTEM OPERATOR!
	rootCmd.AddCommand(commands.NewOperatorCommand())         // PC-Operator flows (safe)
	rootCmd.AddCommand(commands.NewCoderCommand())            // Coder workflow (Go-first)
	rootCmd.AddCommand(commands.NewMenuCommand())             // Minimal menu (Operator + Coder)

	// Configuration and management
	rootCmd.AddCommand(commands.NewConfigCommand())
	rootCmd.AddCommand(commands.NewProfileCommand())
	rootCmd.AddCommand(commands.NewModelsCommand())

	// Performance and testing
	rootCmd.AddCommand(commands.NewBenchmarkCommand())

	// Version command
	rootCmd.AddCommand(commands.NewVersionCommand())

	return nil
}

func addGlobalFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()

	// Configuration flags
	flags.StringVarP(&globalFlags.ConfigFile, "config", "c", "",
		"config file path (default: search in ., ~/.gollm, /etc/gollm)")

	// Output and logging flags
	flags.StringVarP(&globalFlags.LogLevel, "log-level", "l", "info",
		"log level (debug, info, warn, error, fatal)")
	flags.StringVarP(&globalFlags.OutputFormat, "output", "o", "text",
		"output format (text, json, yaml, markdown)")
	flags.BoolVar(&globalFlags.NoColor, "no-color", false,
		"disable colored output")
	flags.BoolVarP(&globalFlags.Quiet, "quiet", "q", false,
		"suppress non-error output")
	flags.BoolVarP(&globalFlags.Verbose, "verbose", "v", false,
		"enable verbose output")
	flags.BoolVar(&globalFlags.Debug, "debug", false,
		"enable debug mode with detailed logging")

	// Provider and model flags
	flags.StringVarP(&globalFlags.Provider, "provider", "p", "",
		"LLM provider to use (openai, anthropic, ollama, etc.)")
	flags.StringVarP(&globalFlags.Model, "model", "m", "",
		"model to use for requests")

	// Request parameter flags
	flags.Float64VarP(&globalFlags.Temperature, "temperature", "t", 0,
		"sampling temperature (0.0-2.0, 0=use provider default)")
	flags.IntVar(&globalFlags.MaxTokens, "max-tokens", 0,
		"maximum tokens in response (0=use provider default)")
	flags.DurationVar(&globalFlags.Timeout, "timeout", 0,
		"request timeout (0=use provider default)")

	// Mark some flags as hidden for cleaner help output
	flags.MarkHidden("debug")
}

// shouldSkipInitialization checks if the command should skip root context initialization.
func shouldSkipInitialization(cmd *cobra.Command) bool {
	// Commands that don't need configuration
	// These are interactive/demo modes or commands that manage themselves without global config.
    skipCommands := []string{
        "version",
        "help",
        "completion",
        "ultimate-work",
        "ultimate-real",
        "chat-loop",
        "ai-operator",
        "menu",
        "operator",
        "coder",
    }

	cmdName := cmd.Name()
	for _, skip := range skipCommands {
		if cmdName == skip {
			return true
		}
	}

	// Also skip if parent is help (e.g., "gollm help version")
	if cmd.Parent() != nil && cmd.Parent().Name() == "help" {
		return true
	}

	return false
}

// initializeRootContext initializes the shared root context.
func initializeRootContext(cmd *cobra.Command, buildInfo BuildInfo) error {
	// Initialize configuration
	cfg, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create provider registry
	registry := core.NewProviderRegistry()

	// TODO: Register providers based on configuration
	// This will be implemented when provider packages are ready

	// Create root context
	rootContext = &RootContext{
		Config:      cfg,
		BuildInfo:   buildInfo,
		GlobalFlags: globalFlags,
		Registry:    registry,
	}

	// Set up logging based on flags
	if err := setupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	return nil
}

// loadConfiguration loads the application configuration.
func loadConfiguration() (*config.Config, error) {
	opts := config.LoadOptions{}

	// Use specific config file if provided
	if globalFlags.ConfigFile != "" {
		if _, err := os.Stat(globalFlags.ConfigFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", globalFlags.ConfigFile)
		}
		opts.ConfigFile = globalFlags.ConfigFile
	}

	// Skip validation in debug mode for development
	opts.SkipValidation = globalFlags.Debug

	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		return nil, err
	}

	// Apply CLI flag overrides
	applyFlagOverrides(cfg)

	return cfg, nil
}

// applyFlagOverrides applies CLI flag values to the configuration.
func applyFlagOverrides(cfg *config.Config) {
	// Override provider if specified
	if globalFlags.Provider != "" {
		cfg.DefaultProvider = globalFlags.Provider
	}

	// Override global settings based on flags
	if globalFlags.Temperature > 0 {
		cfg.Settings.Temperature = globalFlags.Temperature
	}

	if globalFlags.MaxTokens > 0 {
		cfg.Settings.MaxTokens = globalFlags.MaxTokens
	}

	if globalFlags.Timeout > 0 {
		cfg.Settings.Timeout = globalFlags.Timeout
	}

	if globalFlags.OutputFormat != "text" {
		cfg.Settings.OutputFormat = globalFlags.OutputFormat
	}

	// Override logging level
	if globalFlags.LogLevel != "info" {
		cfg.Logging.Level = globalFlags.LogLevel
	}

	// Enable debug logging if debug flag is set
	if globalFlags.Debug {
		cfg.Logging.Level = "debug"
	}

	// Set quiet mode
	if globalFlags.Quiet {
		cfg.Logging.Level = "error"
	}
}

// setupLogging configures the application logging.
func setupLogging() error {
	// TODO: Implement proper logging setup based on configuration
	// This will be implemented when the logging package is ready
	return nil
}

// GetRootContext returns the shared root context.
func GetRootContext() *RootContext {
	return rootContext
}

// newVersionCommand creates the version subcommand.
func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display detailed version information including build details.",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.GetBuildInfo()

			// Check for detailed flag
			detailed, _ := cmd.Flags().GetBool("detailed")
			if detailed {
				fmt.Println(info.Detailed())
				return
			}

			// Check for short flag
			short, _ := cmd.Flags().GetBool("short")
			if short {
				fmt.Println(info.Short())
				return
			}

			// Default output - one line summary
			fmt.Printf("GOLLM %s\n", info.String())
		},
	}

	// Add flags for different output formats
	cmd.Flags().BoolP("short", "s", false, "show short version only")
	cmd.Flags().BoolP("detailed", "d", false, "show detailed version information")

	return cmd
}

// newCompletionCommand creates the shell completion command.
func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for GOLLM.

The completion script for each shell will be output to stdout.
You can source it or write it to a file and source it from your shell's
profile script.

Examples:
  # Bash
  gollm completion bash > /etc/bash_completion.d/gollm

  # Zsh
  gollm completion zsh > "${fpath[1]}/_gollm"

  # Fish
  gollm completion fish > ~/.config/fish/completions/gollm.fish

  # PowerShell
  gollm completion powershell > gollm.ps1`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell type: %s", args[0])
			}
		},
	}

	return cmd
}

// getHelpTemplate returns a custom help template for better formatting.
func getHelpTemplate() string {
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

// getGoVersion returns the Go version used to build the binary.
func getGoVersion() string {
	// This will be set by build info in production builds
	return "unknown"
}

// getPlatform returns the platform information.
func getPlatform() string {
	// This will be set by build info in production builds
	return "unknown"
}

// Utility functions for subcommands

// ValidateProvider validates that a provider name is configured.
func ValidateProvider(providerName string) error {
	if rootContext == nil || rootContext.Config == nil {
		return fmt.Errorf("configuration not initialized")
	}

	if !rootContext.Config.HasProvider(providerName) {
		available := rootContext.Config.ListProviders()
		return fmt.Errorf("provider %q not configured; available providers: %s",
			providerName, strings.Join(available, ", "))
	}

	return nil
}

// GetDefaultProvider returns the default provider name and configuration.
func GetDefaultProvider() (string, config.ProviderConfig, error) {
	if rootContext == nil || rootContext.Config == nil {
		return "", config.ProviderConfig{}, fmt.Errorf("configuration not initialized")
	}

	return rootContext.Config.GetDefaultProvider()
}

// GetProvider returns a specific provider configuration.
func GetProvider(name string) (config.ProviderConfig, error) {
	if rootContext == nil || rootContext.Config == nil {
		return config.ProviderConfig{}, fmt.Errorf("configuration not initialized")
	}

	providerConfig, _, err := rootContext.Config.GetProvider(name)
	return providerConfig, err
}

// ResolveConfigPath resolves a configuration file path with proper expansion.
func ResolveConfigPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}

// SetupContext sets up command context with timeout if specified.
func SetupContext(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Apply timeout from global flags or configuration
	var timeout time.Duration
	if globalFlags.Timeout > 0 {
		timeout = globalFlags.Timeout
	} else if rootContext != nil && rootContext.Config != nil {
		timeout = rootContext.Config.Settings.Timeout
	}

	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}

	return context.WithCancel(ctx)
}
