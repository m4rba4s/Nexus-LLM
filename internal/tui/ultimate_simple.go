package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SimpleUltimateModel - упрощённая версия Ultimate интерфейса
type SimpleUltimateModel struct {
	screen   int
	width    int
	height   int
	choice   int
	quitting bool
	message  string
	apiKey   string
	provider string
	model    string
}

var (
	// Стили
	simpleHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7C3AED")).
				Align(lipgloss.Center).
				Padding(1, 0)

	simpleMenuStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2)

	simpleSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true)

	simpleNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F3F4F6"))
)

// NewSimpleUltimateModel создаёт новую упрощённую модель
func NewSimpleUltimateModel() *SimpleUltimateModel {
	return &SimpleUltimateModel{
		screen:   0,
		provider: "openai",
		model:    "gpt-3.5-turbo",
		message:  "Добро пожаловать в Ultimate AI Interface!",
	}
}

func (m SimpleUltimateModel) Init() tea.Cmd {
	return nil
}

func (m SimpleUltimateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.choice > 0 {
				m.choice--
			}

		case "down", "j":
			if m.choice < 8 {
				m.choice++
			}

		case "enter":
			return m.handleChoice()

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			choice := int(msg.String()[0] - '1')
			if choice >= 0 && choice <= 8 {
				m.choice = choice
				return m.handleChoice()
			}

		case "0":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.screen != 0 {
				m.screen = 0
				m.message = "Вернулись в главное меню"
			}
		}
	}

	return m, nil
}

func (m SimpleUltimateModel) handleChoice() (tea.Model, tea.Cmd) {
	switch m.choice {
	case 0: // Chat
		m.screen = 1
		m.message = "💬 Режим чата (в разработке)"
	case 1: // Models
		m.screen = 2
		m.message = "🤖 Выбор модели"
	case 2: // API Keys
		m.screen = 3
		m.message = "🔑 Настройка API ключей"
	case 3: // Settings
		m.screen = 4
		m.message = "⚙️ Настройки"
	case 4: // System Prompt
		m.screen = 5
		m.message = "📝 Системный промпт"
	case 5: // Rules
		m.screen = 6
		m.message = "📋 Правила и инструкции"
	case 6: // Save
		m.message = "💾 Конфигурация сохранена!"
	case 7: // Stats
		m.message = "📊 Статистика"
	case 8: // Help
		m.message = "❓ Помощь"
	}
	return m, nil
}

func (m SimpleUltimateModel) View() string {
	if m.quitting {
		return "👋 До свидания!\n"
	}

	switch m.screen {
	case 0:
		return m.renderMainMenu()
	case 1:
		return m.renderChatScreen()
	case 2:
		return m.renderModelsScreen()
	case 3:
		return m.renderAPIKeysScreen()
	case 4:
		return m.renderSettingsScreen()
	case 5:
		return m.renderSystemPromptScreen()
	case 6:
		return m.renderRulesScreen()
	default:
		return m.renderMainMenu()
	}
}

