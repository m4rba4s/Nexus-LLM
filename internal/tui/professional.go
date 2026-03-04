package tui

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"
)

// OperationMode represents the two main modes of operation
type OperationMode int

const (
	ModeOperator OperationMode = iota // PC Operator - system admin, automation
	ModeCoder                         // Coder - development, refactoring, testing
)

// ProfessionalModel represents the state of our professional TUI
type ProfessionalModel struct {
	// Streaming control
	streamChan   <-chan core.StreamChunk
	cancelStream context.CancelFunc

	// Core components
	// Core components
	config    *config.Config
	aiService *AIService
	mode      OperationMode

	// UI state
	width    int
	height   int
	ready    bool
	quitting bool

	// Current selections
	provider string
	model    string

	// Menu state
	menuVisible bool
	menuIndex   int
	menuItems   []ProMenuItem

	// Chat components
	viewport  viewport.Model
	textarea  textarea.Model
	messages  []ProMessage
	streaming bool

	// Visual components
	spinner spinner.Model
	help    help.Model
	keymap  KeyMap

	// Tool states
	toolsActive map[string]bool

	// System info
	systemInfo SystemInfo

	// Performance metrics
	metrics PerformanceMetrics
}

// ProMenuItem represents a menu option
type ProMenuItem struct {
	Title       string
	Description string
	Action      func() tea.Cmd
	Hotkey      string
	Icon        string
}

// ProMessage represents a chat message
type ProMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
	Provider  string
	Model     string
	Tokens    int
	Latency   time.Duration
}

// SystemInfo contains system information
type SystemInfo struct {
	OS         string
	Arch       string
	CPUCores   int
	Memory     string
	GoVersion  string
	WorkingDir string
}

// PerformanceMetrics tracks performance data
type PerformanceMetrics struct {
	TotalRequests  int
	AverageLatency time.Duration
	TokensUsed     int
	ErrorCount     int
	SuccessRate    float64
}

// KeyMap defines all keyboard shortcuts
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Menu     key.Binding
	Mode     key.Binding
	Provider key.Binding
	Model    key.Binding
	Clear    key.Binding
	Copy     key.Binding
	Paste    key.Binding
	Execute  key.Binding
	Debug    key.Binding
	Stats    key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send/select"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
		Menu: key.NewBinding(
			key.WithKeys("ctrl+m"),
			key.WithHelp("ctrl+m", "toggle menu"),
		),
		Mode: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "switch mode"),
		),
		Provider: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "select provider"),
		),
		Model: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "select model"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear chat"),
		),
		Copy: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "copy"),
		),
		Paste: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("ctrl+v", "paste"),
		),
		Execute: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "execute code"),
		),
		Debug: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("f2", "debug mode"),
		),
		Stats: key.NewBinding(
			key.WithKeys("f3"),
			key.WithHelp("f3", "statistics"),
		),
		Help: key.NewBinding(
			key.WithKeys("f1", "?"),
			key.WithHelp("f1/?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+q", "esc"),
			key.WithHelp("ctrl+q/esc", "quit"),
		),
	}
}

// Professional color scheme
var (
	// Main palette - modern dark theme
	ColorPrimary   = lipgloss.Color("#00D9FF") // Cyan
	ColorSecondary = lipgloss.Color("#FF00FF") // Magenta
	ColorAccent    = lipgloss.Color("#00FF88") // Green
	ColorWarning   = lipgloss.Color("#FFB800") // Orange
	ColorError     = lipgloss.Color("#FF0066") // Red
	ColorSuccess   = lipgloss.Color("#00FF00") // Bright Green

	// Background colors
	ColorBg          = lipgloss.Color("#0A0E27") // Deep blue-black
	ColorBgAlt       = lipgloss.Color("#1A1E37") // Slightly lighter
	ColorBgHighlight = lipgloss.Color("#2A2E47") // Highlight

	// Text colors
	ColorText       = lipgloss.Color("#E0E0E0") // Light gray
	ColorTextMuted  = lipgloss.Color("#808080") // Muted gray
	ColorTextBright = lipgloss.Color("#FFFFFF") // White

	// Border colors
	ColorBorder      = lipgloss.Color("#3A3E57") // Border gray
	ColorBorderFocus = lipgloss.Color("#00D9FF") // Cyan focus
)

