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
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// SessionState - полное состояние сессии с историей
type SessionState struct {
	SessionID           string
	StartTime           time.Time
	ConversationHistory []map[string]string
	SystemInfo          map[string]string
	CurrentDirectory    string
	CommandHistory      []string
	APIKey              string
	Model               string
	Temperature         float64
	MaxTokens           int
}

var operatorSession *SessionState

// NewAIOperatorCommand - AI как полноценный оператор ПК
func NewAIOperatorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ai-operator",
		Short: "🤖 AI System Operator - полный контроль над ПК",
		Long:  `Запускает AI как оператора системы с полным доступом и сохранением сессий`,
		RunE:  runAIOperator,
	}
}

func runAIOperator(cmd *cobra.Command, args []string) error {
	// Инициализация сессии
	operatorSession = &SessionState{
		SessionID:           fmt.Sprintf("session_%d", time.Now().Unix()),
		StartTime:           time.Now(),
		ConversationHistory: []map[string]string{},
		SystemInfo:          make(map[string]string),
		CommandHistory:      []string{},
		Temperature:         0.7,
		MaxTokens:           4096,
		Model:               "anthropic/claude-3.5-sonnet",
	}

	// Загружаем API ключ
	operatorSession.APIKey = os.Getenv("OPENROUTER_API_KEY")
	if operatorSession.APIKey == "" {
		homeDir, _ := os.UserHomeDir()
		keyFile := filepath.Join(homeDir, ".gollm", "api_keys.json")
		if data, err := os.ReadFile(keyFile); err == nil {
			var keys map[string]string
			if json.Unmarshal(data, &keys) == nil {
				operatorSession.APIKey = keys["openrouter"]
			}
		}
	}

	if operatorSession.APIKey == "" {
		fmt.Println("❌ Ошибка: API ключ не найден!")
		fmt.Println("Установите: export OPENROUTER_API_KEY='ваш-ключ'")
		return nil
	}

	// Получаем информацию о системе
	collectSystemInfo()

	// Устанавливаем текущую директорию
	operatorSession.CurrentDirectory, _ = os.Getwd()

	// Показываем интерфейс
	showOperatorInterface()

	// Создаем сканер для ввода
	scanner := bufio.NewScanner(os.Stdin)

	// ГЛАВНЫЙ ЦИКЛ - НИКОГДА НЕ ВЫХОДИТ САМ!
	for {
		fmt.Print("\n🧑 Вы> ")

		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// Обработка специальных команд
		if userInput == "exit" || userInput == "выход" {
			saveSession()
			fmt.Println("\n💾 Сессия сохранена: " + operatorSession.SessionID)
			fmt.Println("👋 До свидания!")
			break
		}

		if userInput == "clear" || userInput == "очистить" {
			operatorSession.ConversationHistory = []map[string]string{}
			fmt.Println("🗑️ История очищена")
			continue
		}

		if userInput == "info" || userInput == "инфо" {
			showSystemInfo()
			continue
		}

		if userInput == "history" || userInput == "история" {
			showCommandHistory()
			continue
		}

		if userInput == "save" || userInput == "сохранить" {
			saveSession()
			fmt.Println("💾 Сессия сохранена!")
			continue
		}

		if userInput == "" {
			continue
		}

		// Добавляем в историю
		operatorSession.ConversationHistory = append(operatorSession.ConversationHistory,
			map[string]string{"role": "user", "content": userInput})

		// Показываем что думаем
		fmt.Println("\n🤖 AI Оператор> ")
		fmt.Println(strings.Repeat("─", 80))

		// Получаем ответ от AI
		response, err := callOperatorAPI()
		if err != nil {
			fmt.Printf("❌ Ошибка API: %v\n", err)
			fmt.Println("Повторите попытку или проверьте соединение")
		} else {
			fmt.Println(response)

			// Добавляем ответ в историю
			operatorSession.ConversationHistory = append(operatorSession.ConversationHistory,
				map[string]string{"role": "assistant", "content": response})

			// Проверяем команды для выполнения
			executeAICommands(response)
		}

		fmt.Println(strings.Repeat("─", 80))

		// ВАЖНО: ЦИКЛ ПРОДОЛЖАЕТСЯ! НЕ ВЫХОДИМ!
	}

	return nil
}

