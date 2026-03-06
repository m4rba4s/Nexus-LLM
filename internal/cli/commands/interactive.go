// Package commands provides interactive chat functionality for GOLLM.
package commands

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/m4rba4s/Nexus-LLM/internal/core"

	// Import providers to trigger their init() registration
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/anthropic"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/mock"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/openai"
)

// InteractiveFlags contains flags specific to the interactive command.
type InteractiveFlags struct {
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
	SaveHistory      bool
	HistoryFile      string
	Multiline        bool
	Quiet            bool
	ShowTokens       bool
	Stop             []string
}

// InteractiveSession represents an active interactive chat session.
type InteractiveSession struct {
	provider     core.Provider
	model        string
	flags        *InteractiveFlags
	conversation []core.Message
	totalTokens  int
	requestCount int
	startTime    time.Time
	out          io.Writer
	errOut       io.Writer
	in           io.Reader
}

// NewInteractiveCommand creates the interactive chat command.
func NewInteractiveCommand() *cobra.Command {
	flags := &InteractiveFlags{}

	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Start an interactive chat session",
		Long: `Start an interactive chat session with an LLM provider.

The interactive mode allows you to have a continuous conversation with
the LLM without needing to run separate commands for each message.
The conversation context is maintained throughout the session.

Features:
  • Continuous conversation with context
  • Real-time streaming responses
  • Conversation history
  • Multi-line input support
  • Token usage tracking
  • Session statistics

Examples:
  # Start interactive session with default provider
  gollm interactive

  # Start with specific provider and model
  gollm interactive --provider openai --model gpt-4

  # Start with system message
  gollm interactive --system "You are a helpful coding assistant"

  # Enable multiline input mode
  gollm interactive --multiline

  # Save conversation history
  gollm interactive --save-history --history-file chat.log

Controls:
  • Type your message and press Enter to send
  • Type /quit or /exit to end the session
  • Type /help to see available commands
  • Type /clear to clear conversation history
  • Type /stats to see session statistics
  • Type /system <message> to set system message
  • Use Ctrl+C to exit immediately`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveCommand(cmd, flags)
		},
	}

	// Add command-specific flags
	addInteractiveFlags(cmd, flags)

	return cmd
}

// addInteractiveFlags adds flags specific to the interactive command.
func addInteractiveFlags(cmd *cobra.Command, flags *InteractiveFlags) {
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

	// Session behavior flags
	f.BoolVar(&flags.Stream, "stream", false,
		"stream response tokens")
	f.BoolVar(&flags.NoStream, "no-stream", false,
		"disable streaming and wait for complete response")
	f.BoolVar(&flags.SaveHistory, "save-history", false,
		"save conversation history to file")
	f.StringVar(&flags.HistoryFile, "history-file", "gollm_history.txt",
		"file to save conversation history")
	f.BoolVar(&flags.Multiline, "multiline", false,
		"enable multiline input mode")
	f.BoolVar(&flags.Quiet, "quiet", false,
		"minimal output, only show responses")
	f.BoolVar(&flags.ShowTokens, "show-tokens", true,
		"show token usage after each response")

	// Mark mutually exclusive flags
	cmd.MarkFlagsMutuallyExclusive("stream", "no-stream")

	// Add completion for provider and model flags
	cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	cmd.RegisterFlagCompletionFunc("model", completeModels)
}

// runInteractiveCommand executes the interactive chat command.
func runInteractiveCommand(cmd *cobra.Command, flags *InteractiveFlags) error {
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

	// Create session
	session := &InteractiveSession{
		provider:     provider,
		model:        model,
		flags:        flags,
		conversation: []core.Message{},
		startTime:    time.Now(),
		out:          cmd.OutOrStdout(),
		errOut:       cmd.ErrOrStderr(),
		in:           cmd.InOrStdin(),
	}

	// Add system message if provided
	if flags.SystemMessage != "" {
		session.conversation = append(session.conversation, core.Message{
			Role:    core.RoleSystem,
			Content: flags.SystemMessage,
		})
	}

	// Show welcome message
	if !flags.Quiet {
		showWelcomeMessage(providerName, model, flags)
		fmt.Fprintln(cmd.OutOrStdout(), "Interactive mode")
	}

	// Start interactive loop
	return session.run()
}

