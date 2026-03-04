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

// Model represents an AI model
type Model struct {
	ID          string
	Name        string
	Provider    string
	Context     string
	Price       string
	Description string
}

// GlobalState - хранит состояние приложения
type GlobalState struct {
	CurrentProvider string
	CurrentModel    string
	APIKeys         map[string]string
	Models          []Model
	SystemPrompt    string
	Temperature     float64
	MaxTokens       int
	StreamMode      bool
}

var state = &GlobalState{
	CurrentProvider: "openrouter",
	CurrentModel:    "anthropic/claude-3.5-sonnet",
	APIKeys:         make(map[string]string),
	SystemPrompt:    "You are a helpful AI assistant.",
	Temperature:     0.7,
	MaxTokens:       4096,
	StreamMode:      true,
	Models:          getLatestModels(),
}

// NewUltimateRealCommand creates the REAL working ultimate command
func NewUltimateRealCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ultimate-real",
		Short: "🔥 REAL Ultimate Mode with actual API integration",
		Long:  `Launch the REAL Ultimate AI Interface with working API calls and latest models!`,
		RunE:  runRealUltimate,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}

func getLatestModels() []Model {
	return []Model{
		// OpenRouter Models (ALL LATEST 2024-2025)
		{ID: "openai/gpt-4o", Name: "GPT-4o", Provider: "openrouter", Context: "128K", Price: "$5/1M", Description: "🔥 Latest GPT-4 Omni model"},
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openrouter", Context: "128K", Price: "$0.15/1M", Description: "⚡ Fast and cheap GPT-4"},
		{ID: "openai/o1-preview", Name: "OpenAI o1 Preview", Provider: "openrouter", Context: "128K", Price: "$15/1M", Description: "🧠 Reasoning model"},
		{ID: "openai/o1-mini", Name: "OpenAI o1 Mini", Provider: "openrouter", Context: "128K", Price: "$3/1M", Description: "🎯 Smaller reasoning model"},

		{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Provider: "openrouter", Context: "200K", Price: "$3/1M", Description: "🏆 Best Claude model"},
		{ID: "anthropic/claude-3-opus", Name: "Claude 3 Opus", Provider: "openrouter", Context: "200K", Price: "$15/1M", Description: "💎 Most powerful Claude"},
		{ID: "anthropic/claude-3-haiku", Name: "Claude 3 Haiku", Provider: "openrouter", Context: "200K", Price: "$0.25/1M", Description: "⚡ Fast Claude"},

		{ID: "google/gemini-pro-1.5", Name: "Gemini Pro 1.5", Provider: "openrouter", Context: "2M", Price: "$2.5/1M", Description: "🌟 2M context window!"},
		{ID: "google/gemini-flash-1.5", Name: "Gemini Flash 1.5", Provider: "openrouter", Context: "1M", Price: "$0.25/1M", Description: "⚡ Fast Gemini"},

		{ID: "meta-llama/llama-3.1-405b-instruct", Name: "Llama 3.1 405B", Provider: "openrouter", Context: "128K", Price: "$3/1M", Description: "🦙 Largest Llama"},
		{ID: "meta-llama/llama-3.1-70b-instruct", Name: "Llama 3.1 70B", Provider: "openrouter", Context: "128K", Price: "$0.7/1M", Description: "🦙 Powerful Llama"},

		{ID: "deepseek/deepseek-chat", Name: "DeepSeek V3", Provider: "openrouter", Context: "64K", Price: "$0.14/1M", Description: "🚀 Latest DeepSeek"},
		{ID: "deepseek/deepseek-coder", Name: "DeepSeek Coder V2", Provider: "openrouter", Context: "128K", Price: "$0.14/1M", Description: "💻 Best for coding"},

		{ID: "qwen/qwen-2.5-72b-instruct", Name: "Qwen 2.5 72B", Provider: "openrouter", Context: "128K", Price: "$0.35/1M", Description: "🇨🇳 Latest Qwen"},
		{ID: "qwen/qwen-2.5-coder-32b", Name: "Qwen 2.5 Coder", Provider: "openrouter", Context: "128K", Price: "$0.18/1M", Description: "💻 Qwen for code"},

		{ID: "moonshot/kimi-1.5-long", Name: "Kimi 1.5 Long", Provider: "openrouter", Context: "256K", Price: "$0.3/1M", Description: "🌙 Long context Kimi"},

		{ID: "01-ai/yi-large", Name: "Yi Large", Provider: "openrouter", Context: "200K", Price: "$3/1M", Description: "🎯 Yi-Large model"},

		{ID: "zhipu/glm-4-plus", Name: "GLM-4 Plus", Provider: "openrouter", Context: "128K", Price: "$0.5/1M", Description: "🇨🇳 Latest GLM"},

		{ID: "mistralai/mistral-large", Name: "Mistral Large", Provider: "openrouter", Context: "128K", Price: "$2/1M", Description: "🇫🇷 Mistral flagship"},
		{ID: "mistralai/mixtral-8x22b", Name: "Mixtral 8x22B", Provider: "openrouter", Context: "64K", Price: "$0.9/1M", Description: "🔀 MoE model"},

		{ID: "cohere/command-r-plus", Name: "Command R+", Provider: "openrouter", Context: "128K", Price: "$2.5/1M", Description: "📚 RAG optimized"},

		{ID: "perplexity/llama-3.1-sonar-huge", Name: "Perplexity Sonar", Provider: "openrouter", Context: "128K", Price: "$5/1M", Description: "🔍 Web search model"},

		// Direct API models
		{ID: "gpt-4-turbo-preview", Name: "GPT-4 Turbo", Provider: "openai", Context: "128K", Price: "$10/1M", Description: "Direct OpenAI"},
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "anthropic", Context: "200K", Price: "$15/1M", Description: "Direct Anthropic"},
	}
}

