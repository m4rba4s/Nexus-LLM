package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message represents a chat message
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// Advanced TUI with modern design and animations
type AdvancedModel struct {
	// UI Components
	viewport  viewport.Model
	textInput textinput.Model
	spinner   spinner.Model
	help      help.Model

	// State
	messages     []Message
	menuItems    []MenuItem
	selectedMenu int
	showMenu     bool
	typing       bool
	processing   bool

	// Animation
	animFrame   int
	logoFrame   int
	gradientPos float64

	// Layout
	width  int
	height int

	// Features
	autoExecute bool
	mcpEnabled  bool

	// Styles
	theme Theme
}

type MenuItem struct {
	Icon        string
	Title       string
	Description string
	Action      string
	Hotkey      string
}

type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Text       lipgloss.Color
	Border     lipgloss.Color
	Success    lipgloss.Color
	Error      lipgloss.Color
	Warning    lipgloss.Color
}

// Neon Cyberpunk Theme
var NeonTheme = Theme{
	Primary:    lipgloss.Color("#00ffff"), // Cyan
	Secondary:  lipgloss.Color("#ff00ff"), // Magenta
	Accent:     lipgloss.Color("#ffff00"), // Yellow
	Background: lipgloss.Color("#0a0e27"), // Dark blue
	Text:       lipgloss.Color("#e0e0e0"), // Light gray
	Border:     lipgloss.Color("#00ff88"), // Green
	Success:    lipgloss.Color("#00ff00"), // Bright green
	Error:      lipgloss.Color("#ff0040"), // Red
	Warning:    lipgloss.Color("#ffa500"), // Orange
}

func NewAdvancedModel() AdvancedModel {
	ti := textinput.New()
	ti.Placeholder = "💬 Type your message or / for commands..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(NeonTheme.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(NeonTheme.Text)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(NeonTheme.Accent)

	vp := viewport.New(80, 20)
	vp.MouseWheelEnabled = true

	h := help.New()
	h.ShowAll = false

	menuItems := []MenuItem{
		{Icon: "🤖", Title: "Chat Mode", Description: "Interactive conversation with AI", Action: "chat", Hotkey: "c"},
		{Icon: "📝", Title: "Code Assistant", Description: "Generate and run code", Action: "code", Hotkey: "x"},
		{Icon: "🚀", Title: "Execute Command", Description: "Run system commands", Action: "exec", Hotkey: "e"},
		{Icon: "🔧", Title: "MCP Tools", Description: "Access MCP server tools", Action: "mcp", Hotkey: "m"},
		{Icon: "📊", Title: "Analytics", Description: "View usage statistics", Action: "stats", Hotkey: "s"},
		{Icon: "🎨", Title: "Change Theme", Description: "Switch UI theme", Action: "theme", Hotkey: "t"},
		{Icon: "⚙️", Title: "Settings", Description: "Configure options", Action: "settings", Hotkey: "o"},
		{Icon: "📚", Title: "Help", Description: "Show help and shortcuts", Action: "help", Hotkey: "h"},
	}

	return AdvancedModel{
		viewport:     vp,
		textInput:    ti,
		spinner:      s,
		help:         h,
		messages:     []Message{},
		menuItems:    menuItems,
		selectedMenu: 0,
		showMenu:     false,
		typing:       false,
		processing:   false,
		animFrame:    0,
		logoFrame:    0,
		gradientPos:  0,
		width:        80,
		height:       24,
		autoExecute:  false,
		mcpEnabled:   true,
		theme:        NeonTheme,
	}
}

func (m AdvancedModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		animationTick(),
	)
}

type tickMsg time.Time

func animationTick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m AdvancedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		m.textInput.Width = msg.Width - 6

	case tickMsg:
		m.animFrame++
		m.logoFrame = (m.logoFrame + 1) % 60
		m.gradientPos += 0.05
		if m.gradientPos > 2*math.Pi {
			m.gradientPos = 0
		}

		if m.processing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

		cmds = append(cmds, animationTick())

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+m"))):
			m.showMenu = !m.showMenu

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+e"))):
			m.autoExecute = !m.autoExecute

		case m.showMenu:
			switch msg.String() {
			case "up", "k":
				if m.selectedMenu > 0 {
					m.selectedMenu--
				} else {
					m.selectedMenu = len(m.menuItems) - 1
				}
			case "down", "j":
				m.selectedMenu = (m.selectedMenu + 1) % len(m.menuItems)
			case "enter":
				m.executeMenuItem(m.menuItems[m.selectedMenu])
				m.showMenu = false
			case "esc":
				m.showMenu = false
			default:
				// Check hotkeys
				for i, item := range m.menuItems {
					if msg.String() == item.Hotkey {
						m.selectedMenu = i
						m.executeMenuItem(item)
						m.showMenu = false
						break
					}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.textInput.Value() != "" {
				m.handleInput(m.textInput.Value())
				m.textInput.SetValue("")
			}

		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
		}

	case spinner.TickMsg:
		if m.processing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update viewport with messages
	m.updateViewport()

	return m, tea.Batch(cmds...)
}