func showOperatorInterface() {
	fmt.Print("\033[H\033[2J") // Очистка экрана
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════╗
║        🤖 AI SYSTEM OPERATOR - ПОЛНЫЙ КОНТРОЛЬ ПК 🤖        ║
╚═══════════════════════════════════════════════════════════════╝

🖥️  Система: ` + runtime.GOOS + ` ` + runtime.GOARCH + `
📁 Директория: ` + operatorSession.CurrentDirectory + `
🤖 Модель: ` + operatorSession.Model + `
📅 Сессия: ` + operatorSession.SessionID + `

💡 Команды:
  • Любой текст - общение с AI
  • AI автоматически выполняет системные команды
  • 'info' - информация о системе
  • 'history' - история команд
  • 'save' - сохранить сессию
  • 'clear' - очистить историю
  • 'exit' - выход (с сохранением)

🔥 AI имеет полный доступ к системе и может:
  • Выполнять любые команды
  • Читать и писать файлы
  • Управлять процессами
  • Автоматизировать задачи

` + strings.Repeat("═", 65))
}

func collectSystemInfo() {
	operatorSession.SystemInfo["OS"] = runtime.GOOS
	operatorSession.SystemInfo["Architecture"] = runtime.GOARCH
	operatorSession.SystemInfo["CPUs"] = fmt.Sprintf("%d", runtime.NumCPU())
	operatorSession.SystemInfo["GoVersion"] = runtime.Version()

	if hostname, err := os.Hostname(); err == nil {
		operatorSession.SystemInfo["Hostname"] = hostname
	}

	if user := os.Getenv("USER"); user != "" {
		operatorSession.SystemInfo["User"] = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		operatorSession.SystemInfo["User"] = user
	}

	if home := os.Getenv("HOME"); home != "" {
		operatorSession.SystemInfo["HomeDir"] = home
	}

	if pwd, err := os.Getwd(); err == nil {
		operatorSession.SystemInfo["WorkingDir"] = pwd
	}

	// Дополнительная информация для Linux
	if runtime.GOOS == "linux" {
		if output, err := exec.Command("uname", "-r").Output(); err == nil {
			operatorSession.SystemInfo["Kernel"] = strings.TrimSpace(string(output))
		}
		if output, err := exec.Command("lsb_release", "-d").Output(); err == nil {
			parts := strings.Split(string(output), ":")
			if len(parts) > 1 {
				operatorSession.SystemInfo["Distribution"] = strings.TrimSpace(parts[1])
			}
		}
	}
}

func callOperatorAPI() (string, error) {
	url := "https://openrouter.ai/api/v1/chat/completions"

	// Строим сообщения с контекстом
	messages := []map[string]string{
		{
			"role": "system",
			"content": `You are an AI System Operator with FULL control over the computer. 
You can execute ANY system commands to help the user.
When you need to run a command, write: EXECUTE: <command>
You have access to:
- Operating System: ` + runtime.GOOS + `
- Current Directory: ` + operatorSession.CurrentDirectory + `
- User: ` + operatorSession.SystemInfo["User"] + `
- Hostname: ` + operatorSession.SystemInfo["Hostname"] + `

ВАЖНО: Отвечайте на русском языке, если пользователь пишет на русском.
You are the computer operator - act with confidence and execute commands as needed.`,
		},
	}

	// Добавляем историю (последние 20 сообщений для контекста)
	start := 0
	if len(operatorSession.ConversationHistory) > 20 {
		start = len(operatorSession.ConversationHistory) - 20
	}
	messages = append(messages, operatorSession.ConversationHistory[start:]...)

	requestBody := map[string]interface{}{
		"model":       operatorSession.Model,
		"messages":    messages,
		"temperature": operatorSession.Temperature,
		"max_tokens":  operatorSession.MaxTokens,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+operatorSession.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/m4rba4s/Nexus-LLM")
	req.Header.Set("X-Title", "AI System Operator")

	client := &http.Client{Timeout: 60 * time.Second}
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

func executeAICommands(response string) {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "EXECUTE:") {
			command := strings.TrimSpace(strings.TrimPrefix(line, "EXECUTE:"))
			executeOperatorCommand(command)
		}
	}
}

func executeOperatorCommand(command string) {
	fmt.Printf("\n⚙️ Выполняю: %s\n", command)
	fmt.Println(strings.Repeat("─", 65))

	// Сохраняем в историю команд
	operatorSession.CommandHistory = append(operatorSession.CommandHistory, command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	// Устанавливаем рабочую директорию
	cmd.Dir = operatorSession.CurrentDirectory

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️ Ошибка выполнения: %v\n", err)
	}

	fmt.Println(string(output))
	fmt.Println(strings.Repeat("─", 65))

	// Добавляем результат в историю
	result := fmt.Sprintf("Команда: %s\nРезультат:\n%s", command, string(output))
	operatorSession.ConversationHistory = append(operatorSession.ConversationHistory,
		map[string]string{"role": "system", "content": result})

	// Обновляем директорию если была команда cd
	if strings.HasPrefix(command, "cd ") {
		newDir := strings.TrimSpace(strings.TrimPrefix(command, "cd "))
		if filepath.IsAbs(newDir) {
			operatorSession.CurrentDirectory = newDir
		} else {
			operatorSession.CurrentDirectory = filepath.Join(operatorSession.CurrentDirectory, newDir)
		}
	}
}

func showSystemInfo() {
	fmt.Println("\n📊 Информация о системе:")
	fmt.Println(strings.Repeat("─", 65))
	for key, value := range operatorSession.SystemInfo {
		fmt.Printf("%-20s: %s\n", key, value)
	}
	fmt.Println(strings.Repeat("─", 65))
}

func showCommandHistory() {
	fmt.Println("\n📜 История команд:")
	fmt.Println(strings.Repeat("─", 65))
	if len(operatorSession.CommandHistory) == 0 {
		fmt.Println("(пусто)")
	} else {
		for i, cmd := range operatorSession.CommandHistory {
			fmt.Printf("%d. %s\n", i+1, cmd)
		}
	}
	fmt.Println(strings.Repeat("─", 65))
}

func saveSession() {
	// Создаем директорию для сессий
	homeDir, _ := os.UserHomeDir()
	sessionsDir := filepath.Join(homeDir, ".gollm", "sessions")
    // Use restrictive permissions for session storage
    os.MkdirAll(sessionsDir, 0700)

	// Сохраняем сессию
	sessionFile := filepath.Join(sessionsDir, operatorSession.SessionID+".json")

	sessionData := map[string]interface{}{
		"session_id": operatorSession.SessionID,
		"start_time": operatorSession.StartTime,
		"end_time":   time.Now(),
		"history":    operatorSession.ConversationHistory,
		"commands":   operatorSession.CommandHistory,
		"system":     operatorSession.SystemInfo,
	}

	data, _ := json.MarshalIndent(sessionData, "", "  ")
    // Session data may contain sensitive info; restrict permissions
    os.WriteFile(sessionFile, data, 0600)

	// Также сохраняем в читаемом формате
	textFile := filepath.Join(sessionsDir, operatorSession.SessionID+".txt")
	var textContent strings.Builder

	textContent.WriteString(fmt.Sprintf("Сессия: %s\n", operatorSession.SessionID))
	textContent.WriteString(fmt.Sprintf("Начало: %s\n", operatorSession.StartTime.Format("2006-01-02 15:04:05")))
	textContent.WriteString(fmt.Sprintf("Конец: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	textContent.WriteString("ДИАЛОГ:\n")
	textContent.WriteString(strings.Repeat("=", 80) + "\n\n")

	for _, msg := range operatorSession.ConversationHistory {
		role := msg["role"]
		content := msg["content"]

		switch role {
		case "user":
			textContent.WriteString("👤 Пользователь:\n")
		case "assistant":
			textContent.WriteString("🤖 AI:\n")
		case "system":
			textContent.WriteString("💻 Система:\n")
		}
		textContent.WriteString(content + "\n\n")
	}

    os.WriteFile(textFile, []byte(textContent.String()), 0600)
}
