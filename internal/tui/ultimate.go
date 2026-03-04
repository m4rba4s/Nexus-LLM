package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screens
const (
	ScreenMain = iota
	ScreenChat
	ScreenSettings
	ScreenModels
	ScreenAPIKeys
	ScreenSystemPrompt
	ScreenRules
)

// UltimateModel - полноценный интерфейс с всеми функциями
type UltimateModel struct {
	// Screens
	currentScreen int

	// UI Components
	viewport  viewport.Model
	textarea  textarea.Model
	textInput textinput.Model
	modelList list.Model
	spinner   spinner.Model
	help      tea.Model

	// Chat
	messages  []ChatMessage
	streaming bool

	// Configuration
	config *Configuration

	// State
	width  int
	height int
	err    error
	status string

	// Providers and Models
	providers       []Provider
	currentProvider string
	currentModel    string

	// Settings inputs
	apiKeyInputs   map[string]textinput.Model
	settingsInputs map[string]textinput.Model

	// System Prompt and Rules
	systemPrompt string
	rules        []Rule

	// Focus management
	focusedInput int
}

type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
}

type Configuration struct {
	Providers       map[string]ProviderConfig `json:"providers"`
	CurrentProvider string                    `json:"current_provider"`
	CurrentModel    string                    `json:"current_model"`
	SystemPrompt    string                    `json:"system_prompt"`
	Rules           []Rule                    `json:"rules"`
	Theme           string                    `json:"theme"`
	AutoSave        bool                      `json:"auto_save"`
	StreamMode      bool                      `json:"stream_mode"`
	MaxTokens       int                       `json:"max_tokens"`
	Temperature     float64                   `json:"temperature"`
}

type ProviderConfig struct {
	Name          string            `json:"name"`
	APIKey        string            `json:"api_key"`
	BaseURL       string            `json:"base_url"`
	Models        []string          `json:"models"`
	Enabled       bool              `json:"enabled"`
	CustomHeaders map[string]string `json:"custom_headers"`
}

type Provider struct {
	Name        string
	DisplayName string
	Models      []AIModel
	RequiresKey bool
	BaseURL     string
}

type AIModel struct {
	ID          string
	Name        string
	Description string
	MaxTokens   int
	Available   bool
}

type Rule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Enabled     bool   `json:"enabled"`
}

// Styles
var (
	// Main color scheme
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#EC4899") // Pink
	accentColor    = lipgloss.Color("#06B6D4") // Cyan
	successColor   = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Orange
	errorColor     = lipgloss.Color("#EF4444") // Red
	bgColor        = lipgloss.Color("#111827") // Dark gray
	fgColor        = lipgloss.Color("#F3F4F6") // Light gray
	borderColor    = lipgloss.Color("#374151") // Medium gray

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(bgColor).
			Padding(1, 2).
			MarginBottom(1)

	menuStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1).
			MarginBottom(1)

	activeMenuItemStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Background(lipgloss.Color("#1F2937")).
				Bold(true).
				Padding(0, 2)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(fgColor).
			Padding(0, 2)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(fgColor).
			Padding(0, 1)

	chatMessageStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Padding(0, 1).
				MarginBottom(1)

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor).
			Background(bgColor).
			Align(lipgloss.Center).
			Padding(1, 0)
)