func runRealUltimate(cmd *cobra.Command, args []string) error {
	clearRealScreen()
	// Load saved keys and env var
	loadSavedKeys()
	// Try to refresh models at startup (best-effort)
	_ = fetchOpenRouterModels()
	showRealLogo()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		showRealMainMenu()
		fmt.Print("\n🎯 Выберите опцию (0-9): ")

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "0", "q", "exit":
			fmt.Println("\n👋 До свидания! Спасибо за использование REAL Ultimate!")
			saveKeys()
			return nil

		case "1":
			handleRealChat(scanner)

		case "2":
			handleRealModelSelection(scanner)

		case "3":
			handleRealAPIKeys(scanner)

		case "4":
			handleRealSettings(scanner)

		case "5":
			handleRealSystemPrompt(scanner)

		case "6":
			handleTestAPI(scanner)

		case "7":
			handleModelInfo()

		case "8":
			handleQuickSetup(scanner)

		case "9":
			handleRealHelp()

		default:
			fmt.Println("\n❌ Неверный выбор. Попробуйте снова.")
		}

		if choice != "1" { // Не ждём Enter после чата
			fmt.Println("\n📌 Нажмите Enter для продолжения...")
			scanner.Scan()
		}
		clearRealScreen()
	}

	return nil
}

func clearRealScreen() {
	fmt.Print("\033[H\033[2J")
}

func showRealLogo() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║   🔥 NEXUS LLM - ULTIMATE REAL MODE 2025 🔥                 ║
║                                                               ║
║     ███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗            ║
║     ████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝            ║
║     ██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗            ║
║     ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║            ║
║     ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║            ║
║     ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝            ║
║                                                               ║
║   💎 LATEST MODELS | REAL API | WORKING CHAT 💎             ║
╚═══════════════════════════════════════════════════════════════╝`)

	// Показываем текущий статус
	fmt.Printf("\n📡 Provider: %s | 🤖 Model: %s\n", state.CurrentProvider, getModelName(state.CurrentModel))
	if apiKey, ok := state.APIKeys[state.CurrentProvider]; ok && apiKey != "" {
		prefixLen := 6
		if len(apiKey) < 10 {
			prefixLen = len(apiKey) / 2
		}
		fmt.Printf("🔑 API Key: %s...%s ✅\n", apiKey[:prefixLen], apiKey[len(apiKey)-4:])
	} else {
		fmt.Println("🔑 API Key: ❌ Not configured")
	}
}

func showRealMainMenu() {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    🌟 ГЛАВНОЕ МЕНЮ 🌟                        ║
╚═══════════════════════════════════════════════════════════════╝

  [1] 💬 REAL Chat         - Настоящий чат с выбранной моделью
  [2] 🤖 Выбор модели      - 25+ последних моделей 2024-2025
  [3] 🔑 API ключи         - Настройка OpenRouter и других
  [4] ⚙️  Настройки         - Temperature, tokens, streaming
  [5] 📝 System Prompt     - Настроить поведение AI
  [6] 🧪 Test API          - Проверить работу API
  [7] 📊 Model Info        - Информация о всех моделях
  [8] ⚡ Quick Setup       - Быстрая настройка OpenRouter
  [9] ❓ Помощь            - Как использовать
  [0] 🚪 Выход             - Сохранить и выйти`)
}

