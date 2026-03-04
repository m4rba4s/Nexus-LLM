package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// NewChatLoopFixCommand creates a working chat loop command
func NewChatLoopFixCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "chat-loop",
		Short: "💬 Fixed continuous chat with AI (no auto-exit)",
		Long:  `Start a continuous chat session that doesn't exit after each response`,
		RunE:  runChatLoopFix,
	}
}

func runChatLoopFix(cmd *cobra.Command, args []string) error {
	// Load API key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ Error: OPENROUTER_API_KEY environment variable not set")
		fmt.Println("Please set it: export OPENROUTER_API_KEY='your-key-here'")
		return nil
	}

	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║           💬 CONTINUOUS CHAT MODE - NO AUTO EXIT!            ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Println("\n📡 Using OpenRouter API")
	fmt.Printf("🔑 API Key: %s\n", safeMaskKey(apiKey))
	fmt.Println("\n💡 Commands:")
	fmt.Println("  • Type 'exit' or 'quit' to leave chat")
	fmt.Println("  • Type 'clear' to clear conversation history")
	fmt.Println("\n" + strings.Repeat("─", 65))

	scanner := bufio.NewScanner(os.Stdin)
	conversationHistory := []map[string]string{}

	// IMPORTANT: This is the chat loop that keeps going!
	for {
		fmt.Print("\n💭 You> ")

		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())

		// Check for exit commands
		if message == "exit" || message == "quit" || message == "q" {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		if message == "clear" {
			conversationHistory = []map[string]string{}
			fmt.Println("🗑️ Conversation cleared")
			continue
		}

		if message == "" {
			continue
		}

		// Add to history
		conversationHistory = append(conversationHistory,
			map[string]string{"role": "user", "content": message})

		// Call API
		fmt.Println("\n🤖 AI> ")
		fmt.Println(strings.Repeat("─", 65))

		response, err := callChatAPI(apiKey, conversationHistory)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Println(response)
			// Add response to history
			conversationHistory = append(conversationHistory,
				map[string]string{"role": "assistant", "content": response})
		}

		fmt.Println(strings.Repeat("─", 65))
		// IMPORTANT: Loop continues here! No exit!
	}

	return nil
}

func safeMaskKey(k string) string {
	// Minimal, robust masking for any key length
	n := len(k)
	if n == 0 {
		return "(empty)"
	}
	if n <= 4 {
		return strings.Repeat("*", n)
	}
	pre := 6
	suf := 4
	if n < pre+suf+1 {
		pre = max(2, n/2)
		suf = n - pre
	}
	return fmt.Sprintf("%s...%s", k[:pre], k[n-suf:])
}

func callChatAPI(apiKey string, history []map[string]string) (string, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	// Build messages
	messages := []map[string]string{
		{"role": "system", "content": "You are a helpful AI assistant."},
	}

	// Add recent history (last 10 messages)
	start := 0
	if len(history) > 10 {
		start = len(history) - 10
	}
	messages = append(messages, history[start:]...)

	requestBody := map[string]interface{}{
		"model":       "anthropic/claude-3.5-sonnet",
		"messages":    messages,
		"temperature": 0.7,
		"max_tokens":  4096,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/m4rba4s/Nexus-LLM")
	req.Header.Set("X-Title", "Nexus Chat Loop")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response")
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