// run starts the main interactive loop.
func (s *InteractiveSession) run() error {
	if s.in == nil {
		s.in = os.Stdin
	}
	if s.out == nil {
		s.out = os.Stdout
	}
	if s.errOut == nil {
		s.errOut = os.Stderr
	}
	reader := bufio.NewReader(s.in)

	for {
		// Show prompt
		if !s.flags.Quiet {
			fmt.Fprint(s.out, "\n> ")
		}

		// Read user input
		var input string
		var err error

		if s.flags.Multiline {
			input, err = s.readMultilineInput(reader)
		} else {
			input, err = reader.ReadString('\n')
		}

		if err != nil {
			if err == io.EOF {
				if !s.flags.Quiet {
					fmt.Fprintln(s.out, "\nGoodbye!")
				}
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if handled := s.handleCommand(input); handled {
				lower := strings.ToLower(strings.TrimSpace(input))
				if lower == "/quit" || lower == "/exit" || lower == "/q" {
					return nil
				}
				continue
			}
		}

		// Process chat message
		if err := s.processMessage(input); err != nil {
			fmt.Fprintf(s.errOut, "Error: %v\n", err)
			continue
		}
	}
}

// readMultilineInput reads multiline input ending with double newline or EOF.
func (s *InteractiveSession) readMultilineInput(reader *bufio.Reader) (string, error) {
	if !s.flags.Quiet {
		fmt.Fprintln(s.out, "(Type your message. Press Enter twice to send, Ctrl+D to finish)")
	}

	var lines []string
	emptyLines := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(lines) > 0 {
				break
			}
			return "", err
		}

		line = strings.TrimRight(line, "\n")
		if line == "" {
			emptyLines++
			if emptyLines >= 2 {
				break
			}
		} else {
			emptyLines = 0
		}

		lines = append(lines, line)
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n"), nil
}

// handleCommand processes special commands.
func (s *InteractiveSession) handleCommand(input string) bool {
	parts := strings.Fields(input)
	command := strings.ToLower(parts[0])

	switch command {
	case "/quit", "/exit", "/q":
		if !s.flags.Quiet {
			s.showSessionStats()
			fmt.Fprintln(s.out, "Goodbye!")
		}
		return true

	case "/help", "/h":
		s.showHelp()
		return true

	case "/clear", "/c":
		s.clearHistory()
		if !s.flags.Quiet {
			fmt.Fprintln(s.out, "Conversation history cleared.")
		}
		return true

	case "/stats", "/s":
		s.showSessionStats()
		return true

	case "/system":
		if len(parts) < 2 {
			fmt.Fprintln(s.out, "Usage: /system <message>")
			return true
		}
		systemMsg := strings.Join(parts[1:], " ")
		s.setSystemMessage(systemMsg)
		if !s.flags.Quiet {
			fmt.Fprintf(s.out, "System message set: %s\n", systemMsg)
		}
		return true

	case "/history":
		s.showHistory()
		return true

	case "/save":
		filename := s.flags.HistoryFile
		if len(parts) > 1 {
			filename = parts[1]
		}
		if err := s.saveHistory(filename); err != nil {
			fmt.Fprintf(s.errOut, "Failed to save history: %v\n", err)
		} else {
			fmt.Fprintf(s.out, "History saved to %s\n", filename)
		}
		return true

	default:
		return false
	}
}

// processMessage processes a chat message and gets response from LLM.
func (s *InteractiveSession) processMessage(message string) error {
	// Add user message to conversation
	s.conversation = append(s.conversation, core.Message{
		Role:    core.RoleUser,
		Content: message,
	})

	// Build request
	request := s.buildRequest()

	// Get context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send request and handle response
	shouldStream := s.flags.Stream && !s.flags.NoStream

	var assistantMessage string
	var usage *core.Usage

	if shouldStream {
		assistantMessage, usage = s.handleStreamingResponse(ctx, request)
	} else {
		assistantMessage, usage = s.handleNonStreamingResponse(ctx, request)
	}

	// Add assistant response to conversation
	if assistantMessage != "" {
		s.conversation = append(s.conversation, core.Message{
			Role:    core.RoleAssistant,
			Content: assistantMessage,
		})
	}

	// Update session stats
	s.requestCount++
	if usage != nil {
		s.totalTokens += usage.TotalTokens
	}

	// Show token usage
	if s.flags.ShowTokens && !s.flags.Quiet && usage != nil {
		fmt.Fprintf(os.Stderr, "\n[Tokens: %d prompt + %d completion = %d total | Session total: %d]\n",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, s.totalTokens)
	}

	return nil
}