func handleRealChat(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                  💬 REAL AI CHAT                             ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Printf("\n🤖 Model: %s\n", getModelName(state.CurrentModel))
	fmt.Printf("🌡️ Temperature: %.1f | 📊 Max Tokens: %d | ⚡ Stream: %v\n",
		state.Temperature, state.MaxTokens, state.StreamMode)
	fmt.Println("\n" + strings.Repeat("─", 65))

	// Проверяем API ключ
	apiKey := state.APIKeys[state.CurrentProvider]
	if apiKey == "" {
		fmt.Println("\n⚠️ WARNING: No API key configured!")
		fmt.Println("Press 3 to configure API key or continue for demo mode")
	}

	fmt.Print("\n💭 Your message (or 'exit' to quit):\n> ")

	if scanner.Scan() {
		message := scanner.Text()
		if message != "exit" && message != "" {
			fmt.Println("\n🤖 AI Response:")
			fmt.Println("─────────────────────────────────────────────────────────────")

			if apiKey != "" {
				// REAL API CALL!
				response, err := callRealAPI(message)
				if err != nil {
					fmt.Printf("❌ Error: %v\n", err)
					fmt.Println("💡 Tip: Check your API key and internet connection")
				} else {
					if state.StreamMode {
						// Simulate streaming
						for _, char := range response {
							fmt.Print(string(char))
							time.Sleep(10 * time.Millisecond)
						}
						fmt.Println()
					} else {
						fmt.Println(response)
					}
				}
			} else {
				// Demo mode
				fmt.Println("🎭 DEMO MODE: Configure API key for real responses")
				fmt.Println("This is a demo response. Add your OpenRouter API key to get real AI responses!")
			}

			fmt.Println("─────────────────────────────────────────────────────────────")
		}
	}
}

