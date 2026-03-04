package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// EnhancedState includes conversation history and system info
type EnhancedState struct {
	GlobalState
	ConversationHistory []map[string]string
	SystemInfo          map[string]string
	CommandMode         bool
}

var enhancedState = &EnhancedState{
	GlobalState: GlobalState{
		CurrentProvider: "openrouter",
		CurrentModel:    "anthropic/claude-3.5-sonnet",
		APIKeys:         make(map[string]string),
		SystemPrompt:    "You are an advanced AI assistant with system command execution capabilities. When asked to execute commands, you can run them by prefixing with EXECUTE: followed by the command. You help automate tasks and manage systems efficiently.",
		Temperature:     0.7,
		MaxTokens:       4096,
		StreamMode:      true,
		Models:          getLatestModels(),
	},
	ConversationHistory: []map[string]string{},
	SystemInfo:          make(map[string]string),
	CommandMode:         true,
}

// NewUltimateEnhancedCommand creates the enhanced ultimate command
func NewUltimateEnhancedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ultimate-enhanced",
		Short: "🚀 Enhanced Ultimate Mode with automation",
		Long:  `Launch the Enhanced Ultimate AI Interface with chat loop, command execution, and system learning!`,
		RunE:  runEnhancedUltimate,
	}
}

func runEnhancedUltimate(cmd *cobra.Command, args []string) error {
	clearEnhancedScreen()
	loadEnhancedKeys()
	_ = fetchOpenRouterModelsEnhanced()
	showEnhancedLogo()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		showEnhancedMenu()
		fmt.Print("\n🎯 Select option (0-9): ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "0", "q", "exit":
			fmt.Println("\n👋 Goodbye! Thanks for using Enhanced Ultimate!")
			saveEnhancedKeys()
			return nil

		case "1":
			handleEnhancedChat(scanner)

		case "2":
			handleEnhancedModelSelection(scanner)

		case "3":
			handleEnhancedAPIKeys(scanner)

		case "4":
			handleEnhancedSettings(scanner)

		case "5":
			handleEnhancedSystemPrompt(scanner)

		case "6":
			handleEnhancedTestAPI(scanner)

		case "7":
			handleEnhancedSystemInfo()

		case "8":
			handleEnhancedQuickSetup(scanner)

		case "9":
			handleEnhancedHelp()

		default:
			fmt.Println("\n❌ Invalid choice. Try again.")
		}

		if choice != "1" { // Don't wait after chat
			fmt.Println("\n📌 Press Enter to continue...")
			scanner.Scan()
		}
		clearEnhancedScreen()
	}

	return nil
}

func handleEnhancedChat(scanner *bufio.Scanner) {
	clearEnhancedScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║              💬 ENHANCED AI CHAT WITH AUTOMATION             ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Printf("\n🤖 Model: %s\n", getModelName(enhancedState.CurrentModel))
	fmt.Printf("🌡️ Temperature: %.1f | 📊 Max Tokens: %d | ⚡ Stream: %v\n",
		enhancedState.Temperature, enhancedState.MaxTokens, enhancedState.StreamMode)
	fmt.Printf("🔧 Command Mode: %v\n", enhancedState.CommandMode)

	// Initialize conversation history
	if enhancedState.ConversationHistory == nil {
		enhancedState.ConversationHistory = []map[string]string{}
	}

	fmt.Println("\n💡 Commands:")
	fmt.Println("  • Type 'exit' to quit chat")
	fmt.Println("  • Type 'clear' to clear history")
	fmt.Println("  • Type '/cmd <command>' to execute system commands")
	fmt.Println("  • Type '/learn' to analyze and remember this system")
	fmt.Println("  • Type '/auto' to toggle automatic command execution")
	fmt.Println("\n" + strings.Repeat("─", 65))

	// IMPORTANT: Chat loop - keep conversation going!
	for {
		fmt.Print("\n💭 You> ")

		if !scanner.Scan() {
			break
		}

		message := scanner.Text()

		// Handle special commands
		if strings.ToLower(message) == "exit" {
			fmt.Println("👋 Exiting chat...")
			break
		}

		if strings.ToLower(message) == "clear" {
			enhancedState.ConversationHistory = []map[string]string{}
			fmt.Println("🗑️ Conversation history cleared")
			continue
		}

		if strings.HasPrefix(message, "/cmd ") {
			command := strings.TrimPrefix(message, "/cmd ")
			executeSystemCommand(command)
			continue
		}

		if message == "/learn" {
			learnSystemInfo()
			continue
		}

		if message == "/auto" {
			enhancedState.CommandMode = !enhancedState.CommandMode
			fmt.Printf("🔧 Automatic command execution: %v\n", enhancedState.CommandMode)
			continue
		}

		// Add user message to history
		enhancedState.ConversationHistory = append(enhancedState.ConversationHistory,
			map[string]string{"role": "user", "content": message})

		fmt.Println("\n🤖 AI> ")
		fmt.Println(strings.Repeat("─", 65))

		response, err := callEnhancedAPIWithHistory()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Println(response)

			// Add AI response to history
			enhancedState.ConversationHistory = append(enhancedState.ConversationHistory,
				map[string]string{"role": "assistant", "content": response})

			// Check if AI wants to execute commands
			if enhancedState.CommandMode && strings.Contains(response, "EXECUTE:") {
				handleAICommands(response)
			}
		}

		fmt.Println(strings.Repeat("─", 65))
	}
}