func (m SimpleUltimateModel) renderMainMenu() string {
	var s strings.Builder

	// Header
	header := simpleHeaderStyle.Width(m.width).Render(`
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

	// Status message
	s.WriteString(fmt.Sprintf("📢 %s\n\n", m.message))

	// Menu
	menuItems := []string{
		"[1] 💬 Чат с AI",
		"[2] 🤖 Выбор модели",
		"[3] 🔑 API ключи",
		"[4] ⚙️  Настройки",
		"[5] 📝 Системный промпт",
		"[6] 📋 Правила",
		"[7] 💾 Сохранить",
		"[8] 📊 Статистика",
		"[9] ❓ Помощь",
		"[0] 🚪 Выход",
	}

	menu := simpleMenuStyle.Render(
		"🎯 ГЛАВНОЕ МЕНЮ\n\n" +
			func() string {
				var items []string
				for i, item := range menuItems {
					if i == m.choice {
						items = append(items, simpleSelectedStyle.Render("→ "+item))
					} else {
						items = append(items, simpleNormalStyle.Render("  "+item))
					}
				}
				return strings.Join(items, "\n")
			}(),
	)
	s.WriteString(menu + "\n\n")

	// Footer
	footer := simpleNormalStyle.Render(
		"Навигация: ↑↓ или j/k • Выбор: Enter или цифра • Выход: q или Ctrl+C",
	)
	s.WriteString(footer)

	return s.String()
}

func (m SimpleUltimateModel) renderChatScreen() string {
	return simpleHeaderStyle.Render("💬 ЧАТ С AI") + "\n\n" +
		simpleMenuStyle.Render(
			"Провайдер: "+m.provider+"\n"+
				"Модель: "+m.model+"\n\n"+
				"[Функция чата в разработке]\n\n"+
				"Нажмите ESC для возврата в меню",
		)
}

func (m SimpleUltimateModel) renderModelsScreen() string {
	return simpleHeaderStyle.Render("🤖 ВЫБОР МОДЕЛИ") + "\n\n" +
		simpleMenuStyle.Render(
			"ДОСТУПНЫЕ ПРОВАЙДЕРЫ:\n\n"+
				"1. OpenAI\n"+
				"   • gpt-4-turbo-preview\n"+
				"   • gpt-4\n"+
				"   • gpt-3.5-turbo ✓\n\n"+
				"2. Anthropic Claude\n"+
				"   • claude-3-opus\n"+
				"   • claude-3-sonnet\n"+
				"   • claude-3-haiku\n\n"+
				"3. Google Gemini\n"+
				"   • gemini-pro\n"+
				"   • gemini-pro-vision\n\n"+
				"4. Ollama (Локальные)\n"+
				"   • llama2\n"+
				"   • mistral\n"+
				"   • codellama\n\n"+
				"Текущий выбор: "+m.provider+" / "+m.model+"\n\n"+
				"Нажмите ESC для возврата",
		)
}

func (m SimpleUltimateModel) renderAPIKeysScreen() string {
	maskedKey := ""
	if m.apiKey != "" {
		if len(m.apiKey) > 8 {
			maskedKey = m.apiKey[:4] + "..." + m.apiKey[len(m.apiKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	return simpleHeaderStyle.Render("🔑 НАСТРОЙКА API КЛЮЧЕЙ") + "\n\n" +
		simpleMenuStyle.Render(
			"ПРОВАЙДЕРЫ:\n\n"+
				"OpenAI:     "+func() string {
				if maskedKey != "" {
					return "✓ " + maskedKey
				}
				return "❌ Не настроен"
			}()+"\n"+
				"Anthropic:  ❌ Не настроен\n"+
				"Gemini:     ❌ Не настроен\n"+
				"DeepSeek:   ❌ Не настроен\n"+
				"OpenRouter: ❌ Не настроен\n"+
				"Ollama:     ✓ Локальный (ключ не требуется)\n\n"+
				"[Функция ввода ключей в разработке]\n\n"+
				"Нажмите ESC для возврата",
		)
}

func (m SimpleUltimateModel) renderSettingsScreen() string {
	return simpleHeaderStyle.Render("⚙️ НАСТРОЙКИ") + "\n\n" +
		simpleMenuStyle.Render(
			"ПАРАМЕТРЫ ГЕНЕРАЦИИ:\n\n"+
				"Max Tokens:    2048\n"+
				"Temperature:   0.7\n"+
				"Top P:         1.0\n"+
				"Stream Mode:   ✓ Включен\n"+
				"Auto Save:     ✓ Включен\n\n"+
				"ИНТЕРФЕЙС:\n"+
				"Тема:          Modern\n"+
				"Язык:          Русский\n\n"+
				"[Функция изменения настроек в разработке]\n\n"+
				"Нажмите ESC для возврата",
		)
}

func (m SimpleUltimateModel) renderSystemPromptScreen() string {
	return simpleHeaderStyle.Render("📝 СИСТЕМНЫЙ ПРОМПТ") + "\n\n" +
		simpleMenuStyle.Render(
			"ТЕКУЩИЙ ПРОМПТ:\n\n"+
				"\"Ты полезный AI ассистент, который помогает\n"+
				"пользователям решать различные задачи.\n"+
				"Отвечай точно, подробно и вежливо.\"\n\n"+
				"ШАБЛОНЫ:\n"+
				"1. Профессиональный ассистент\n"+
				"2. Дружелюбный помощник\n"+
				"3. Технический эксперт\n"+
				"4. Креативный писатель\n"+
				"5. Учитель\n\n"+
				"[Функция редактирования в разработке]\n\n"+
				"Нажмите ESC для возврата",
		)
}

func (m SimpleUltimateModel) renderRulesScreen() string {
	return simpleHeaderStyle.Render("📋 ПРАВИЛА И ИНСТРУКЦИИ") + "\n\n" +
		simpleMenuStyle.Render(
			"АКТИВНЫЕ ПРАВИЛА:\n\n"+
				"✓ Правило 1: Всегда проверять факты\n"+
				"✓ Правило 2: Быть вежливым и профессиональным\n"+
				"✓ Правило 3: Давать развёрнутые ответы\n"+
				"□ Правило 4: Использовать эмодзи\n"+
				"□ Правило 5: Добавлять примеры кода\n\n"+
				"ДЕЙСТВИЯ:\n"+
				"[A] Добавить правило\n"+
				"[E] Редактировать\n"+
				"[D] Удалить\n"+
				"[T] Переключить активность\n\n"+
				"[Функция управления правилами в разработке]\n\n"+
				"Нажмите ESC для возврата",
		)
}