func (m *AdvancedModel) executeMenuItem(item MenuItem) {
	switch item.Action {
	case "chat":
		m.addSystemMessage("💬 Chat mode activated")
	case "code":
		m.addSystemMessage("📝 Code assistant ready")
	case "exec":
		m.addSystemMessage("🚀 Command execution mode")
	case "mcp":
		if m.mcpEnabled {
			m.addSystemMessage("🔧 MCP tools available")
		} else {
			m.addSystemMessage("⚠️ MCP server not connected")
		}
	case "stats":
		m.showStatistics()
	case "theme":
		m.cycleTheme()
	case "settings":
		m.showSettings()
	case "help":
		m.showHelp()
	}
}

func (m *AdvancedModel) handleInput(input string) {
	// Add user message
	m.addUserMessage(input)

	// Check for commands
	if strings.HasPrefix(input, "/") {
		m.handleCommand(input[1:])
		return
	}

	// Process with AI
	m.processing = true
	// Here would be actual AI processing
	go func() {
		time.Sleep(2 * time.Second) // Simulate processing
		m.addAIMessage("This is a simulated AI response with **markdown** support!")
		m.processing = false
	}()
}

func (m *AdvancedModel) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "run", "exec":
		if len(parts) > 1 {
			command := strings.Join(parts[1:], " ")
			m.executeCommand(command)
		}
	case "clear":
		m.messages = []Message{}
	case "help":
		m.showHelp()
	case "mcp":
		m.showMCPTools()
	default:
		m.addSystemMessage(fmt.Sprintf("❌ Unknown command: %s", parts[0]))
	}
}

func (m *AdvancedModel) executeCommand(cmd string) {
	if m.autoExecute {
		m.addSystemMessage(fmt.Sprintf("⚡ Auto-executing: %s", cmd))
		// Execute command here
	} else {
		m.addSystemMessage(fmt.Sprintf("⚠️ Command ready: %s\nPress Ctrl+E to enable auto-execution", cmd))
	}
}

func (m *AdvancedModel) addUserMessage(text string) {
	m.messages = append(m.messages, Message{
		Role:      "user",
		Content:   text,
		Timestamp: time.Now(),
	})
}

func (m *AdvancedModel) addAIMessage(text string) {
	m.messages = append(m.messages, Message{
		Role:      "assistant",
		Content:   text,
		Timestamp: time.Now(),
	})
}

func (m *AdvancedModel) addSystemMessage(text string) {
	m.messages = append(m.messages, Message{
		Role:      "system",
		Content:   text,
		Timestamp: time.Now(),
	})
}

func (m *AdvancedModel) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		style := m.getMessageStyle(msg.Role)
		icon := m.getMessageIcon(msg.Role)

		content.WriteString(fmt.Sprintf("%s %s\n%s\n\n",
			icon,
			msg.Timestamp.Format("15:04:05"),
			style.Render(msg.Content)))
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m AdvancedModel) getMessageStyle(role string) lipgloss.Style {
	base := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		MarginBottom(1)

	switch role {
	case "user":
		return base.
			Foreground(m.theme.Text).
			Background(lipgloss.Color("#1a1a2e")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Primary)
	case "assistant":
		return base.
			Foreground(m.theme.Text).
			Background(lipgloss.Color("#16213e")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Secondary)
	case "system":
		return base.
			Foreground(m.theme.Warning).
			Italic(true)
	default:
		return base
	}
}

func (m AdvancedModel) getMessageIcon(role string) string {
	switch role {
	case "user":
		return "👤"
	case "assistant":
		return "🤖"
	case "system":
		return "ℹ️"
	default:
		return "📝"
	}
}

func (m AdvancedModel) View() string {
	var s strings.Builder

	// Animated header with logo
	s.WriteString(m.renderAnimatedHeader())
	s.WriteString("\n")

	// Main content area
	if m.showMenu {
		s.WriteString(m.renderMenu())
	} else {
		s.WriteString(m.renderChat())
	}

	// Status bar
	s.WriteString(m.renderStatusBar())

	// Input area
	s.WriteString("\n")
	s.WriteString(m.textInput.View())

	// Help line
	s.WriteString("\n")
	s.WriteString(m.renderHelpLine())

	return s.String()
}

func (m AdvancedModel) renderAnimatedHeader() string {
	// Animated gradient logo
	logo := m.generateAnimatedLogo()

	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(m.theme.Border).
		Padding(1, 2)

	return headerStyle.Render(logo)
}

