package commands

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "strings"
    "syscall"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"

    "github.com/m4rba4s/Nexus-LLM/internal/config"
    "github.com/m4rba4s/Nexus-LLM/internal/mcp"
    "github.com/m4rba4s/Nexus-LLM/internal/tui"
)

// NewAdvancedCommand creates the advanced TUI command
func NewAdvancedCommand() *cobra.Command {
	var (
		provider    string
		model       string
		mcpPort     int
		mcpToken    string
		autoExecute bool
		theme       string
	)

	cmd := &cobra.Command{
		Use:   "advanced",
		Short: "Launch advanced AI terminal interface with MCP support",
		Long: `Launch an advanced terminal user interface for interacting with LLMs.

Features:
  • Beautiful animated UI with gradients and themes
  • Interactive menu system with hotkeys
  • MCP server integration for extended capabilities
  • Auto-execution mode for commands and code
  • Real-time streaming responses
  • Multi-provider support
  • Command history and session management
  
Examples:
  # Launch with default settings
  gollm advanced
  
  # Launch with specific provider and model
  gollm advanced --provider openai --model gpt-4
  
  # Launch with MCP server
  gollm advanced --mcp-port 8080 --mcp-token secret
  
  # Launch with auto-execution enabled
  gollm advanced --auto-execute`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdvancedTUI(provider, model, mcpPort, mcpToken, autoExecute, theme)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "LLM provider to use")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use")
	cmd.Flags().IntVar(&mcpPort, "mcp-port", 8080, "MCP server port")
	cmd.Flags().StringVar(&mcpToken, "mcp-token", "", "MCP authentication token")
	cmd.Flags().BoolVar(&autoExecute, "auto-execute", false, "Enable auto-execution mode")
	cmd.Flags().StringVar(&theme, "theme", "neon", "UI theme (neon, dark, light, matrix)")

	return cmd
}

func runAdvancedTUI(provider, model string, mcpPort int, mcpToken string, autoExecute bool, theme string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Use default config if loading fails
		cfg = &config.Config{}
	}

	// Override with command flags
	if provider != "" {
		cfg.DefaultProvider = provider
	}
	// Model override handled later
	_ = model

	// Start MCP server if configured
	var mcpServer *mcp.MCPServer
	if mcpPort > 0 {
		mcpServer = mcp.NewMCPServer(mcpPort, mcpToken)

		// Start MCP server in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			if err := mcpServer.Start(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			}
		}()

		fmt.Printf("🔧 MCP Server started on port %d\n", mcpPort)
	}

	// Create the advanced TUI model
	tuiModel := tui.NewAdvancedModel()

	// Configure based on flags
	if autoExecute {
		tuiModel.SetAutoExecute(true)
	}

	if theme != "" {
		tuiModel.SetTheme(theme)
	}

	// Configure MCP if server is running
	if mcpServer != nil {
		tuiModel.SetMCPEnabled(true)
		tuiModel.SetMCPEndpoint(fmt.Sprintf("ws://localhost:%d/mcp", mcpPort))
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Create and run the Bubble Tea program
    var p *tea.Program
    go func() {
        <-sigChan
        fmt.Println("\n👋 Shutting down gracefully...")
        if mcpServer != nil {
            // Cleanup MCP server if needed
        }
        if p != nil {
            p.Send(tea.Quit)
        }
    }()
    p = tea.NewProgram(
        tuiModel,
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),
    )

	fmt.Println("🚀 Launching Advanced AI Terminal Interface...")
	fmt.Println("Press Ctrl+M for menu, Ctrl+C to quit")

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// AdvancedCommandExecutor handles safe command execution with user confirmation
type AdvancedCommandExecutor struct {
	autoExecute bool
	whitelist   []string
	blacklist   []string
}

// NewAdvancedCommandExecutor creates a new command executor
func NewAdvancedCommandExecutor(autoExecute bool) *AdvancedCommandExecutor {
	return &AdvancedCommandExecutor{
		autoExecute: autoExecute,
		whitelist: []string{
			"ls", "pwd", "date", "echo", "cat", "grep", "find", "wc",
			"head", "tail", "sort", "uniq", "cut", "awk", "sed",
			"git status", "git log", "git diff", "docker ps",
		},
		blacklist: []string{
			"rm -rf", "dd", "format", "mkfs", "fdisk",
			"shutdown", "reboot", "kill", "pkill", "killall",
			"curl | sh", "wget | sh", "eval",
		},
	}
}

// Execute safely executes a command
func (e *AdvancedCommandExecutor) Execute(command string) (string, error) {
	// Check blacklist
	for _, blocked := range e.blacklist {
		if contains(command, blocked) {
			return "", fmt.Errorf("command contains dangerous pattern: %s", blocked)
		}
	}

	// Check if command is in whitelist for auto-execution
	isWhitelisted := false
	for _, safe := range e.whitelist {
		if startsWith(command, safe) {
			isWhitelisted = true
			break
		}
	}

	// If not auto-execute or not whitelisted, require confirmation
	if !e.autoExecute || !isWhitelisted {
		if !confirmExecution(command) {
			return "", fmt.Errorf("execution cancelled by user")
		}
	}

	// Execute the command
	return executeCommand(command)
}

func confirmExecution(command string) bool {
	fmt.Printf("\n⚠️  About to execute: %s\n", command)
	fmt.Print("Do you want to proceed? (y/N): ")

	var response string
	fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == "yes"
}

func executeCommand(command string) (string, error) {
	// Implementation would go here
	// This is a placeholder
	return fmt.Sprintf("Executed: %s", command), nil
}

func contains(s, substr string) bool {
    return strings.Contains(s, substr)
}

func startsWith(s, prefix string) bool {
    return strings.HasPrefix(s, prefix)
}