// Styles
var (
	// Main container style
	AppStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText)

	// Header style with gradient effect
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBgAlt).
			Padding(1, 2).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder).
			Align(lipgloss.Center)

	// Mode indicator styles
	ModeOperatorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgHighlight).
				Padding(0, 2).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent)

	ModeCoderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Background(ColorBgHighlight).
			Padding(0, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary)

	// Menu styles
	MenuStyle = lipgloss.NewStyle().
			Background(ColorBgAlt).
			Foreground(ColorText).
			Padding(1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			MarginRight(2)

	MenuItemStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 2)

	MenuItemSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Background(ColorBgHighlight).
				Bold(true).
				Padding(0, 2)

	// Chat styles
	ChatContainerStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(1)

	UserMessageStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				BorderStyle(lipgloss.NormalBorder()).
				BorderLeft(true).
				BorderForeground(ColorAccent).
				PaddingLeft(1).
				MarginBottom(1)

	AIMessageStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true).
			BorderForeground(ColorPrimary).
			PaddingLeft(1).
			MarginBottom(1)

	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				BorderStyle(lipgloss.NormalBorder()).
				BorderLeft(true).
				BorderForeground(ColorWarning).
				PaddingLeft(1).
				MarginBottom(1)

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorBgAlt).
			Foreground(ColorText).
			Padding(0, 1)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorderFocus).
			Padding(0, 1)

	// Tool panel styles
	ToolPanelStyle = lipgloss.NewStyle().
			Background(ColorBgAlt).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1).
			MarginTop(1)
)

// NewProfessionalModel creates a new professional TUI model
func NewProfessionalModel(cfg *config.Config, tuiConfig *Config) *ProfessionalModel {
	// Initialize text area
	ta := textarea.New()
	ta.Placeholder = "💬 Type your message... (Ctrl+Enter to send)"
	ta.CharLimit = 5000
	ta.SetWidth(80)
	ta.SetHeight(4)
	ta.ShowLineNumbers = false
	ta.Focus()

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(ColorBgHighlight)
	ta.FocusedStyle.Base = InputStyle

	// Initialize viewport
	vp := viewport.New(80, 20)
	vp.HighPerformanceRendering = true

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Initialize help
	h := help.New()
	h.ShowAll = false

	// Get system info
	sysInfo := getSystemInfo()

	// Determine initial mode based on config or default
	initialMode := ModeOperator
	if tuiConfig != nil && tuiConfig.Profile == "coder" {
		initialMode = ModeCoder
	}

	// Create the model
	m := &ProfessionalModel{
		config:      cfg,
		mode:        initialMode,
		textarea:    ta,
		viewport:    vp,
		spinner:     s,
		help:        h,
		keymap:      DefaultKeyMap(),
		messages:    []ProMessage{},
		toolsActive: make(map[string]bool),
		systemInfo:  sysInfo,
		menuVisible: true,
		provider:    tuiConfig.Provider,
		model:       tuiConfig.Model,
	}

	// Initialize menu items based on mode
	m.updateMenuItems()

	// Create AI service
	if tuiConfig.Provider != "" && tuiConfig.Model != "" {
		m.initAIService()
	}

	// Add welcome message
	m.addSystemMessage(m.getWelcomeMessage())

	return m
}

// getSystemInfo gathers system information
func getSystemInfo() SystemInfo {
	wd, _ := os.Getwd()
	return SystemInfo{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPUCores:   runtime.NumCPU(),
		GoVersion:  runtime.Version(),
		WorkingDir: wd,
		Memory:     "N/A", // Would need additional library for memory info
	}
}

// Init initializes the model
func (m *ProfessionalModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		textarea.Blink,
		m.listenForActivity(),
	)
}