func (m AdvancedModel) generateAnimatedLogo() string {
	frames := []string{
		`╔══════════════════════════════════════╗
║  ▄████  ▒█████   ██▓     ██▓     ███▄ ▄███▓ ║
║ ██▒ ▀█▒▒██▒  ██▒▓██▒    ▓██▒    ▓██▒▀█▀ ██▒ ║
║▒██░▄▄▄░▒██░  ██▒▒██░    ▒██░    ▓██    ▓██░ ║
║░▓█  ██▓▒██   ██░▒██░    ▒██░    ▒██    ▒██  ║
║░▒▓███▀▒░ ████▓▒░░██████▒░██████▒▒██▒   ░██▒ ║
║     Advanced AI Terminal Interface          ║
╚══════════════════════════════════════╝`,
		`╔══════════════════════════════════════╗
║  ▓████  ▒█████   ██▓     ██▓     ███▄ ▄███▓ ║
║ ██▒ ▀█▒▒██▒  ██▒▓██▒    ▓██▒    ▓██▒▀█▀ ██▒ ║
║▒██░▄▄▄░▒██░  ██▒▒██░    ▒██░    ▓██    ▓██░ ║
║░▓█  ██▓▒██   ██░▒██░    ▒██░    ▒██    ▒██  ║
║░▒▓███▀▒░ ████▓▒░░██████▒░██████▒▒██▒   ░██▒ ║
║     Advanced AI Terminal Interface          ║
╚══════════════════════════════════════╝`,
	}

	// Apply rainbow gradient effect
	colors := []lipgloss.Color{
		lipgloss.Color("#ff0000"),
		lipgloss.Color("#ff7f00"),
		lipgloss.Color("#ffff00"),
		lipgloss.Color("#00ff00"),
		lipgloss.Color("#0000ff"),
		lipgloss.Color("#4b0082"),
		lipgloss.Color("#9400d3"),
	}

	colorIndex := int(m.gradientPos*2) % len(colors)
	style := lipgloss.NewStyle().Foreground(colors[colorIndex])

	frameIndex := m.logoFrame / 30
	return style.Render(frames[frameIndex%len(frames)])
}

func (m AdvancedModel) renderMenu() string {
	menuStyle := lipgloss.NewStyle().
		Width(m.width-4).
		Height(m.height-12).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Padding(1, 2)

	var items strings.Builder
	items.WriteString("🎯 MAIN MENU\n\n")

	for i, item := range m.menuItems {
		cursor := "  "
		if i == m.selectedMenu {
			cursor = "▶ "
		}

		itemStyle := lipgloss.NewStyle()
		if i == m.selectedMenu {
			itemStyle = itemStyle.
				Foreground(m.theme.Accent).
				Bold(true).
				Background(lipgloss.Color("#1a1a2e"))
		} else {
			itemStyle = itemStyle.Foreground(m.theme.Text)
		}

		items.WriteString(itemStyle.Render(fmt.Sprintf("%s%s %s [%s]\n   %s\n\n",
			cursor, item.Icon, item.Title, item.Hotkey, item.Description)))
	}

	return menuStyle.Render(items.String())
}

func (m AdvancedModel) renderChat() string {
	chatStyle := lipgloss.NewStyle().
		Width(m.width - 4).
		Height(m.height - 12).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	content := m.viewport.View()

	if m.processing {
		content += "\n" + m.spinner.View() + " Processing..."
	}

	return chatStyle.Render(content)
}

func (m AdvancedModel) renderStatusBar() string {
	statusItems := []string{
		fmt.Sprintf("📊 Messages: %d", len(m.messages)),
		fmt.Sprintf("🔌 MCP: %s", m.boolToStatus(m.mcpEnabled)),
		fmt.Sprintf("⚡ Auto-Exec: %s", m.boolToStatus(m.autoExecute)),
		fmt.Sprintf("🎨 Theme: Neon"),
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Background(lipgloss.Color("#0a0e27")).
		Padding(0, 1)

	return statusStyle.Render(strings.Join(statusItems, " │ "))
}

func (m AdvancedModel) renderHelpLine() string {
	help := "Ctrl+M: Menu │ Ctrl+E: Toggle Auto-Exec │ Ctrl+C: Quit │ /help: Commands"

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)

	return helpStyle.Render(help)
}

func (m AdvancedModel) boolToStatus(b bool) string {
	if b {
		return "✅ ON"
	}
	return "❌ OFF"
}

func (m *AdvancedModel) showStatistics() {
	stats := fmt.Sprintf(`📊 USAGE STATISTICS
━━━━━━━━━━━━━━━━━━━━
Total Messages: %d
Session Duration: %s
Commands Executed: 0
Tokens Used: 0
Average Response Time: 0ms
`, len(m.messages), time.Since(time.Now()).String())

	m.addSystemMessage(stats)
}