func callRealAPI(message string) (string, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	requestBody := map[string]interface{}{
		"model":       state.CurrentModel,
		"messages":    []map[string]string{{"role": "system", "content": state.SystemPrompt}, {"role": "user", "content": message}},
		"temperature": state.Temperature,
		"max_tokens":  state.MaxTokens,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	apiKey := state.APIKeys[state.CurrentProvider]
	if apiKey == "" {
		return "", fmt.Errorf("missing API key for provider %s", state.CurrentProvider)
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HTTP-Referer", "https://github.com/m4rba4s/Nexus-LLM")
		req.Header.Set("X-Title", "Nexus LLM Ultimate")

		client := &http.Client{Timeout: 45 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Try to parse error
			var eresp map[string]interface{}
			if json.Unmarshal(body, &eresp) == nil {
				if e, ok := eresp["error"]; ok {
					if em, ok := e.(map[string]interface{}); ok {
						lastErr = fmt.Errorf("HTTP %d: %v", resp.StatusCode, em["message"])
						// Retry on 429/5xx
						if resp.StatusCode == 429 || resp.StatusCode >= 500 {
							time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
							continue
						}
						return "", lastErr
					}
				}
			}
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				time.Sleep(time.Duration(attempt) * 700 * time.Millisecond)
				continue
			}
			return "", lastErr
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %s", string(body))
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
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
		lastErr = fmt.Errorf("unexpected response format")
		// no retry on format error
		break
	}
	return "", lastErr
}

func handleRealModelSelection(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║              🤖 LATEST AI MODELS 2024-2025                   ║
╚═══════════════════════════════════════════════════════════════╝`)

	if err := fetchOpenRouterModels(); err != nil {
		fmt.Printf("\n⚠️ Could not refresh models: %v (using fallback)\n", err)
	}

	// group by provider prefix
	groups := map[string][]Model{}
	order := []string{"openai", "anthropic", "google", "moonshot", "zhipu", "qwen", "deepseek", "meta-llama", "mistralai", "01-ai", "perplexity", "cohere"}
	for _, m := range state.Models {
		parts := strings.SplitN(m.ID, "/", 2)
		prefix := parts[0]
		groups[prefix] = append(groups[prefix], m)
	}

	modelIndex := 1
	modelMap := map[int]Model{}
	for _, prefix := range order {
		ms := groups[prefix]
		if len(ms) == 0 {
			continue
		}
		fmt.Printf("\n[%s]\n", strings.ToUpper(prefix))
		fmt.Println(strings.Repeat("─", 65))
		for _, model := range ms {
			selected := ""
			if model.ID == state.CurrentModel {
				selected = " ✅"
			}
			fmt.Printf("[%2d] %-30s %8s | %s%s\n", modelIndex, model.Name, model.Context, model.Price, selected)
			if model.Description != "" {
				fmt.Printf("     %s\n", model.Description)
			}
			modelMap[modelIndex] = model
			modelIndex++
		}
	}

	fmt.Print("\n🎯 Select model number (or 0 to cancel): ")
	if scanner.Scan() {
		var choice int
		fmt.Sscanf(scanner.Text(), "%d", &choice)
		if model, ok := modelMap[choice]; ok {
			state.CurrentModel = model.ID
			state.CurrentProvider = model.Provider
			fmt.Printf("\n✅ Selected: %s (%s)\n", model.Name, model.ID)
		}
	}
}

func handleRealAPIKeys(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    🔑 API KEY MANAGEMENT                     ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Println("\nCurrent API Keys:")
	fmt.Println(strings.Repeat("─", 65))

	providers := []string{"openrouter", "openai", "anthropic", "google"}
	for i, provider := range providers {
		status := "❌ Not configured"
		if key, ok := state.APIKeys[provider]; ok && key != "" {
			status = fmt.Sprintf("✅ %s...%s", key[:min(10, len(key))], key[max(0, len(key)-4):])
		}
		fmt.Printf("[%d] %-12s: %s\n", i+1, strings.Title(provider), status)
	}

	fmt.Println("\n📝 OpenRouter supports ALL models with one API key!")
	fmt.Println("   Get your key at: https://openrouter.ai/keys")

	fmt.Print("\n🔐 Select provider to configure (1-4) or 0 to cancel: ")
	if scanner.Scan() {
		var choice int
		fmt.Sscanf(scanner.Text(), "%d", &choice)
		if choice > 0 && choice <= len(providers) {
			provider := providers[choice-1]
			fmt.Printf("\n📝 Enter API key for %s: ", provider)
			if scanner.Scan() {
				key := strings.TrimSpace(scanner.Text())
				if key != "" {
					state.APIKeys[provider] = key
					fmt.Println("✅ API key saved!")

					// Automatically switch to this provider
					state.CurrentProvider = provider
					fmt.Printf("🔄 Switched to %s provider\n", provider)
				}
			}
		}
	}
}

func handleRealSettings(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    ⚙️  SETTINGS                               ║
╚═══════════════════════════════════════════════════════════════╝`)

	for {
		fmt.Printf(`
Current Settings:
─────────────────────────────────────────────────────────────────
[1] 🌡️  Temperature:     %.1f  (0.0-2.0, current: %.1f)
[2] 📊 Max Tokens:      %d   (1-100000, current: %d)
[3] ⚡ Stream Mode:     %v
[4] 🎯 Provider:        %s
[0] ← Back to menu

`, state.Temperature, state.Temperature, state.MaxTokens, state.MaxTokens,
			state.StreamMode, state.CurrentProvider)

		fmt.Print("Select setting to change: ")
		if !scanner.Scan() {
			break
		}

		choice := scanner.Text()
		switch choice {
		case "1":
			fmt.Print("Enter temperature (0.0-2.0): ")
			if scanner.Scan() {
				var temp float64
				fmt.Sscanf(scanner.Text(), "%f", &temp)
				if temp >= 0 && temp <= 2 {
					state.Temperature = temp
					fmt.Println("✅ Temperature updated!")
				}
			}
		case "2":
			fmt.Print("Enter max tokens: ")
			if scanner.Scan() {
				var tokens int
				fmt.Sscanf(scanner.Text(), "%d", &tokens)
				if tokens > 0 && tokens <= 100000 {
					state.MaxTokens = tokens
					fmt.Println("✅ Max tokens updated!")
				}
			}
		case "3":
			state.StreamMode = !state.StreamMode
			fmt.Printf("✅ Stream mode: %v\n", state.StreamMode)
		case "0":
			return
		}
	}
}

