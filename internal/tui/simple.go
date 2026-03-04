// Package tui provides a simple terminal chat interface for debugging
package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"

	"github.com/yourusername/gollm/internal/config"
)

// SimpleChat represents a basic terminal chat interface
type SimpleChat struct {
	config    *config.Config
	tuiConfig *Config
	aiService *AIService
	messages  []string
	running   bool
	colors    struct {
		primary   *color.Color
		secondary *color.Color
		user      *color.Color
		ai        *color.Color
		system    *color.Color
		error     *color.Color
	}
}

// NewSimpleChat creates a new simple chat interface
func NewSimpleChat(cfg *config.Config, tuiConfig *Config) (*SimpleChat, error) {
	chat := &SimpleChat{
		config:    cfg,
		tuiConfig: tuiConfig,
		messages:  []string{},
		running:   true,
	}

	// Initialize colors
	chat.colors.primary = color.New(color.FgGreen, color.Bold)
	chat.colors.secondary = color.New(color.FgCyan)
	chat.colors.user = color.New(color.FgBlue, color.Bold)
	chat.colors.ai = color.New(color.FgGreen)
	chat.colors.system = color.New(color.FgYellow)
	chat.colors.error = color.New(color.FgRed, color.Bold)

	// Initialize AI service with fallback
    aiService, err := NewAIService(cfg, tuiConfig)
    if err != nil {
		// Fallback to safe mode with mock responses
		chat.colors.error.Printf("⚠️  AI service initialization failed: %v\n", err)
		chat.colors.system.Println("🛡️  Falling back to safe mode with mock responses")
		chat.aiService = nil // Will use mock responses
    } else {
        chat.aiService = aiService
        // Apply initial system message if provided
        if tuiConfig.SystemMessage != "" {
            chat.aiService.SetSystemMessage(tuiConfig.SystemMessage)
        }
    }

	return chat, nil
}

// Run starts the simple chat interface
func (s *SimpleChat) Run() error {
	// Setup signal handling
	s.setupSignalHandling()

	// Show welcome
	s.showWelcome()

	// Show AI service status
	s.showAIStatus()

	// Create scanner for input
	scanner := bufio.NewScanner(os.Stdin)

	// Main chat loop
	for s.running && scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			s.handleCommand(input)
			continue
		}

		// Handle regular messages
		s.handleUserMessage(input)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("input error: %w", err)
	}

	return nil
}

// setupSignalHandling sets up graceful shutdown
func (s *SimpleChat) setupSignalHandling() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-c
        s.colors.system.Println("\n👋 Goodbye! Thanks for using GOLLM!")
        s.running = false
    }()
}

// showWelcome displays the welcome message
func (s *SimpleChat) showWelcome() {
	// Clear screen
	fmt.Print("\033[2J\033[H")

	// ASCII art header
	s.colors.primary.Println("╔══════════════════════════════════════════════════════════╗")
	s.colors.primary.Println("║                    🤖 GOLLM SIMPLE CHAT                 ║")
	s.colors.primary.Println("║                  Debug Terminal Interface                ║")
	s.colors.primary.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	s.colors.system.Println("✅ Simple chat interface loaded successfully!")
	s.colors.system.Println("💬 Type your message and press Enter")
	s.colors.system.Println("📝 Available commands:")
	s.colors.secondary.Println("   /help    - Show help")
	s.colors.secondary.Println("   /quit    - Exit chat")
	s.colors.secondary.Println("   /clear   - Clear screen")
	s.colors.secondary.Println("   /status  - Show status")
	s.colors.secondary.Println("   /test    - Test AI connection")
	s.colors.secondary.Println("   /system  - Set system message")
	fmt.Println()

	s.showPrompt()
}

// showAIStatus displays the AI service initialization status
func (s *SimpleChat) showAIStatus() {
	if s.aiService == nil {
		s.colors.system.Println("🛡️  Running in Safe Mode (Mock Responses)")
		s.colors.system.Println("   Real AI integration failed - using fallback responses")
	} else {
		providerName, model, err := s.aiService.GetProviderInfo()
		if err != nil {
			s.colors.error.Printf("⚠️  AI Service Error: %v\n", err)
		} else {
			s.colors.system.Printf("🤖 AI Ready: %s (%s)\n", providerName, model)
		}
	}
	fmt.Println()
}

// showPrompt displays the input prompt
func (s *SimpleChat) showPrompt() {
	s.colors.user.Print("You> ")
}