// buildRequest constructs a completion request from the current conversation.
func (s *InteractiveSession) buildRequest() *core.CompletionRequest {
	request := &core.CompletionRequest{
		Model:     s.model,
		Messages:  s.conversation,
		CreatedAt: time.Now(),
	}

	// Set optional parameters from flags
	if s.flags.MaxTokens > 0 {
		request.MaxTokens = &s.flags.MaxTokens
	}
	if s.flags.Temperature > 0 {
		request.Temperature = &s.flags.Temperature
	}
	if s.flags.TopP > 0 {
		request.TopP = &s.flags.TopP
	}
	if s.flags.FrequencyPenalty != 0 {
		request.FrequencyPenalty = &s.flags.FrequencyPenalty
	}
	if s.flags.PresencePenalty != 0 {
		request.PresencePenalty = &s.flags.PresencePenalty
	}
	if len(s.flags.Stop) > 0 {
		request.Stop = s.flags.Stop
	}

	request.Stream = s.flags.Stream && !s.flags.NoStream

	return request
}

// handleStreamingResponse handles streaming LLM responses.
func (s *InteractiveSession) handleStreamingResponse(ctx context.Context, request *core.CompletionRequest) (string, *core.Usage) {
	streamer, ok := s.provider.(core.Streamer)
	if !ok {
		// Fall back to non-streaming
		return s.handleNonStreamingResponse(ctx, request)
	}

	chunks, err := streamer.StreamCompletion(ctx, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Streaming error: %v\n", err)
		return "", nil
	}

	var responseBuilder strings.Builder
	var finalUsage *core.Usage

	if !s.flags.Quiet {
		fmt.Print("\n")
	}

	for chunk := range chunks {
		if chunk.Error != nil {
			fmt.Fprintf(os.Stderr, "Stream error: %v\n", chunk.Error)
			break
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				responseBuilder.WriteString(content)
				if !s.flags.Quiet {
					fmt.Print(content)
				}
			}
		}

		if chunk.Usage != nil {
			finalUsage = chunk.Usage
		}

		if chunk.Done {
			break
		}
	}

	if !s.flags.Quiet {
		fmt.Println()
	}

	return responseBuilder.String(), finalUsage
}

// handleNonStreamingResponse handles non-streaming LLM responses.
func (s *InteractiveSession) handleNonStreamingResponse(ctx context.Context, request *core.CompletionRequest) (string, *core.Usage) {
	request.Stream = false

	response, err := s.provider.CreateCompletion(ctx, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Completion error: %v\n", err)
		return "", nil
	}

	var responseContent string
	if len(response.Choices) > 0 {
		responseContent = response.Choices[0].Message.Content
	}

	if !s.flags.Quiet {
		fmt.Fprintf(s.out, "\n%s\n", responseContent)
	}

	return responseContent, &response.Usage
}

// displayInteractiveLogo displays the GOLLM ASCII logo for interactive mode
func displayInteractiveLogo() {
	miniLogo := `GOLLM ░▒▓█▓▒░`

	// Apply colors if supported
	if shouldUseColorsInInteractive() {
		coloredLogo := color.New(color.FgHiCyan, color.Bold).Sprint(miniLogo)
		fmt.Println("              " + coloredLogo)

		// Add tagline
		tagline := "🤖 Interactive Chat Mode • Type /help for commands • /quit to exit"
		fmt.Println(color.New(color.FgHiWhite).Sprint("         " + tagline))
	} else {
		fmt.Println("              " + miniLogo)
		fmt.Println("         🤖 Interactive Chat Mode • Type /help for commands • /quit to exit")
	}
}

// shouldUseColorsInInteractive determines if colors should be used in interactive mode
func shouldUseColorsInInteractive() bool {
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for FORCE_COLOR environment variable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Use fatih/color's built-in detection
	return !color.NoColor
}

// showWelcomeMessage displays the session welcome message.
func showWelcomeMessage(provider, model string, flags *InteractiveFlags) {
	// Display GOLLM ASCII logo
	displayInteractiveLogo()
	fmt.Println()

	fmt.Printf("🤖 Interactive Chat Mode\n")
	fmt.Printf("═══════════════════════════\n")
	fmt.Printf("Provider: %s\n", provider)
	fmt.Printf("Model: %s\n", model)
	if flags.SystemMessage != "" {
		fmt.Printf("System: %s\n", flags.SystemMessage)
	}
	fmt.Printf("\nType your message and press Enter to chat.")
	fmt.Printf("\nType /help for commands or /quit to exit.\n")
	fmt.Printf("═══════════════════════════\n")
}