func callEnhancedAPIWithHistory() (string, error) {
	messages := []map[string]string{
		{"role": "system", "content": enhancedState.SystemPrompt},
	}

	// Add system info if available
	if len(enhancedState.SystemInfo) > 0 {
		systemContext := "System Information:\n"
		for key, value := range enhancedState.SystemInfo {
			systemContext += fmt.Sprintf("- %s: %s\n", key, value)
		}
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": systemContext,
		})
	}

	// Add conversation history (limit to last 10 messages for context window)
	start := 0
	if len(enhancedState.ConversationHistory) > 10 {
		start = len(enhancedState.ConversationHistory) - 10
	}
	for _, msg := range enhancedState.ConversationHistory[start:] {
		messages = append(messages, msg)
	}

	return callEnhancedAPI(messages)
}

func callEnhancedAPI(messages []map[string]string) (string, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model":       enhancedState.CurrentModel,
		"messages":    messages,
		"temperature": enhancedState.Temperature,
		"max_tokens":  enhancedState.MaxTokens,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	apiKey := enhancedState.APIKeys[enhancedState.CurrentProvider]
	if apiKey == "" {
		return "", fmt.Errorf("missing API key for provider %s", enhancedState.CurrentProvider)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/m4rba4s/Nexus-LLM")
	req.Header.Set("X-Title", "Nexus LLM Enhanced")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %s", string(body))
	}

	if errorData, ok := result["error"]; ok {
		if em, ok := errorData.(map[string]interface{}); ok {
			return "", fmt.Errorf("API error: %v", em["message"])
		}
		return "", fmt.Errorf("API error: %v", errorData)
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		if message, ok := choice["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				return content, nil
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func executeSystemCommand(command string) {
	fmt.Printf("\n⚙️ Executing: %s\n", command)
	fmt.Println(strings.Repeat("─", 65))

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	}

	fmt.Println(string(output))
	fmt.Println(strings.Repeat("─", 65))

	// Add to conversation history
	enhancedState.ConversationHistory = append(enhancedState.ConversationHistory,
		map[string]string{"role": "system", "content": fmt.Sprintf("Command executed: %s\nOutput: %s", command, string(output))})
}

func handleAICommands(response string) {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "EXECUTE:") {
			command := strings.TrimSpace(strings.TrimPrefix(line, "EXECUTE:"))
			fmt.Printf("\n🤖 AI wants to execute: %s\n", command)
			fmt.Print("Allow? (y/n): ")

			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))

			if answer == "y" || answer == "yes" {
				executeSystemCommand(command)
			} else {
				fmt.Println("⛔ Command execution denied")
			}
		}
	}
}

func learnSystemInfo() {
	fmt.Println("\n🧠 Learning system information...")

	// Gather system info
	enhancedState.SystemInfo["OS"] = runtime.GOOS
	enhancedState.SystemInfo["Architecture"] = runtime.GOARCH
	enhancedState.SystemInfo["CPUs"] = fmt.Sprintf("%d", runtime.NumCPU())
	enhancedState.SystemInfo["Go Version"] = runtime.Version()

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		enhancedState.SystemInfo["Hostname"] = hostname
	}

	// Get current directory
	if pwd, err := os.Getwd(); err == nil {
		enhancedState.SystemInfo["Working Directory"] = pwd
	}

	// Get user info
	if user := os.Getenv("USER"); user != "" {
		enhancedState.SystemInfo["User"] = user
	}

	// Get more system info based on OS
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		// Get kernel version
		if output, err := exec.Command("uname", "-r").Output(); err == nil {
			enhancedState.SystemInfo["Kernel"] = strings.TrimSpace(string(output))
		}

		// Get memory info
		if runtime.GOOS == "linux" {
			if output, err := exec.Command("free", "-h").Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				if len(lines) > 1 {
					enhancedState.SystemInfo["Memory"] = lines[1]
				}
			}
		}
	}

	fmt.Println("✅ System information learned:")
	for key, value := range enhancedState.SystemInfo {
		fmt.Printf("  • %s: %s\n", key, value)
	}

	// Add to conversation context
	systemContext := "I've learned about this system:\n"
	for key, value := range enhancedState.SystemInfo {
		systemContext += fmt.Sprintf("- %s: %s\n", key, value)
	}

	enhancedState.ConversationHistory = append(enhancedState.ConversationHistory,
		map[string]string{"role": "system", "content": systemContext})
}

func clearEnhancedScreen() {
	fmt.Print("\033[H\033[2J")
}