func NewUltimateModel() *UltimateModel {
	// Initialize text area for chat
	ta := textarea.New()
	ta.Placeholder = "Введите ваше сообщение..."
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Focus()

	// Initialize viewport for messages
	vp := viewport.New(80, 20)

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot

	// Initialize with defaults for now
	config := &Configuration{
		Providers:       make(map[string]ProviderConfig),
		CurrentProvider: "openai",
		CurrentModel:    "gpt-3.5-turbo",
		SystemPrompt:    "Ты полезный AI ассистент.",
		Rules:           []Rule{},
		Theme:           "modern",
		AutoSave:        true,
		StreamMode:      true,
		MaxTokens:       2048,
		Temperature:     0.7,
	}

	// Initialize providers
	providers := getAvailableProviders()

	// Initialize API key inputs
	apiKeyInputs := make(map[string]textinput.Model)
	for _, provider := range providers {
		ti := textinput.New()
		ti.Placeholder = fmt.Sprintf("Введите API ключ для %s", provider.DisplayName)
		ti.CharLimit = 200
		ti.Width = 60
		ti.EchoMode = textinput.EchoPassword
		apiKeyInputs[provider.Name] = ti
	}

	return &UltimateModel{
		currentScreen:   ScreenMain,
		viewport:        vp,
		textarea:        ta,
		spinner:         s,
		config:          config,
		providers:       providers,
		currentProvider: config.CurrentProvider,
		currentModel:    config.CurrentModel,
		apiKeyInputs:    apiKeyInputs,
		systemPrompt:    config.SystemPrompt,
		rules:           config.Rules,
		messages:        []ChatMessage{},
		status:          "Готов к работе",
	}
}

func loadConfiguration() *Configuration {
	configPath := filepath.Join(os.Getenv("HOME"), ".gollm", "ultimate_config.json")

	// Default configuration
	config := &Configuration{
		Providers:       make(map[string]ProviderConfig),
		CurrentProvider: "openai",
		CurrentModel:    "gpt-3.5-turbo",
		SystemPrompt:    "Ты полезный AI ассистент.",
		Rules:           []Rule{},
		Theme:           "modern",
		AutoSave:        true,
		StreamMode:      true,
		MaxTokens:       2048,
		Temperature:     0.7,
	}

	// Try to load existing config
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, config)
	}

	return config
}

func (m *UltimateModel) saveConfiguration() error {
	configPath := filepath.Join(os.Getenv("HOME"), ".gollm", "ultimate_config.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func getAvailableProviders() []Provider {
	return []Provider{
		{
			Name:        "openai",
			DisplayName: "OpenAI",
			RequiresKey: true,
			BaseURL:     "https://api.openai.com/v1",
			Models: []AIModel{
				{ID: "gpt-4-turbo-preview", Name: "GPT-4 Turbo", Description: "Самая мощная модель", MaxTokens: 128000},
				{ID: "gpt-4", Name: "GPT-4", Description: "Мощная модель для сложных задач", MaxTokens: 8192},
				{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Description: "Быстрая и эффективная", MaxTokens: 4096},
			},
		},
		{
			Name:        "anthropic",
			DisplayName: "Anthropic Claude",
			RequiresKey: true,
			BaseURL:     "https://api.anthropic.com/v1",
			Models: []AIModel{
				{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Description: "Самая мощная модель Claude", MaxTokens: 200000},
				{ID: "claude-3-sonnet-20240229", Name: "Claude 3 Sonnet", Description: "Баланс скорости и качества", MaxTokens: 200000},
				{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Description: "Быстрая модель", MaxTokens: 200000},
			},
		},
		{
			Name:        "ollama",
			DisplayName: "Ollama (Локальные модели)",
			RequiresKey: false,
			BaseURL:     "http://localhost:11434",
			Models: []AIModel{
				{ID: "llama2", Name: "Llama 2", Description: "Open source модель от Meta", MaxTokens: 4096},
				{ID: "mistral", Name: "Mistral", Description: "Эффективная open source модель", MaxTokens: 8192},
				{ID: "codellama", Name: "Code Llama", Description: "Специализированная для кода", MaxTokens: 4096},
			},
		},
		{
			Name:        "gemini",
			DisplayName: "Google Gemini",
			RequiresKey: true,
			BaseURL:     "https://generativelanguage.googleapis.com/v1",
			Models: []AIModel{
				{ID: "gemini-pro", Name: "Gemini Pro", Description: "Мультимодальная модель Google", MaxTokens: 32768},
				{ID: "gemini-pro-vision", Name: "Gemini Pro Vision", Description: "Модель с поддержкой изображений", MaxTokens: 32768},
			},
		},
	}
}

func (m UltimateModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		textarea.Blink,
	)
}