func handleRealSystemPrompt(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    📝 SYSTEM PROMPT                          ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Println("\nCurrent prompt:")
	fmt.Println(strings.Repeat("─", 65))
	fmt.Println(state.SystemPrompt)
	fmt.Println(strings.Repeat("─", 65))

	prompts := []struct {
		name string
		text string
	}{
		{"Default Assistant", "You are a helpful AI assistant."},
		{"Expert Coder", "You are an expert programmer. Write clean, efficient code with comments."},
		{"Creative Writer", "You are a creative writer. Use rich language and vivid descriptions."},
		{"Data Analyst", "You are a data analyst. Provide insights and use examples with data."},
		{"Teacher", "You are a patient teacher. Explain concepts clearly step by step."},
	}

	fmt.Println("\nPreset prompts:")
	for i, p := range prompts {
		fmt.Printf("[%d] %s\n", i+1, p.name)
	}
	fmt.Println("[C] Custom prompt")
	fmt.Println("[0] Keep current")

	fmt.Print("\nSelect option: ")
	if scanner.Scan() {
		choice := scanner.Text()
		switch choice {
		case "1", "2", "3", "4", "5":
			idx := int(choice[0] - '1')
			if idx < len(prompts) {
				state.SystemPrompt = prompts[idx].text
				fmt.Println("✅ System prompt updated!")
			}
		case "c", "C":
			fmt.Println("Enter custom prompt (end with empty line):")
			var lines []string
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					break
				}
				lines = append(lines, line)
			}
			if len(lines) > 0 {
				state.SystemPrompt = strings.Join(lines, "\n")
				fmt.Println("✅ Custom prompt saved!")
			}
		}
	}
}

func handleTestAPI(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    🧪 API TEST                               ║
╚═══════════════════════════════════════════════════════════════╝`)

	apiKey := state.APIKeys[state.CurrentProvider]
	if apiKey == "" {
		fmt.Println("\n⚠️ No API key configured!")
		fmt.Println("Press 3 to configure API key first")
		return
	}

	fmt.Printf("\n🔍 Testing with:\n")
	fmt.Printf("   Provider: %s\n", state.CurrentProvider)
	fmt.Printf("   Model: %s\n", getModelName(state.CurrentModel))
	fmt.Printf("   API Key: %s...%s\n", apiKey[:10], apiKey[len(apiKey)-4:])

	fmt.Println("\n⏳ Sending test request...")

	response, err := callRealAPI("Hello! Please respond with 'API test successful!' if you receive this.")
	if err != nil {
		fmt.Printf("\n❌ Test FAILED: %v\n", err)
		fmt.Println("\n💡 Troubleshooting tips:")
		fmt.Println("   1. Check your API key is correct")
		fmt.Println("   2. Ensure you have credits/balance")
		fmt.Println("   3. Check internet connection")
		fmt.Println("   4. Try a different model")
	} else {
		fmt.Println("\n✅ Test SUCCESSFUL!")
		fmt.Printf("\n🤖 Response: %s\n", response)
	}
}

func handleModelInfo() {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    📊 MODEL INFORMATION                      ║
╚═══════════════════════════════════════════════════════════════╝`)

	// Try to refresh models from OpenRouter
	if err := fetchOpenRouterModels(); err != nil {
		fmt.Printf("\n⚠️ Could not refresh model list from OpenRouter: %v\n", err)
		fmt.Println("   Using built-in model list as fallback.")
	}

	fmt.Println("\n🌟 Available Models (via OpenRouter):")
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("%-40s %-10s %-12s %s\n", "Model", "Context", "Price/1M", "Description")
	fmt.Println(strings.Repeat("─", 80))

	for _, model := range state.Models {
		if model.Provider == "openrouter" {
			current := ""
			if model.ID == state.CurrentModel {
				current = " ←"
			}
			fmt.Printf("%-40s %-10s %-12s %s%s\n",
				model.Name, model.Context, model.Price, model.Description, current)
		}
	}

	fmt.Println("\n💡 Tips:")
	fmt.Println("   • All these models work with ONE OpenRouter API key")
	fmt.Println("   • Prices shown are per 1M tokens")
	fmt.Println("   • Larger context = can process more text")
	fmt.Println("   • o1 models = reasoning specialists")
}