// handleCommand processes chat commands
func (s *SimpleChat) handleCommand(input string) {
	cmd := strings.ToLower(strings.TrimSpace(input))

	switch cmd {
	case "/help":
		s.showHelp()
	case "/quit", "/exit":
		s.colors.system.Println("👋 Goodbye!")
		s.running = false
		return
	case "/clear":
		fmt.Print("\033[2J\033[H")
		s.showWelcome()
		return
	case "/status":
		s.showStatus()
	case "/test":
		s.testConnection()
	case "/system":
		s.colors.system.Println("💬 Enter system message (or empty to clear):")
		s.colors.user.Print("System> ")
		return
	default:
		if strings.HasPrefix(cmd, "/system ") {
			systemMsg := strings.TrimSpace(cmd[8:])
			s.setSystemMessage(systemMsg)
		} else {
			s.colors.error.Printf("❌ Unknown command: %s\n", cmd)
			s.colors.system.Println("Type /help for available commands")
		}
	}

	s.showPrompt()
}

// showHelp displays help information
func (s *SimpleChat) showHelp() {
	fmt.Println()
	s.colors.primary.Println("🔧 GOLLM Simple Chat - Help")
	s.colors.primary.Println("═════════════════════════════")
	fmt.Println()
	s.colors.system.Println("This is a debug version of GOLLM's TUI interface.")
	s.colors.system.Println("It uses simple terminal I/O instead of complex TUI frameworks.")
	fmt.Println()
	s.colors.secondary.Println("Commands:")
	fmt.Println("  /help    - Show this help")
	fmt.Println("  /quit    - Exit the application")
	fmt.Println("  /clear   - Clear the screen")
	fmt.Println("  /status  - Show configuration status")
	fmt.Println("  /test    - Test AI connection")
	fmt.Println("  /system <msg> - Set system message")
	fmt.Println()
	s.colors.secondary.Println("Usage:")
	fmt.Println("  - Type any message and press Enter to chat")
	fmt.Println("  - Press Ctrl+C to exit anytime")
	fmt.Println()
}

// showStatus displays current status
func (s *SimpleChat) showStatus() {
	fmt.Println()
	s.colors.primary.Println("📊 GOLLM Status")
	s.colors.primary.Println("═══════════════")
	fmt.Println()

	// Get provider info
	var providerName, model string
	var totalTokens, requestCount int
	var conversationLength int

	if s.aiService != nil {
		var err error
		providerName, model, err = s.aiService.GetProviderInfo()
		if err != nil {
			providerName = "ERROR"
			model = "ERROR"
		}
		totalTokens, requestCount = s.aiService.GetStats()
		conversationLength = s.aiService.GetConversationLength()
	} else {
		providerName = "Mock/Safe Mode"
		model = "fallback"
		requestCount = len(s.messages) / 2 // Approximate
		conversationLength = len(s.messages)
	}

	s.colors.secondary.Printf("Provider: ")
	s.colors.system.Println(providerName)

	s.colors.secondary.Printf("Model: ")
	s.colors.system.Println(model)

	s.colors.secondary.Printf("Theme: ")
	s.colors.system.Println(s.tuiConfig.Theme)

	s.colors.secondary.Printf("Debug Mode: ")
	if s.tuiConfig.Debug {
		s.colors.system.Println("ON")
	} else {
		s.colors.system.Println("OFF")
	}

	s.colors.secondary.Printf("Messages Sent: ")
	s.colors.system.Printf("%d\n", requestCount)

	s.colors.secondary.Printf("Total Tokens: ")
	s.colors.system.Printf("%d\n", totalTokens)

	s.colors.secondary.Printf("Conversation Length: ")
	s.colors.system.Printf("%d\n", conversationLength)

	fmt.Println()
}