// showHelp displays available commands.
func (s *InteractiveSession) showHelp() {
	fmt.Printf("\n🔧 Interactive Commands:\n")
	fmt.Printf("  /help, /h          Show this help message\n")
	fmt.Printf("  /quit, /exit, /q   Exit the session\n")
	fmt.Printf("  /clear, /c         Clear conversation history\n")
	fmt.Printf("  /stats, /s         Show session statistics\n")
	fmt.Printf("  /system <msg>      Set system message\n")
	fmt.Printf("  /history           Show conversation history\n")
	fmt.Printf("  /save [file]       Save conversation to file\n")
	fmt.Printf("\n💡 Tips:\n")
	fmt.Printf("  • Use --multiline flag for multi-line input\n")
	fmt.Printf("  • Conversation context is maintained throughout the session\n")
	fmt.Printf("  • Use Ctrl+C for immediate exit\n")
}

// showSessionStats displays session statistics.
func (s *InteractiveSession) showSessionStats() {
	duration := time.Since(s.startTime)

	fmt.Printf("\n📊 Session Statistics:\n")
	fmt.Printf("  Duration: %v\n", duration.Round(time.Second))
	fmt.Printf("  Messages: %d\n", s.requestCount)
	fmt.Printf("  Total Tokens: %d\n", s.totalTokens)
	fmt.Printf("  Conversation Length: %d messages\n", len(s.conversation))

	if s.requestCount > 0 {
		avgTokens := float64(s.totalTokens) / float64(s.requestCount)
		fmt.Printf("  Average Tokens/Request: %.1f\n", avgTokens)
	}
}

// clearHistory clears the conversation history except system messages.
func (s *InteractiveSession) clearHistory() {
	var systemMessages []core.Message
	for _, msg := range s.conversation {
		if msg.Role == core.RoleSystem {
			systemMessages = append(systemMessages, msg)
		}
	}
	s.conversation = systemMessages
	s.totalTokens = 0
	s.requestCount = 0
}

// setSystemMessage sets or updates the system message.
func (s *InteractiveSession) setSystemMessage(message string) {
	// Remove existing system messages
	var nonSystemMessages []core.Message
	for _, msg := range s.conversation {
		if msg.Role != core.RoleSystem {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	// Add new system message at the beginning
	s.conversation = append([]core.Message{{
		Role:    core.RoleSystem,
		Content: message,
	}}, nonSystemMessages...)
}

// showHistory displays the conversation history.
func (s *InteractiveSession) showHistory() {
	if len(s.conversation) == 0 {
		fmt.Println("No conversation history.")
		return
	}

	fmt.Printf("\n📜 Conversation History (%d messages):\n", len(s.conversation))
	fmt.Printf("═══════════════════════════\n")

	for i, msg := range s.conversation {
		var roleIcon string
		switch msg.Role {
		case core.RoleSystem:
			roleIcon = "⚙️"
		case core.RoleUser:
			roleIcon = "👤"
		case core.RoleAssistant:
			roleIcon = "🤖"
		default:
			roleIcon = "❓"
		}

		fmt.Printf("%s %s (%d):\n", roleIcon, strings.ToUpper(msg.Role), i+1)

		// Truncate long messages for display
		content := msg.Content
		if len(content) > 200 {
			content = content[:197] + "..."
		}

		// Add indentation to content
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}
}

// saveHistory saves the conversation history to a file.
func (s *InteractiveSession) saveHistory(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "GOLLM Interactive Chat History\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Duration: %v\n", time.Since(s.startTime).Round(time.Second))
	fmt.Fprintf(file, "Messages: %d\n", len(s.conversation))
	fmt.Fprintf(file, "Total Tokens: %d\n\n", s.totalTokens)
	fmt.Fprintf(file, "═══════════════════════════\n\n")

	for i, msg := range s.conversation {
		fmt.Fprintf(file, "[%s] %s:\n", strings.ToUpper(msg.Role), time.Now().Format("15:04:05"))
		fmt.Fprintf(file, "%s\n\n", msg.Content)

		if i < len(s.conversation)-1 {
			fmt.Fprintf(file, "───────────────────────────\n\n")
		}
	}

	return nil
}