func (m UltimateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		m.textarea.SetWidth(msg.Width - 4)

	case tea.KeyMsg:
		switch m.currentScreen {
		case ScreenMain:
			return m.handleMainScreen(msg)
		case ScreenChat:
			return m.handleChatScreen(msg)
		case ScreenSettings:
			return m.handleSettingsScreen(msg)
		case ScreenModels:
			return m.handleModelsScreen(msg)
		case ScreenAPIKeys:
			return m.handleAPIKeysScreen(msg)
		case ScreenSystemPrompt:
			return m.handleSystemPromptScreen(msg)
		case ScreenRules:
			return m.handleRulesScreen(msg)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m UltimateModel) View() string {
	switch m.currentScreen {
	case ScreenMain:
		return m.renderMainScreen()
	case ScreenChat:
		return m.renderChatScreen()
	case ScreenSettings:
		return m.renderSettingsScreen()
	case ScreenModels:
		return m.renderModelsScreen()
	case ScreenAPIKeys:
		return m.renderAPIKeysScreen()
	case ScreenSystemPrompt:
		return m.renderSystemPromptScreen()
	case ScreenRules:
		return m.renderRulesScreen()
	default:
		return m.renderMainScreen()
	}
}

func (m UltimateModel) renderMainScreen() string {
	var s strings.Builder

	// Header
	header := headerStyle.Width(m.width).Render(`
╔══════════════════════════════════════════════════════════════╗
║     ██████╗  ██████╗ ██╗     ██╗     ███╗   ███╗          ║
║    ██╔════╝ ██╔═══██╗██║     ██║     ████╗ ████║          ║
║    ██║  ███╗██║   ██║██║     ██║     ██╔████╔██║          ║
║    ██║   ██║██║   ██║██║     ██║     ██║╚██╔╝██║          ║
║    ╚██████╔╝╚██████╔╝███████╗███████╗██║ ╚═╝ ██║          ║
║     ╚═════╝  ╚═════╝ ╚══════╝╚══════╝╚═╝     ╚═╝          ║
║                  ULTIMATE AI INTERFACE                      ║
╚══════════════════════════════════════════════════════════════╝`)
	s.WriteString(header + "\n\n")

	// Menu items
	menuItems := []struct {
		key   string
		icon  string
		title string
		desc  string
	}{
		{"1", "💬", "Чат с AI", "Начать диалог с выбранной моделью"},
		{"2", "🤖", "Выбор модели", "Выбрать провайдера и модель"},
		{"3", "🔑", "API ключи", "Настроить ключи доступа"},
		{"4", "⚙️", "Настройки", "Параметры генерации"},
		{"5", "📝", "Системный промпт", "Настроить поведение AI"},
		{"6", "📋", "Правила и инструкции", "Добавить специальные правила"},
		{"7", "💾", "Сохранить конфигурацию", "Сохранить все настройки"},
		{"8", "📊", "Статистика", "Просмотр использования"},
		{"9", "❓", "Помощь", "Справка по использованию"},
		{"0", "🚪", "Выход", "Закрыть приложение"},
	}

	menu := menuStyle.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("🎯 ГЛАВНОЕ МЕНЮ"),
			"",
			func() string {
				var items []string
				for _, item := range menuItems {
					style := menuItemStyle
					line := fmt.Sprintf("[%s] %s %s\n    %s",
						item.key, item.icon, item.title, item.desc)
					items = append(items, style.Render(line))
				}
				return strings.Join(items, "\n\n")
			}(),
		),
	)
	s.WriteString(menu + "\n")

	// Status bar
	status := m.renderStatusBar()
	s.WriteString(status)

	return s.String()
}