// handleUserMessage processes a user message
func (s *SimpleChat) handleUserMessage(message string) {
	// Add timestamp
	timestamp := time.Now().Format("15:04:05")

	// Display user message
	s.colors.user.Printf("[%s] You: ", timestamp)
	fmt.Println(message)

	// Store message
	s.messages = append(s.messages, fmt.Sprintf("User: %s", message))

	// Show thinking indicator
	s.showThinking()

	var responseContent string
	var responseError error
	var latency time.Duration
	var tokens int

	if s.aiService != nil {
		// Send to real AI service
		response, err := s.aiService.SendMessage(message)
		if err != nil {
			responseError = err
		} else if response.IsError {
			responseError = response.Error
		} else {
			responseContent = response.Content
			latency = response.Latency
			tokens = response.Tokens
		}
	} else {
		// Use mock responses (safe mode)
		responseContent = s.generateMockResponse(message)
		latency = 100 * time.Millisecond // Simulate latency
		tokens = 0
	}

	if responseError != nil {
		s.colors.error.Printf("[%s] Error: ", time.Now().Format("15:04:05"))
		fmt.Printf("Failed to get AI response: %v\n", responseError)
	} else {
		// Display AI response
		s.colors.ai.Printf("[%s] AI: ", time.Now().Format("15:04:05"))
		fmt.Println(responseContent)

		// Show response stats if in debug mode
		if s.tuiConfig.Debug && s.aiService != nil {
			s.colors.system.Printf("    [Tokens: %d, Latency: %v]\n", tokens, latency.Round(time.Millisecond))
		}
	}

	// Store AI response
	s.messages = append(s.messages, fmt.Sprintf("AI: %s", responseContent))

	fmt.Println()
	s.showPrompt()
}

// showThinking displays a thinking indicator
func (s *SimpleChat) showThinking() {
	s.colors.system.Print("🤔 AI is thinking")

	for i := 0; i < 3; i++ {
		time.Sleep(200 * time.Millisecond)
		s.colors.system.Print(".")
	}

	fmt.Print("\r" + strings.Repeat(" ", 20) + "\r") // Clear line
}

// testConnection tests the AI service connection
func (s *SimpleChat) testConnection() {
	s.colors.system.Println("🔧 Testing AI connection...")

	if s.aiService == nil {
		s.colors.system.Println("🛡️  Running in Safe Mode - Mock responses only")
		s.colors.system.Println("✅ Mock response test successful!")
	} else {
		err := s.aiService.TestConnection()
		if err != nil {
			s.colors.error.Printf("❌ Connection test failed: %v\n", err)
		} else {
			s.colors.system.Println("✅ Connection test successful!")

			providerName, model, _ := s.aiService.GetProviderInfo()
			s.colors.system.Printf("Connected to: %s (%s)\n", providerName, model)
		}
	}

	fmt.Println()
	s.showPrompt()
}

// setSystemMessage sets or clears the system message
func (s *SimpleChat) setSystemMessage(message string) {
	if s.aiService == nil {
		s.colors.system.Println("🛡️  System messages not available in Safe Mode")
	} else {
		if message == "" {
			s.aiService.SetSystemMessage("")
			s.colors.system.Println("✅ System message cleared")
		} else {
			s.aiService.SetSystemMessage(message)
			s.colors.system.Printf("✅ System message set: %s\n", message)
		}
	}

	fmt.Println()
	s.showPrompt()
}

// generateMockResponse creates a mock AI response for safe mode
func (s *SimpleChat) generateMockResponse(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	// Predefined responses
	responses := map[string]string{
		"hello":             "Hello! I'm running in Safe Mode with mock responses. How can I help you today?",
		"hi":                "Hi there! Welcome to GOLLM Safe Mode. What would you like to know?",
		"test":              "✅ Safe Mode test successful! Mock responses are working correctly.",
		"test connection":   "🛡️  Safe Mode active - using fallback responses instead of real AI.",
		"help":              "I'm here to help! This is Safe Mode with mock responses. Ask me anything and I'll provide a basic response.",
		"who are you":       "I'm a mock AI assistant in GOLLM Safe Mode. The real AI integration failed to initialize.",
		"quit":              "Goodbye! Thanks for using GOLLM Safe Mode.",
		"status":            "Safe Mode operational! Using mock responses instead of real AI.",
		"what is safe mode": "Safe Mode activates when real AI providers fail to initialize. It provides mock responses so you can still test the interface.",
	}

	// Check for exact matches
	if response, exists := responses[input]; exists {
		return response
	}

	// Check for partial matches
	for key, response := range responses {
		if strings.Contains(input, key) {
			return response
		}
	}

	// Default response
	return fmt.Sprintf("🛡️  Safe Mode Response: I received your message '%s'. This is a mock response since real AI integration failed. The interface is working correctly in fallback mode!", input)
}

// RunSimpleChat starts the simple chat interface
func RunSimpleChat(cfg *config.Config, tuiConfig *Config) error {
	chat, err := NewSimpleChat(cfg, tuiConfig)
	if err != nil {
		return fmt.Errorf("failed to create simple chat: %w", err)
	}
	return chat.Run()
}