func handleQuickSetup(scanner *bufio.Scanner) {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    ⚡ QUICK SETUP                            ║
╚═══════════════════════════════════════════════════════════════╝`)

	fmt.Println("\n🚀 Let's get you started in 30 seconds!")
	fmt.Println(strings.Repeat("─", 65))

	fmt.Println("\n1️⃣ OpenRouter Setup")
	fmt.Println("   Get your FREE API key at: https://openrouter.ai/keys")

	// Prefer environment variable if present
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		state.APIKeys["openrouter"] = key
		state.CurrentProvider = "openrouter"
		fmt.Println("   ✅ API key loaded from environment!")
	} else {
		fmt.Print("   Enter your OpenRouter API key: ")
		if scanner.Scan() {
			key := strings.TrimSpace(scanner.Text())
			if key != "" {
				state.APIKeys["openrouter"] = key
				state.CurrentProvider = "openrouter"
				fmt.Println("   ✅ API key saved!")
			}
		}
	}

	// Refresh model list
	if err := fetchOpenRouterModels(); err != nil {
		fmt.Printf("   ⚠️ Could not refresh model list: %v (using fallback)\n", err)
	}

	fmt.Println("\n2️⃣ Select your favorite model:")
	// pick some popular defaults if available
	popular := []string{
		"anthropic/claude-3.5-sonnet",
		"openai/gpt-4o",
		"google/gemini-flash-1.5",
		"zhipu/glm-4.5",
		"moonshot/kimi-k2",
	}
	candidates := []Model{}
	for _, pid := range popular {
		for _, m := range state.Models {
			if m.ID == pid {
				candidates = append(candidates, m)
				break
			}
		}
	}
	if len(candidates) == 0 {
		// fallback to a few from the list
		for _, m := range state.Models {
			if len(candidates) >= 4 {
				break
			}
			candidates = append(candidates, m)
		}
	}
	for i, m := range candidates {
		fmt.Printf("   [%d] %s\n", i+1, m.Name)
	}
	fmt.Printf("   [%d] Keep current (%s)\n", len(candidates)+1, getModelName(state.CurrentModel))

	fmt.Print("\n   Select: ")
	if scanner.Scan() {
		var choice int
		fmt.Sscanf(scanner.Text(), "%d", &choice)
		if choice >= 1 && choice <= len(candidates) {
			state.CurrentModel = candidates[choice-1].ID
			fmt.Printf("   ✅ Selected: %s\n", candidates[choice-1].Name)
		}
	}

	fmt.Println("\n3️⃣ Quick settings:")
	fmt.Println("   Temperature: 0.7 (balanced)")
	fmt.Println("   Max tokens: 4096")
	fmt.Println("   Stream mode: ON")
	state.Temperature = 0.7
	state.MaxTokens = 4096
	state.StreamMode = true

	fmt.Println("\n✨ Setup complete! Press 1 to start chatting!")
}

func handleRealHelp() {
	clearRealScreen()
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║                    ❓ HELP & GUIDE                           ║
╚═══════════════════════════════════════════════════════════════╝

🎯 QUICK START:
   1. Press [8] for Quick Setup
   2. Get your OpenRouter API key
   3. Press [1] to start chatting!

🔑 ABOUT OPENROUTER:
   • ONE API key for 25+ models
   • Pay per use (no subscription)
   • Free $1 credit on signup
   • Supports all major models

💰 PRICING GUIDE:
   Cheap:  Haiku, Flash, Mini (~$0.25/1M tokens)
   Medium: Sonnet, GPT-4o (~$3-5/1M tokens)  
   Premium: Opus, o1-preview (~$15/1M tokens)

🌟 RECOMMENDED MODELS:
   Best Overall: Claude 3.5 Sonnet
   Best Value: Gemini Flash 1.5
   Best Coding: DeepSeek Coder V2
   Best Open: Llama 3.1 70B

⌨️ HOTKEYS:
   1-9: Quick menu selection
   0/Q: Exit
   Enter: Confirm

🔗 LINKS:
   OpenRouter: https://openrouter.ai
   Get API Key: https://openrouter.ai/keys
   GitHub: https://github.com/m4rba4s/Nexus-LLM`)
}