func (m UltimateModel) renderChatScreen() string {
	var s strings.Builder

	// Header
	header := titleStyle.Width(m.width).Render(
		fmt.Sprintf("💬 ЧАТ | %s | %s", m.currentProvider, m.currentModel),
	)
	s.WriteString(header + "\n")

	// Messages viewport
	messages := m.renderMessages()
	s.WriteString(messages + "\n")

	// Input area
	s.WriteString(inputStyle.Render(m.textarea.View()) + "\n")

	// Help
	help := "ESC: Меню | Enter: Отправить | Ctrl+C: Выход"
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Render(help))

	return s.String()
}

func (m UltimateModel) renderMessages() string {
	var content strings.Builder

	for _, msg := range m.messages {
		var style lipgloss.Style
		var icon string

		switch msg.Role {
		case "user":
			style = chatMessageStyle.BorderForeground(primaryColor)
			icon = "👤"
		case "assistant":
			style = chatMessageStyle.BorderForeground(accentColor)
			icon = "🤖"
		case "system":
			style = chatMessageStyle.BorderForeground(warningColor)
			icon = "⚙️"
		}

		header := fmt.Sprintf("%s %s [%s]", icon, msg.Role, msg.Timestamp.Format("15:04:05"))
		if msg.Model != "" {
			header += fmt.Sprintf(" | %s", msg.Model)
		}

		content.WriteString(style.Render(header+"\n"+msg.Content) + "\n")
	}

	m.viewport.SetContent(content.String())
	return m.viewport.View()
}

func (m UltimateModel) renderModelsScreen() string {
	var s strings.Builder

	header := titleStyle.Width(m.width).Render("🤖 ВЫБОР МОДЕЛИ")
	s.WriteString(header + "\n\n")

	for i, provider := range m.providers {
		providerStyle := menuItemStyle
		if provider.Name == m.currentProvider {
			providerStyle = activeMenuItemStyle
		}

		s.WriteString(providerStyle.Render(
			fmt.Sprintf("[%d] %s", i+1, provider.DisplayName),
		) + "\n")

		if provider.Name == m.currentProvider {
			for j, model := range provider.Models {
				modelStyle := menuItemStyle
				if model.ID == m.currentModel {
					modelStyle = activeMenuItemStyle
				}

				s.WriteString(modelStyle.Render(
					fmt.Sprintf("    [%c] %s - %s (Max: %d токенов)",
						'a'+j, model.Name, model.Description, model.MaxTokens),
				) + "\n")
			}
		}
		s.WriteString("\n")
	}

	s.WriteString("\nESC: Назад | Enter: Выбрать")

	return s.String()
}

func (m UltimateModel) renderAPIKeysScreen() string {
	var s strings.Builder

	header := titleStyle.Width(m.width).Render("🔑 НАСТРОЙКА API КЛЮЧЕЙ")
	s.WriteString(header + "\n\n")

	for _, provider := range m.providers {
		if !provider.RequiresKey {
			continue
		}

		s.WriteString(fmt.Sprintf("%s:\n", provider.DisplayName))

		input := m.apiKeyInputs[provider.Name]
		hasKey := input.Value() != ""

		status := "❌ Не установлен"
		if hasKey {
			status = "✅ Установлен"
		}

		s.WriteString(fmt.Sprintf("Статус: %s\n", status))
		s.WriteString(input.View() + "\n\n")
	}

	s.WriteString("\nTab: Следующее поле | Enter: Сохранить | ESC: Назад")

	return s.String()
}

func (m UltimateModel) renderSettingsScreen() string {
	var s strings.Builder

	header := titleStyle.Width(m.width).Render("⚙️ НАСТРОЙКИ")
	s.WriteString(header + "\n\n")

	settings := []struct {
		name  string
		value string
		desc  string
	}{
		{"Max Tokens", fmt.Sprintf("%d", m.config.MaxTokens), "Максимальное количество токенов"},
		{"Temperature", fmt.Sprintf("%.1f", m.config.Temperature), "Креативность (0.0-2.0)"},
		{"Stream Mode", fmt.Sprintf("%v", m.config.StreamMode), "Потоковый вывод"},
		{"Auto Save", fmt.Sprintf("%v", m.config.AutoSave), "Автосохранение"},
		{"Theme", m.config.Theme, "Тема оформления"},
	}

	for i, setting := range settings {
		s.WriteString(fmt.Sprintf("[%d] %s: %s\n    %s\n\n",
			i+1, setting.name, setting.value, setting.desc))
	}

	s.WriteString("\nВыберите настройку для изменения | ESC: Назад")

	return s.String()
}