func (m *AdvancedModel) showSettings() {
	settings := fmt.Sprintf(`⚙️ SETTINGS
━━━━━━━━━━━━━━━━━━━━
Auto-Execute: %s
MCP Server: %s
Theme: Neon
Max Tokens: 2048
Temperature: 0.7
Stream Mode: Enabled
`, m.boolToStatus(m.autoExecute), m.boolToStatus(m.mcpEnabled))

	m.addSystemMessage(settings)
}

func (m *AdvancedModel) showHelp() {
	help := `📚 HELP & COMMANDS
━━━━━━━━━━━━━━━━━━━━
SHORTCUTS:
  Ctrl+M     Toggle menu
  Ctrl+E     Toggle auto-execution
  Ctrl+C     Quit application
  ↑/↓        Navigate menu/history
  Enter      Send message/select

COMMANDS:
  /run <cmd>   Execute system command
  /clear       Clear chat history
  /mcp         Show MCP tools
  /theme       Change theme
  /help        Show this help

FEATURES:
  • AI-powered responses
  • Code generation & execution
  • MCP server integration
  • Real-time streaming
  • Multi-provider support`

	m.addSystemMessage(help)
}

func (m *AdvancedModel) showMCPTools() {
	tools := `🔧 MCP TOOLS AVAILABLE
━━━━━━━━━━━━━━━━━━━━
📁 File Operations
  - Read/Write files
  - Directory navigation
  
🌐 Web Requests
  - HTTP/HTTPS requests
  - API interactions
  
💾 Database Access
  - Query execution
  - Schema inspection
  
🔍 Search & Analysis
  - Code search
  - Pattern matching
  
📊 Data Processing
  - JSON/CSV parsing
  - Data transformation`

	m.addSystemMessage(tools)
}

func (m *AdvancedModel) cycleTheme() {
	// In a real implementation, this would cycle through themes
	m.addSystemMessage("🎨 Theme changed to: Neon Cyberpunk")
}

// SetAutoExecute enables or disables auto-execution mode
func (m *AdvancedModel) SetAutoExecute(enabled bool) {
	m.autoExecute = enabled
}

// SetTheme sets the UI theme
func (m *AdvancedModel) SetTheme(themeName string) {
	switch themeName {
	case "neon":
		m.theme = NeonTheme
	case "dark":
		m.theme = DarkTheme
	case "light":
		m.theme = LightTheme
	case "matrix":
		m.theme = MatrixTheme
	default:
		m.theme = NeonTheme
	}
}

// SetMCPEnabled enables or disables MCP integration
func (m *AdvancedModel) SetMCPEnabled(enabled bool) {
	m.mcpEnabled = enabled
}

// SetMCPEndpoint sets the MCP server endpoint
func (m *AdvancedModel) SetMCPEndpoint(endpoint string) {
	// Store endpoint for MCP client connection
	// This would be used to connect to the MCP server
}

// Additional themes
var DarkTheme = Theme{
	Primary:    lipgloss.Color("#4a9eff"),
	Secondary:  lipgloss.Color("#ff6b6b"),
	Accent:     lipgloss.Color("#4ecdc4"),
	Background: lipgloss.Color("#1a1a1a"),
	Text:       lipgloss.Color("#f0f0f0"),
	Border:     lipgloss.Color("#555555"),
	Success:    lipgloss.Color("#4caf50"),
	Error:      lipgloss.Color("#f44336"),
	Warning:    lipgloss.Color("#ff9800"),
}

var LightTheme = Theme{
	Primary:    lipgloss.Color("#2196f3"),
	Secondary:  lipgloss.Color("#e91e63"),
	Accent:     lipgloss.Color("#00bcd4"),
	Background: lipgloss.Color("#ffffff"),
	Text:       lipgloss.Color("#212121"),
	Border:     lipgloss.Color("#e0e0e0"),
	Success:    lipgloss.Color("#4caf50"),
	Error:      lipgloss.Color("#f44336"),
	Warning:    lipgloss.Color("#ff9800"),
}

var MatrixTheme = Theme{
	Primary:    lipgloss.Color("#00ff00"),
	Secondary:  lipgloss.Color("#00cc00"),
	Accent:     lipgloss.Color("#00ff00"),
	Background: lipgloss.Color("#000000"),
	Text:       lipgloss.Color("#00ff00"),
	Border:     lipgloss.Color("#00aa00"),
	Success:    lipgloss.Color("#00ff00"),
	Error:      lipgloss.Color("#ff0000"),
	Warning:    lipgloss.Color("#ffff00"),
}