func showEnhancedLogo() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║   🚀 NEXUS LLM - ENHANCED ULTIMATE MODE 2025 🚀              ║
║                                                               ║
║     ███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗            ║
║     ████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝            ║
║     ██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗            ║
║     ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║            ║
║     ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║            ║
║     ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝            ║
║                                                               ║
║   💎 CHAT LOOP | AUTOMATION | SYSTEM CONTROL 💎             ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Printf("\n📡 Provider: %s | 🤖 Model: %s\n",
		enhancedState.CurrentProvider, getModelName(enhancedState.CurrentModel))

	if apiKey, ok := enhancedState.APIKeys[enhancedState.CurrentProvider]; ok && apiKey != "" {
		prefixLen := 6
		if len(apiKey) < 10 {
			prefixLen = len(apiKey) / 2
		}
		fmt.Printf("🔑 API Key: %s...%s ✅\n", apiKey[:prefixLen], apiKey[len(apiKey)-4:])
	} else {
		fmt.Println("🔑 API Key: ❌ Not configured")
	}
}

func showEnhancedMenu() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    🌟 ENHANCED MENU 🌟                       ║
╚═══════════════════════════════════════════════════════════════╝

  [1] 💬 Enhanced Chat     - Continuous chat with automation
  [2] 🤖 Model Selection   - Choose from 326+ models
  [3] 🔑 API Keys          - Configure OpenRouter
  [4] ⚙️  Settings          - Temperature, tokens, streaming
  [5] 📝 System Prompt     - Customize AI behavior
  [6] 🧪 Test API          - Verify connection
  [7] 🧠 System Info       - View learned system data
  [8] ⚡ Quick Setup       - Fast configuration
  [9] ❓ Help              - Usage instructions
  [0] 🚪 Exit              - Save and quit`)
}

// Stub functions for other menu items (implement as needed)
func handleEnhancedModelSelection(scanner *bufio.Scanner) {
	// Use existing model selection logic
	handleRealModelSelection(scanner)
}

func handleEnhancedAPIKeys(scanner *bufio.Scanner) {
	handleRealAPIKeys(scanner)
}

func handleEnhancedSettings(scanner *bufio.Scanner) {
	handleRealSettings(scanner)
}

func handleEnhancedSystemPrompt(scanner *bufio.Scanner) {
	handleRealSystemPrompt(scanner)
}

func handleEnhancedTestAPI(scanner *bufio.Scanner) {
	handleTestAPI(scanner)
}

func handleEnhancedSystemInfo() {
	clearEnhancedScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    🧠 SYSTEM INFORMATION                     ║
╚═══════════════════════════════════════════════════════════════╝`)

	if len(enhancedState.SystemInfo) == 0 {
		fmt.Println("\n⚠️ No system information learned yet.")
		fmt.Println("Use '/learn' command in chat to analyze this system.")
	} else {
		fmt.Println("\n📊 Learned System Data:")
		fmt.Println(strings.Repeat("─", 65))
		for key, value := range enhancedState.SystemInfo {
			fmt.Printf("%-20s: %s\n", key, value)
		}
	}

	fmt.Println("\n💡 This information helps the AI understand your system better")
	fmt.Println("   and provide more accurate automation suggestions.")
}

func handleEnhancedQuickSetup(scanner *bufio.Scanner) {
	handleQuickSetup(scanner)
}

func handleEnhancedHelp() {
	clearEnhancedScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    ❓ ENHANCED HELP                          ║
╚═══════════════════════════════════════════════════════════════╝

🚀 ENHANCED FEATURES:

1. CONTINUOUS CHAT LOOP
   - Chat continues until you type 'exit'
   - Maintains conversation history
   - No auto-exit after responses

2. SYSTEM COMMAND EXECUTION
   - Type '/cmd <command>' to run system commands
   - AI can suggest commands with 'EXECUTE:' prefix
   - Toggle auto-execution with '/auto'

3. SYSTEM LEARNING
   - Type '/learn' to analyze your system
   - AI remembers system configuration
   - Better automation suggestions

4. CONVERSATION MANAGEMENT
   - Type 'clear' to reset conversation
   - History is maintained across messages
   - Context-aware responses

💡 TIPS:
   • Enable command mode for automation tasks
   • Use '/learn' first for better system-specific help
   • AI can write scripts and execute them
   • Always review commands before execution

🔒 SAFETY:
   • Commands require confirmation
   • Review all AI-suggested commands
   • Use with caution on production systems`)
}

func loadEnhancedKeys() {
	// Load from environment
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		enhancedState.APIKeys["openrouter"] = key
	}

	// Load from config file
	homeDir, _ := os.UserHomeDir()
	configPath := homeDir + "/.gollm/api_keys.json"
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &enhancedState.APIKeys)
	}
}

func saveEnhancedKeys() {
	homeDir, _ := os.UserHomeDir()
	configDir := homeDir + "/.gollm"
	_ = os.MkdirAll(configDir, 0700)

	configPath := configDir + "/api_keys.json"
	if data, err := json.MarshalIndent(enhancedState.APIKeys, "", "  "); err == nil {
		_ = os.WriteFile(configPath, data, 0600)
	}
}

func fetchOpenRouterModelsEnhanced() error {
	// Reuse existing fetch logic
	return fetchOpenRouterModels()
}