func (m UltimateModel) renderSystemPromptScreen() string {
	var s strings.Builder

	header := titleStyle.Width(m.width).Render("📝 СИСТЕМНЫЙ ПРОМПТ")
	s.WriteString(header + "\n\n")

	s.WriteString("Текущий системный промпт:\n\n")
	s.WriteString(chatMessageStyle.Render(m.systemPrompt) + "\n\n")

	s.WriteString("Примеры промптов:\n")
	prompts := []string{
		"[1] Профессиональный ассистент",
		"[2] Дружелюбный помощник",
		"[3] Технический эксперт",
		"[4] Креативный писатель",
		"[5] Учитель",
	}

	for _, p := range prompts {
		s.WriteString(menuItemStyle.Render(p) + "\n")
	}

	s.WriteString("\n[E] Редактировать | [R] Сбросить | ESC: Назад")

	return s.String()
}

func (m UltimateModel) renderRulesScreen() string {
	var s strings.Builder

	header := titleStyle.Width(m.width).Render("📋 ПРАВИЛА И ИНСТРУКЦИИ")
	s.WriteString(header + "\n\n")

	if len(m.rules) == 0 {
		s.WriteString("Нет добавленных правил\n\n")
	} else {
		for i, rule := range m.rules {
			status := "🔴"
			if rule.Enabled {
				status = "🟢"
			}

			s.WriteString(fmt.Sprintf("%s [%d] %s\n", status, i+1, rule.Name))
			s.WriteString(fmt.Sprintf("    %s\n\n", rule.Description))
		}
	}

	s.WriteString("[A] Добавить правило | [E] Редактировать | [D] Удалить | ESC: Назад")

	return s.String()
}

func (m UltimateModel) renderStatusBar() string {
	provider := "Не выбран"
	if m.currentProvider != "" {
		provider = m.currentProvider
	}

	model := "Не выбрана"
	if m.currentModel != "" {
		model = m.currentModel
	}

	apiStatus := "❌"
	if cfg, ok := m.config.Providers[m.currentProvider]; ok && cfg.APIKey != "" {
		apiStatus = "✅"
	}

	status := fmt.Sprintf(
		" Provider: %s | Model: %s | API: %s | Messages: %d | %s ",
		provider, model, apiStatus, len(m.messages), m.status,
	)

	return statusBarStyle.Width(m.width).Render(status)
}

// Handler functions for different screens
func (m UltimateModel) handleMainScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.currentScreen = ScreenChat
	case "2":
		m.currentScreen = ScreenModels
	case "3":
		m.currentScreen = ScreenAPIKeys
	case "4":
		m.currentScreen = ScreenSettings
	case "5":
		m.currentScreen = ScreenSystemPrompt
	case "6":
		m.currentScreen = ScreenRules
	case "7":
		m.saveConfiguration()
		m.status = "✅ Конфигурация сохранена"
	case "0", "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m UltimateModel) handleChatScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentScreen = ScreenMain
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		// Send message
		content := m.textarea.Value()
		if content != "" {
			m.messages = append(m.messages, ChatMessage{
				Role:      "user",
				Content:   content,
				Timestamp: time.Now(),
				Provider:  m.currentProvider,
				Model:     m.currentModel,
			})
			m.textarea.Reset()
			// Here would be actual API call
			m.simulateResponse()
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}