// Fetch dynamic model list from OpenRouter
func fetchOpenRouterModels() error {
	apiURL := "https://openrouter.ai/api/v1/models"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	if k := state.APIKeys["openrouter"]; k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}
	req.Header.Set("HTTP-Referer", "https://github.com/m4rba4s/Nexus-LLM")
	req.Header.Set("X-Title", "Nexus LLM Ultimate")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("models list HTTP %d: %s", resp.StatusCode, string(b))
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return err
	}
	arr, _ := parsed["data"].([]interface{})
	if len(arr) == 0 {
		return fmt.Errorf("no models returned")
	}
	var newModels []Model
	for _, it := range arr {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		name, _ := m["name"].(string)
		ctx := ""
		if ml, ok := m["context_length"].(float64); ok {
			ctx = fmt.Sprintf("%dK", int(ml)/1000)
		}
		price := ""
		if pr, ok := m["pricing"].(map[string]interface{}); ok {
			// best-effort: try prompt.usd
			if p, ok := pr["prompt"]; ok {
				if pm, ok := p.(map[string]interface{}); ok {
					if usd, ok := pm["usd"].(float64); ok && usd > 0 {
						price = fmt.Sprintf("$%.2f/1M", usd*1_000_000)
					}
				}
			}
		}
		prov := "openrouter"
		desc := ""
		newModels = append(newModels, Model{ID: id, Name: name, Provider: prov, Context: ctx, Price: price, Description: desc})
	}
	// Merge preserving current selection
	state.Models = mergeModelLists(state.Models, newModels)
	return nil
}

func mergeModelLists(static []Model, dynamic []Model) []Model {
	seen := map[string]bool{}
	var merged []Model
	for _, m := range dynamic {
		merged = append(merged, m)
		seen[m.ID] = true
	}
	for _, m := range static {
		if !seen[m.ID] {
			merged = append(merged, m)
		}
	}
	return merged
}

// Helper functions

func getModelName(modelID string) string {
	for _, model := range state.Models {
		if model.ID == modelID {
			return model.Name
		}
	}
	return modelID
}

func loadSavedKeys() {
	// Try to load from environment or file
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		state.APIKeys["openrouter"] = key
	}

	homeDir, _ := os.UserHomeDir()
	configPath := homeDir + "/.gollm/api_keys.json"
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &state.APIKeys)
	}
}

func saveKeys() {
	homeDir, _ := os.UserHomeDir()
	configDir := homeDir + "/.gollm"
	_ = os.MkdirAll(configDir, 0700)
	configPath := configDir + "/api_keys.json"
	if data, err := json.MarshalIndent(state.APIKeys, "", "  "); err == nil {
		_ = os.WriteFile(configPath, data, 0600)
	}
}

// Using min/max from benchmark.go