// Update handles messages
func (m *ProfessionalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Adjust component sizes
		menuWidth := 25
		if m.menuVisible {
			m.viewport.Width = msg.Width - menuWidth - 4
		} else {
			m.viewport.Width = msg.Width - 4
		}
		m.viewport.Height = msg.Height - 12 // Leave room for header, input, status
		m.textarea.SetWidth(m.viewport.Width)

		// Update viewport content
		m.updateViewport()

	case tea.KeyMsg:
		// Handle global keys first
		switch {
		case key.Matches(msg, m.keymap.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Menu):
			m.menuVisible = !m.menuVisible
			m.updateLayout()

		case key.Matches(msg, m.keymap.Mode):
			m.switchMode()
			m.updateMenuItems()
			m.addSystemMessage(fmt.Sprintf("🔄 Switched to %s mode", m.getModeString()))

		case key.Matches(msg, m.keymap.Provider):
			m.cycleProvider()
			m.updateMenuItems()
			m.addSystemMessage(fmt.Sprintf("🔁 Provider set to %s (model: %s)", m.provider, m.model))

		case key.Matches(msg, m.keymap.Model):
			m.cycleModel()
			m.addSystemMessage(fmt.Sprintf("🧠 Model set to %s", m.model))

		case key.Matches(msg, m.keymap.Clear):
			m.clearChat()

		case key.Matches(msg, m.keymap.Help):
			// Toggle help display
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keymap.Enter):
			if m.menuVisible && !m.textarea.Focused() {
				// Execute menu item
				if m.menuIndex < len(m.menuItems) {
					cmd := m.menuItems[m.menuIndex].Action()
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			} else if m.textarea.Focused() && strings.TrimSpace(m.textarea.Value()) != "" {
				// Send message
				cmd := m.sendMessage()
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case key.Matches(msg, m.keymap.Up):
			if m.menuVisible && !m.textarea.Focused() {
				m.menuIndex--
				if m.menuIndex < 0 {
					m.menuIndex = len(m.menuItems) - 1
				}
			}

		case key.Matches(msg, m.keymap.Down):
			if m.menuVisible && !m.textarea.Focused() {
				m.menuIndex++
				if m.menuIndex >= len(m.menuItems) {
					m.menuIndex = 0
				}
			}

		case key.Matches(msg, m.keymap.Tab):
			// Toggle focus between menu and input
			if m.menuVisible {
				if m.textarea.Focused() {
					m.textarea.Blur()
				} else {
					m.textarea.Focus()
				}
			}
		}

		// Update textarea if focused
		if m.textarea.Focused() {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case streamMsg:
		// Handle streaming response
		if msg.Error != nil {
			m.addErrorMessage(fmt.Sprintf("Error: %v", msg.Error))
			m.streaming = false
			m.streamChan = nil
			if m.cancelStream != nil {
				m.cancelStream()
				m.cancelStream = nil
			}
		} else if msg.Done {
			m.streaming = false
			m.streamChan = nil
			if m.cancelStream != nil {
				m.cancelStream()
				m.cancelStream = nil
			}
			m.updateMetrics()
		} else {
			// Append to last AI message
			if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
				m.messages[len(m.messages)-1].Content += msg.Content
				m.updateViewport()
			}
			// Read next chunk
			return m, m.readNextChunk()
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m *ProfessionalModel) View() string {
	if !m.ready {
		return "Initializing GOLLM Professional Interface..."
	}

	if m.quitting {
		return HeaderStyle.Render("👋 Thanks for using GOLLM Professional! Goodbye!")
	}

	var sections []string

	// Header
	header := m.renderHeader()
	sections = append(sections, header)

	// Main content area
	var mainContent string
	if m.menuVisible {
		// Menu + Chat side by side
		menu := m.renderMenu()
		chat := m.renderChat()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, menu, chat)
	} else {
		// Full width chat
		mainContent = m.renderChat()
	}
	sections = append(sections, mainContent)

	// Input area
	input := m.renderInput()
	sections = append(sections, input)

	// Status bar
	status := m.renderStatusBar()
	sections = append(sections, status)

	// Help (if visible)
	if m.help.ShowAll {
		helpView := m.renderHelp()
		sections = append(sections, helpView)
	}

	return AppStyle.Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

// renderHeader renders the header section
func (m *ProfessionalModel) renderHeader() string {
	title := "🚀 GOLLM PROFESSIONAL INTERFACE 🚀"

	// Mode indicator
	var modeIndicator string
	if m.mode == ModeOperator {
		modeIndicator = ModeOperatorStyle.Render("⚙️ PC OPERATOR MODE")
	} else {
		modeIndicator = ModeCoderStyle.Render("💻 CODER MODE")
	}

	// Provider info
	providerInfo := lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Render(fmt.Sprintf("Provider: %s | Model: %s", m.provider, m.model))

	// Combine elements
	header := lipgloss.JoinVertical(lipgloss.Center,
		HeaderStyle.Render(title),
		lipgloss.JoinHorizontal(lipgloss.Center, modeIndicator, "  ", providerInfo),
	)

	return header
}

// renderMenu renders the side menu
func (m *ProfessionalModel) renderMenu() string {
	var items []string

	for i, item := range m.menuItems {
		var style lipgloss.Style
		if i == m.menuIndex && !m.textarea.Focused() {
			style = MenuItemSelectedStyle
		} else {
			style = MenuItemStyle
		}

		itemText := fmt.Sprintf("%s %s", item.Icon, item.Title)
		if item.Hotkey != "" {
			itemText += fmt.Sprintf(" [%s]", item.Hotkey)
		}

		items = append(items, style.Render(itemText))
	}

	menuContent := lipgloss.JoinVertical(lipgloss.Left, items...)
	return MenuStyle.Width(25).Height(m.viewport.Height).Render(menuContent)
}

// renderChat renders the chat area
func (m *ProfessionalModel) renderChat() string {
	return ChatContainerStyle.
		Width(m.viewport.Width).
		Height(m.viewport.Height).
		Render(m.viewport.View())
}

// renderInput renders the input area
func (m *ProfessionalModel) renderInput() string {
	var inputIndicator string
	if m.streaming {
		inputIndicator = m.spinner.View() + " AI is thinking..."
	} else if m.textarea.Focused() {
		inputIndicator = "✏️ Typing..."
	} else {
		inputIndicator = "💤 Press Tab to focus input"
	}

	input := InputStyle.Width(m.viewport.Width).Render(m.textarea.View())
	return lipgloss.JoinVertical(lipgloss.Left, inputIndicator, input)
}

// renderStatusBar renders the status bar
func (m *ProfessionalModel) renderStatusBar() string {
	// System info
	sysInfo := fmt.Sprintf("OS: %s/%s | CPUs: %d | Go: %s",
		m.systemInfo.OS, m.systemInfo.Arch, m.systemInfo.CPUCores, m.systemInfo.GoVersion)

	// Metrics
	metrics := fmt.Sprintf("Requests: %d | Avg Latency: %v | Tokens: %d | Success: %.1f%%",
		m.metrics.TotalRequests, m.metrics.AverageLatency, m.metrics.TokensUsed, m.metrics.SuccessRate)

	// Active tools
	var activeTools []string
	for tool, active := range m.toolsActive {
		if active {
			activeTools = append(activeTools, tool)
		}
	}
	toolsStr := "None"
	if len(activeTools) > 0 {
		toolsStr = strings.Join(activeTools, ", ")
	}
	tools := fmt.Sprintf("Active Tools: %s", toolsStr)

	status := lipgloss.JoinVertical(lipgloss.Left,
		StatusBarStyle.Render(sysInfo),
		StatusBarStyle.Render(metrics),
		StatusBarStyle.Render(tools),
	)

	return status
}

// renderHelp renders the help section
func (m *ProfessionalModel) renderHelp() string {
	// Create a help list of key bindings
	keybinds := []key.Binding{
		m.keymap.Mode,
		m.keymap.Menu,
		m.keymap.Clear,
		m.keymap.Provider,
		m.keymap.Model,
		m.keymap.Help,
		m.keymap.Quit,
	}

	helpText := []string{
		"🎮 Keyboard Shortcuts:",
		"━━━━━━━━━━━━━━━━━━━━━",
	}

	for _, kb := range keybinds {
		helpText = append(helpText, fmt.Sprintf("  %s - %s", kb.Help().Key, kb.Help().Desc))
	}

	return lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Padding(1).
		Render(strings.Join(helpText, "\n"))
}

// Helper methods

func (m *ProfessionalModel) cycleProvider() {
	names := m.config.ListProviders()
	if len(names) == 0 {
		m.addErrorMessage("No providers configured")
		return
	}
	if m.provider == "" {
		m.provider = names[0]
	} else {
		idx := 0
		for i, name := range names {
			if name == m.provider {
				idx = i
				break
			}
		}
		idx = (idx + 1) % len(names)
		m.provider = names[idx]
	}
	// Set model to default for new provider if available
	if pc, _, err := m.config.GetProvider(m.provider); err == nil {
		if pc.DefaultModel != "" {
			m.model = pc.DefaultModel
		}
	}
	m.initAIService()
	m.updateViewport()
}

func (m *ProfessionalModel) cycleModel() {
	if m.provider == "" {
		m.addErrorMessage("No provider selected")
		return
	}
	pc, _, err := m.config.GetProvider(m.provider)
	if err != nil {
		m.addErrorMessage(fmt.Sprintf("Failed to get provider config: %v", err))
		return
	}
	models := pc.Models
	// If config doesn't specify models, try provider GetModels
	if len(models) == 0 && m.aiService != nil {
		if lister, ok := m.aiService.provider.(core.ModelLister); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if list, err := lister.GetModels(ctx); err == nil {
				for _, md := range list {
					models = append(models, md.ID)
				}
			}
		}
	}
	if len(models) == 0 {
		m.addErrorMessage("Model list not available; set model manually in config")
		return
	}
	// Cycle through models
	if m.model == "" {
		m.model = models[0]
	} else {
		idx := 0
		for i, id := range models {
			if id == m.model {
				idx = i
				break
			}
		}
		idx = (idx + 1) % len(models)
		m.model = models[idx]
	}
	m.initAIService()
	m.updateViewport()
}

func (m *ProfessionalModel) switchMode() {
	if m.mode == ModeOperator {
		m.mode = ModeCoder
	} else {
		m.mode = ModeOperator
	}
}

func (m *ProfessionalModel) getModeString() string {
	if m.mode == ModeOperator {
		return "PC Operator"
	}
	return "Coder"
}

func (m *ProfessionalModel) updateMenuItems() {
	if m.mode == ModeOperator {
		m.menuItems = m.getOperatorMenuItems()
	} else {
		m.menuItems = m.getCoderMenuItems()
	}
}

func (m *ProfessionalModel) getOperatorMenuItems() []ProMenuItem {
	return []ProMenuItem{
		{
			Title:       "System Info",
			Description: "Show system information",
			Icon:        "📊",
			Hotkey:      "Alt+S",
			Action:      func() tea.Cmd { return m.showSystemInfo() },
		},
		{
			Title:       "Process Manager",
			Description: "Manage running processes",
			Icon:        "⚙️",
			Hotkey:      "Alt+P",
			Action:      func() tea.Cmd { return m.openProcessManager() },
		},
		{
			Title:       "Network Tools",
			Description: "Network diagnostics",
			Icon:        "🌐",
			Hotkey:      "Alt+N",
			Action:      func() tea.Cmd { return m.openNetworkTools() },
		},
		{
			Title:       "File Manager",
			Description: "Browse and manage files",
			Icon:        "📁",
			Hotkey:      "Alt+F",
			Action:      func() tea.Cmd { return m.openFileManager() },
		},
		{
			Title:       "Terminal",
			Description: "Execute shell commands",
			Icon:        "💻",
			Hotkey:      "Alt+T",
			Action:      func() tea.Cmd { return m.openTerminal() },
		},
		{
			Title:       "Automation",
			Description: "Create automation scripts",
			Icon:        "🤖",
			Hotkey:      "Alt+A",
			Action:      func() tea.Cmd { return m.openAutomation() },
		},
		{
			Title:       "Monitoring",
			Description: "System monitoring",
			Icon:        "📈",
			Hotkey:      "Alt+M",
			Action:      func() tea.Cmd { return m.openMonitoring() },
		},
		{
			Title:       "Security",
			Description: "Security tools",
			Icon:        "🔒",
			Hotkey:      "Alt+X",
			Action:      func() tea.Cmd { return m.openSecurity() },
		},
	}
}

func (m *ProfessionalModel) getCoderMenuItems() []ProMenuItem {
	return []ProMenuItem{
		{
			Title:       "Code Generator",
			Description: "Generate code snippets",
			Icon:        "✨",
			Hotkey:      "Alt+G",
			Action:      func() tea.Cmd { return m.openCodeGenerator() },
		},
		{
			Title:       "Refactor",
			Description: "Refactor existing code",
			Icon:        "🔧",
			Hotkey:      "Alt+R",
			Action:      func() tea.Cmd { return m.openRefactor() },
		},
		{
			Title:       "Test Suite",
			Description: "Generate and run tests",
			Icon:        "🧪",
			Hotkey:      "Alt+T",
			Action:      func() tea.Cmd { return m.openTestSuite() },
		},
		{
			Title:       "Documentation",
			Description: "Generate documentation",
			Icon:        "📚",
			Hotkey:      "Alt+D",
			Action:      func() tea.Cmd { return m.openDocGenerator() },
		},
		{
			Title:       "Debug Helper",
			Description: "Debug assistance",
			Icon:        "🐛",
			Hotkey:      "Alt+B",
			Action:      func() tea.Cmd { return m.openDebugHelper() },
		},
		{
			Title:       "Performance",
			Description: "Performance analysis",
			Icon:        "⚡",
			Hotkey:      "Alt+P",
			Action:      func() tea.Cmd { return m.openPerformanceAnalyzer() },
		},
		{
			Title:       "Git Helper",
			Description: "Git operations",
			Icon:        "🌿",
			Hotkey:      "Alt+V",
			Action:      func() tea.Cmd { return m.openGitHelper() },
		},
		{
			Title:       "API Client",
			Description: "Test APIs",
			Icon:        "🔌",
			Hotkey:      "Alt+A",
			Action:      func() tea.Cmd { return m.openAPIClient() },
		},
		{
			Title:       "Code Review",
			Description: "AI code review",
			Icon:        "👁️",
			Hotkey:      "Alt+C",
			Action:      func() tea.Cmd { return m.openCodeReview() },
		},
		{
			Title:       "Architecture",
			Description: "System design helper",
			Icon:        "🏗️",
			Hotkey:      "Alt+H",
			Action:      func() tea.Cmd { return m.openArchitectureHelper() },
		},
	}
}

func (m *ProfessionalModel) updateLayout() {
	menuWidth := 0
	if m.menuVisible {
		menuWidth = 25
	}
	m.viewport.Width = m.width - menuWidth - 4
	m.textarea.SetWidth(m.viewport.Width)
	m.updateViewport()
}

func (m *ProfessionalModel) clearChat() {
	m.messages = []ProMessage{}
	m.addSystemMessage("💬 Chat cleared. Ready for new conversation!")
	m.updateViewport()
}

func (m *ProfessionalModel) sendMessage() tea.Cmd {
	userMsg := strings.TrimSpace(m.textarea.Value())
	if userMsg == "" {
		return nil
	}

	// Add user message
	m.addUserMessage(userMsg)

	// Clear input
	m.textarea.SetValue("")

	// Start streaming
	m.streaming = true

	// Send to AI
	if m.aiService != nil {
		return m.startStreaming(userMsg)
	}

	// Fallback if no AI service
	m.addSystemMessage("⚠️ No AI provider configured. Please configure a provider first.")
	m.streaming = false
	return nil
}

func (m *ProfessionalModel) addUserMessage(content string) {
	m.messages = append(m.messages, ProMessage{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
		Provider:  m.provider,
		Model:     m.model,
	})
	m.updateViewport()
}

func (m *ProfessionalModel) addAIMessage(content string) {
	m.messages = append(m.messages, ProMessage{
		Role:      "assistant",
		Content:   content,
		Timestamp: time.Now(),
		Provider:  m.provider,
		Model:     m.model,
	})
	m.updateViewport()
}

func (m *ProfessionalModel) addSystemMessage(content string) {
	m.messages = append(m.messages, ProMessage{
		Role:      "system",
		Content:   content,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}

func (m *ProfessionalModel) addErrorMessage(content string) {
	m.messages = append(m.messages, ProMessage{
		Role:      "error",
		Content:   content,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}

func (m *ProfessionalModel) updateViewport() {
	var content []string

	for _, msg := range m.messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		var style lipgloss.Style
		var prefix string

		switch msg.Role {
		case "user":
			style = UserMessageStyle
			prefix = fmt.Sprintf("👤 You [%s]:", timestamp)
		case "assistant":
			style = AIMessageStyle
			prefix = fmt.Sprintf("🤖 AI (%s/%s) [%s]:", msg.Provider, msg.Model, timestamp)
		case "system":
			style = SystemMessageStyle
			prefix = fmt.Sprintf("ℹ️ System [%s]:", timestamp)
		case "error":
			style = lipgloss.NewStyle().Foreground(ColorError)
			prefix = fmt.Sprintf("❌ Error [%s]:", timestamp)
		}

		msgContent := fmt.Sprintf("%s\n%s", prefix, msg.Content)
		content = append(content, style.Render(msgContent))
	}

	m.viewport.SetContent(strings.Join(content, "\n"))
	m.viewport.GotoBottom()
}

func (m *ProfessionalModel) getWelcomeMessage() string {
	modeStr := m.getModeString()
	return fmt.Sprintf(`
🎯 Welcome to GOLLM Professional Interface!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Current Mode: %s
🤖 Provider: %s
🧠 Model: %s
💻 System: %s/%s

⌨️ Quick Keys:
• Ctrl+O: Switch mode
• Ctrl+M: Toggle menu
• Ctrl+L: Clear chat
• F1: Show help
• Tab: Switch focus

Ready to assist you! Type your message below or select a tool from the menu.
`, modeStr, m.provider, m.model, m.systemInfo.OS, m.systemInfo.Arch)
}

func (m *ProfessionalModel) updateMetrics() {
	m.metrics.TotalRequests++
	// Calculate other metrics based on responses
	if m.metrics.TotalRequests > 0 {
		m.metrics.SuccessRate = float64(m.metrics.TotalRequests-m.metrics.ErrorCount) / float64(m.metrics.TotalRequests) * 100
	}
}

func (m *ProfessionalModel) listenForActivity() tea.Cmd {
	return func() tea.Msg {
		// Periodic update for metrics or other background tasks
		time.Sleep(5 * time.Second)
		return nil
	}
}

// Tool action implementations (stubs for now)

func (m *ProfessionalModel) showSystemInfo() tea.Cmd {
	info := fmt.Sprintf(`
📊 System Information
━━━━━━━━━━━━━━━━━━━━
OS: %s
Architecture: %s
CPU Cores: %d
Go Version: %s
Working Directory: %s
`, m.systemInfo.OS, m.systemInfo.Arch, m.systemInfo.CPUCores, m.systemInfo.GoVersion, m.systemInfo.WorkingDir)

	m.addSystemMessage(info)
	return nil
}

func (m *ProfessionalModel) openProcessManager() tea.Cmd {
	m.addSystemMessage("⚙️ Process Manager activated. Use 'ps', 'top', or 'htop' commands.")
	m.toolsActive["ProcessManager"] = true
	return nil
}

func (m *ProfessionalModel) openNetworkTools() tea.Cmd {
	m.addSystemMessage("🌐 Network Tools activated. Available: ping, traceroute, netstat, ss, nmap")
	m.toolsActive["NetworkTools"] = true
	return nil
}

func (m *ProfessionalModel) openFileManager() tea.Cmd {
	m.addSystemMessage("📁 File Manager activated. Use ls, cd, find, grep commands.")
	m.toolsActive["FileManager"] = true
	return nil
}

func (m *ProfessionalModel) openTerminal() tea.Cmd {
	m.addSystemMessage("💻 Terminal mode activated. You can execute any shell command.")
	m.toolsActive["Terminal"] = true
	return nil
}

func (m *ProfessionalModel) openAutomation() tea.Cmd {
	m.addSystemMessage("🤖 Automation Builder activated. Create bash scripts, systemd services, or cron jobs.")
	m.toolsActive["Automation"] = true
	return nil
}

func (m *ProfessionalModel) openMonitoring() tea.Cmd {
	m.addSystemMessage("📈 System Monitoring activated. Tracking CPU, Memory, Disk, Network usage.")
	m.toolsActive["Monitoring"] = true
	return nil
}

func (m *ProfessionalModel) openSecurity() tea.Cmd {
	m.addSystemMessage("🔒 Security Tools activated. Available: firewall config, port scanning, log analysis.")
	m.toolsActive["Security"] = true
	return nil
}

func (m *ProfessionalModel) openCodeGenerator() tea.Cmd {
	m.addSystemMessage("✨ Code Generator activated. Describe what you want to build!")
	m.toolsActive["CodeGenerator"] = true
	return nil
}

func (m *ProfessionalModel) openRefactor() tea.Cmd {
	m.addSystemMessage("🔧 Refactoring Assistant activated. Paste your code for improvements.")
	m.toolsActive["Refactor"] = true
	return nil
}

func (m *ProfessionalModel) openTestSuite() tea.Cmd {
	m.addSystemMessage("🧪 Test Suite activated. Generate unit tests, integration tests, or test data.")
	m.toolsActive["TestSuite"] = true
	return nil
}

func (m *ProfessionalModel) openDocGenerator() tea.Cmd {
	m.addSystemMessage("📚 Documentation Generator activated. Create README, API docs, or code comments.")
	m.toolsActive["DocGenerator"] = true
	return nil
}

func (m *ProfessionalModel) openDebugHelper() tea.Cmd {
	m.addSystemMessage("🐛 Debug Helper activated. Paste your error or describe the issue.")
	m.toolsActive["DebugHelper"] = true
	return nil
}

func (m *ProfessionalModel) openPerformanceAnalyzer() tea.Cmd {
	m.addSystemMessage("⚡ Performance Analyzer activated. Analyze code for bottlenecks and optimizations.")
	m.toolsActive["Performance"] = true
	return nil
}

func (m *ProfessionalModel) openGitHelper() tea.Cmd {
	m.addSystemMessage("🌿 Git Helper activated. Assistance with commits, branches, merges, and workflows.")
	m.toolsActive["GitHelper"] = true
	return nil
}

func (m *ProfessionalModel) openAPIClient() tea.Cmd {
	m.addSystemMessage("🔌 API Client activated. Test REST, GraphQL, or gRPC endpoints.")
	m.toolsActive["APIClient"] = true
	return nil
}

func (m *ProfessionalModel) openCodeReview() tea.Cmd {
	m.addSystemMessage("👁️ Code Review activated. Paste code for AI-powered review and suggestions.")
	m.toolsActive["CodeReview"] = true
	return nil
}

func (m *ProfessionalModel) openArchitectureHelper() tea.Cmd {
	m.addSystemMessage("🏗️ Architecture Helper activated. Design patterns, system design, and best practices.")
	m.toolsActive["Architecture"] = true
	return nil
}

// AI Integration methods

func (m *ProfessionalModel) initAIService() {
	if m.config == nil {
		return
	}

	// Get provider config
	providerConfig, _, err := m.config.GetProvider(m.provider)
	if err != nil {
		m.addErrorMessage(fmt.Sprintf("Failed to get provider config: %v", err))
		return
	}

	// Create provider
	coreConfig := providerConfig.ToProviderConfig()
	provider, err := core.CreateProviderFromConfig(providerConfig.Type, coreConfig)
	if err != nil {
		m.addErrorMessage(fmt.Sprintf("Failed to create provider: %v", err))
		return
	}

	// Create AI service
	m.aiService = &AIService{
		provider: provider,
		config:   m.config,
	}
}

// startStreaming initializes streaming for a user message
func (m *ProfessionalModel) startStreaming(message string) tea.Cmd {
	return func() tea.Msg {
		if m.aiService == nil {
			return streamMsg{Error: fmt.Errorf("AI service not initialized")}
		}

		// Build request based on mode
		systemPrompt := ""
		if m.mode == ModeOperator {
			systemPrompt = "You are a system administrator and DevOps expert. Help with system commands, automation, and infrastructure management. Be precise and security-conscious."
		} else {
			systemPrompt = "You are an expert programmer and software architect. Provide clean, efficient, well-documented code with best practices. Focus on maintainability and performance."
		}

		// Add context from active tools
		if len(m.toolsActive) > 0 {
			var activeTools []string
			for tool := range m.toolsActive {
				activeTools = append(activeTools, tool)
			}
			systemPrompt += fmt.Sprintf("\n\nActive tools: %s", strings.Join(activeTools, ", "))
		}

		request := &core.CompletionRequest{
			Model: m.model,
			Messages: []core.Message{
				{Role: core.RoleSystem, Content: systemPrompt},
				{Role: core.RoleUser, Content: message},
			},
			Stream:      true,
			MaxTokens:   intPtr(4096),
			Temperature: float64Ptr(0.7),
		}

		// Start streaming
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelStream = cancel
		streamer, ok := m.aiService.provider.(core.Streamer)
		if !ok {
			// Non-streaming fallback
			resp, err := m.aiService.provider.CreateCompletion(ctx, request)
			if err != nil {
				return streamMsg{Error: err}
			}
			return streamMsg{Content: resp.Choices[0].Message.Content, Done: true}
		}

		// Create AI message placeholder
		m.addAIMessage("")

		// Stream response
		chunks, err := streamer.StreamCompletion(ctx, request)
		if err != nil {
			return streamMsg{Error: err}
		}
		m.streamChan = chunks

		// Return a command that reads the next chunk
		return m.readNextChunk()
	}
}

// Helper functions

// readNextChunk reads the next streaming chunk and returns it as a tea.Msg
func (m *ProfessionalModel) readNextChunk() tea.Cmd {
	return func() tea.Msg {
		if m.streamChan == nil {
			return streamMsg{Done: true}
		}
		chunk, ok := <-m.streamChan
		if !ok {
			return streamMsg{Done: true}
		}
		if chunk.Error != nil {
			return streamMsg{Error: chunk.Error}
		}
		var content string
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content = chunk.Choices[0].Delta.Content
		}
		return streamMsg{Content: content, Done: chunk.Done}
	}
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

// RunProfessionalTUI launches the professional TUI
func RunProfessionalTUI(cfg *config.Config, tuiConfig *Config) error {
	model := NewProfessionalModel(cfg, tuiConfig)
	program := tea.NewProgram(model, tea.WithAltScreen())

	_, err := program.Run()
	return err
}