func (m *UltimateModel) simulateResponse() {
	// Simulate AI response
	time.Sleep(500 * time.Millisecond)

	m.messages = append(m.messages, ChatMessage{
		Role:      "assistant",
		Content:   "Это симуляция ответа. Для реальной работы нужно подключить API.",
		Timestamp: time.Now(),
		Provider:  m.currentProvider,
		Model:     m.currentModel,
	})
}

func (m UltimateModel) handleModelsScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentScreen = ScreenMain
	case "1", "2", "3", "4":
		// Select provider
		index := int(msg.String()[0] - '1')
		if index < len(m.providers) {
			m.currentProvider = m.providers[index].Name
			// Select first model by default
			if len(m.providers[index].Models) > 0 {
				m.currentModel = m.providers[index].Models[0].ID
			}
			m.config.CurrentProvider = m.currentProvider
			m.config.CurrentModel = m.currentModel
		}
	case "a", "b", "c", "d", "e":
		// Select model within provider
		for _, provider := range m.providers {
			if provider.Name == m.currentProvider {
				index := int(msg.String()[0] - 'a')
				if index < len(provider.Models) {
					m.currentModel = provider.Models[index].ID
					m.config.CurrentModel = m.currentModel
				}
				break
			}
		}
	}
	return m, nil
}

func (m UltimateModel) handleAPIKeysScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentScreen = ScreenMain
		// Save API keys to config
		for provider, input := range m.apiKeyInputs {
			if m.config.Providers == nil {
				m.config.Providers = make(map[string]ProviderConfig)
			}
			cfg := m.config.Providers[provider]
			cfg.APIKey = input.Value()
			cfg.Name = provider
			cfg.Enabled = input.Value() != ""
			m.config.Providers[provider] = cfg
		}
		m.saveConfiguration()
	case "tab":
		// Switch between inputs
		m.focusedInput = (m.focusedInput + 1) % len(m.apiKeyInputs)
	default:
		// Update current input
		for i, provider := range m.providers {
			if i == m.focusedInput && provider.RequiresKey {
				input := m.apiKeyInputs[provider.Name]
				var cmd tea.Cmd
				input, cmd = input.Update(msg)
				m.apiKeyInputs[provider.Name] = input
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m UltimateModel) handleSettingsScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentScreen = ScreenMain
	case "1":
		// Change max tokens
		// Would open input dialog
	case "2":
		// Change temperature
		// Would open input dialog
	case "3":
		// Toggle stream mode
		m.config.StreamMode = !m.config.StreamMode
	case "4":
		// Toggle auto save
		m.config.AutoSave = !m.config.AutoSave
	case "5":
		// Change theme
		// Would open theme selector
	}
	return m, nil
}

func (m UltimateModel) handleSystemPromptScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentScreen = ScreenMain
	case "1":
		m.systemPrompt = "Ты профессиональный ассистент. Отвечай точно и по делу."
	case "2":
		m.systemPrompt = "Ты дружелюбный помощник. Будь вежлив и позитивен."
	case "3":
		m.systemPrompt = "Ты технический эксперт. Давай подробные технические ответы."
	case "4":
		m.systemPrompt = "Ты креативный писатель. Используй богатый язык и метафоры."
	case "5":
		m.systemPrompt = "Ты учитель. Объясняй понятно и пошагово."
	case "e":
		// Would open editor
	case "r":
		m.systemPrompt = "Ты полезный AI ассистент."
	}
	m.config.SystemPrompt = m.systemPrompt
	return m, nil
}

func (m UltimateModel) handleRulesScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentScreen = ScreenMain
	case "a":
		// Add new rule
		m.rules = append(m.rules, Rule{
			Name:        fmt.Sprintf("Правило %d", len(m.rules)+1),
			Description: "Новое правило",
			Content:     "Содержание правила",
			Enabled:     true,
		})
	case "d":
		// Delete last rule
		if len(m.rules) > 0 {
			m.rules = m.rules[:len(m.rules)-1]
		}
	}
	m.config.Rules = m.rules
	return m, nil
}
