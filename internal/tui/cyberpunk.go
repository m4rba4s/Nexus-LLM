// Package tui provides a cyberpunk-style terminal user interface for GOLLM
// with enhanced visuals, animations, and interactive features.
package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/gollm/internal/config"
	"github.com/yourusername/gollm/internal/core"
)

// Config represents TUI-specific configuration
type Config struct {
    Provider    string
    Model       string
    Profile     string
    Theme       string
    NoMatrix    bool
    Debug       bool
    Performance bool
    Simple      bool
    // Optional initial system instruction for the session
    SystemMessage string
}

// CyberpunkTheme defines the color scheme and styling for the TUI
type CyberpunkTheme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Surface    lipgloss.Color
	Error      lipgloss.Color
	Warning    lipgloss.Color
	Success    lipgloss.Color
	Text       lipgloss.Color
	TextMuted  lipgloss.Color
	Border     lipgloss.Color
	Glow       lipgloss.Color
}

// DefaultCyberpunkTheme returns the default cyberpunk color scheme
func DefaultCyberpunkTheme() CyberpunkTheme {
	return CyberpunkTheme{
		Primary:    lipgloss.Color("#00ff41"), // Matrix green
		Secondary:  lipgloss.Color("#0066ff"), // Electric blue
		Accent:     lipgloss.Color("#ff00ff"), // Neon pink
		Background: lipgloss.Color("#000000"), // Pure black
		Surface:    lipgloss.Color("#0a0a0a"), // Dark surface
		Error:      lipgloss.Color("#ff3366"), // Neon red
		Warning:    lipgloss.Color("#ffaa00"), // Neon orange
		Success:    lipgloss.Color("#00ff88"), // Neon green
		Text:       lipgloss.Color("#ffffff"), // Bright white
		TextMuted:  lipgloss.Color("#888888"), // Muted gray
		Border:     lipgloss.Color("#333333"), // Dark border
		Glow:       lipgloss.Color("#00ffaa"), // Glow effect
	}
}

// Model represents the state of the cyberpunk TUI
type Model struct {
	// UI Components
	textInput textinput.Model
	viewport  viewport.Model
	spinner   spinner.Model

	// Configuration
	config   *config.Config
	provider core.Provider
	theme    CyberpunkTheme

	// State
	messages     []CyberpunkMessage
	history      []string
	isStreaming  bool
	isLoading    bool
	currentModel string
	tokenCount   int
	sessionStart time.Time

	// Animation state
	frame       int
	glitchChars []rune
	matrixRain  []MatrixColumn

	// UI dimensions
	width  int
	height int

	// Modes
	multilineMode bool
	debugMode     bool
	showStats     bool
}

// CyberpunkMessage represents a chat message with metadata
type CyberpunkMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
	Tokens    int
	Latency   time.Duration
	IsError   bool
}

// MatrixColumn represents a column of falling characters
type MatrixColumn struct {
	X         int
	Y         float64
	Length    int
	Speed     float64
	Chars     []rune
	Intensity []float64
}

// cyberpunkTickMsg is sent by the timer for animations
type cyberpunkTickMsg time.Time

// streamMsg represents a streaming response chunk
type streamMsg struct {
	Content string
	Done    bool
	Error   error
}

// NewModel creates a new cyberpunk TUI model
func NewModel(cfg *config.Config) *Model {
	theme := DefaultCyberpunkTheme()

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "▶ Enter your message..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 80

	// Style the input with cyberpunk theme
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.TextMuted)

	// Initialize viewport for messages
	vp := viewport.New(80, 20)
	vp.HighPerformanceRendering = true

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Pulse
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	// Initialize matrix rain with bounds checking
	matrixRain := make([]MatrixColumn, 20)
	for i := range matrixRain {
		charLength := 15
		if charLength <= 0 {
			charLength = 10
		}

		matrixRain[i] = MatrixColumn{
			X:         i * 4,
			Y:         float64(rand.Intn(10) + 1), // Ensure positive
			Length:    5 + rand.Intn(10),
			Speed:     0.1 + rand.Float64()*0.3,
			Chars:     generateMatrixChars(charLength),
			Intensity: make([]float64, charLength),
		}

		// Initialize intensity values with bounds checking
		for j := range matrixRain[i].Intensity {
			if j < len(matrixRain[i].Intensity) {
				matrixRain[i].Intensity[j] = rand.Float64()
			}
		}
	}

	return &Model{
		textInput:    ti,
		viewport:     vp,
		spinner:      s,
		config:       cfg,
		theme:        theme,
		messages:     []CyberpunkMessage{},
		history:      []string{},
		sessionStart: time.Now(),
		glitchChars:  []rune("▓▒░█▄▀▐│┤╡╢╖╗╝╜╛┐└┴┬├─┼╞╟╚╔╩╦╠═╬╧╨╤╥╙╘╒╓╫╪┘┌"),
		matrixRain:   matrixRain,
	}
}

// generateMatrixChars creates random matrix-style characters
func generateMatrixChars(length int) []rune {
	chars := []rune("01ﾊﾐﾋｰｳｼﾅﾓﾆｻﾜﾂｵﾘｱﾎﾃﾏｹﾒｴｶｷﾑﾕﾗｾﾈｽﾀﾇﾍ日十月火水木金土一二三四五六七八九十")

	// Ensure positive length
	if length <= 0 {
		length = 10
	}

	result := make([]rune, length)
	if len(chars) == 0 {
		// Fallback characters
		for i := range result {
			result[i] = '▓'
		}
		return result
	}

	for i := range result {
		if len(chars) > 0 {
			result[i] = chars[rand.Intn(len(chars))]
		} else {
			result[i] = '▓'
		}
	}
	return result
}

// Init initializes the TUI model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		m.tickAnimation(),
	)
}

// tickAnimation creates a recurring timer for animations
func (m Model) tickAnimation() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return cyberpunkTickMsg(t)
	})
}

// Update handles messages and updates the model state
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 8
		m.textInput.Width = msg.Width - 8
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "enter":
			if m.textInput.Value() != "" {
				return m.handleUserInput()
			}

		case "ctrl+l":
			m.messages = []CyberpunkMessage{}
			m.viewport.SetContent("")
			return m, nil

		case "f1":
			m.showStats = !m.showStats
			return m, nil

		case "f2":
			m.debugMode = !m.debugMode
			return m, nil

		case "f3":
			m.multilineMode = !m.multilineMode
			return m, nil
		}

	case cyberpunkTickMsg:
		m.frame++
		m.updateAnimations()
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd, m.tickAnimation())

	case streamMsg:
		return m.handleStreamResponse(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update text input
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleUserInput processes user input and sends it to the LLM
func (m Model) handleUserInput() (Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())

	// Add to history
	m.history = append(m.history, input)

	// Add user message
	userMsg := CyberpunkMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	// Clear input
	m.textInput.SetValue("")

	// Set loading state
	m.isLoading = true
	m.isStreaming = true

	// Update viewport
	m.updateViewport()

	// Send message to LLM
	return m, m.sendMessage(input)
}

// sendMessage sends a message to the configured LLM provider
func (m Model) sendMessage(input string) tea.Cmd {
	return func() tea.Msg {
		// For now, return a mock response
		// In real implementation, this would call the actual provider
		time.Sleep(500 * time.Millisecond) // Simulate network delay

		responses := []string{
			"🔮 Initializing neural pathways...",
			"⚡ Processing quantum entanglement...",
			"🌐 Accessing the collective consciousness...",
			"🧠 Synthesizing digital wisdom...",
			"✨ Reality.exe has stopped working. Just kidding! Here's your response:",
			"Hello! I'm your AI assistant powered by GOLLM. How can I help you today?",
		}

		response := responses[rand.Intn(len(responses))]

		return streamMsg{
			Content: response,
			Done:    true,
			Error:   nil,
		}
	}
}

// handleStreamResponse handles streaming responses from the LLM
func (m Model) handleStreamResponse(msg streamMsg) (Model, tea.Cmd) {
	if msg.Error != nil {
		// Handle error
		errorMsg := CyberpunkMessage{
			Role:      "assistant",
			Content:   fmt.Sprintf("❌ Error: %s", msg.Error.Error()),
			Timestamp: time.Now(),
			IsError:   true,
		}
		m.messages = append(m.messages, errorMsg)
	} else {
		// Handle successful response
		assistantMsg := CyberpunkMessage{
			Role:      "assistant",
			Content:   msg.Content,
			Timestamp: time.Now(),
			Latency:   time.Since(m.sessionStart),
		}
		m.messages = append(m.messages, assistantMsg)
	}

	if msg.Done {
		m.isLoading = false
		m.isStreaming = false
	}

	m.updateViewport()
	return m, nil
}

// updateAnimations updates the animated elements
func (m *Model) updateAnimations() {
	// Update matrix rain
	for i := range m.matrixRain {
		col := &m.matrixRain[i]
		col.Y += col.Speed

		if col.Y > float64(m.height) {
			col.Y = float64(-col.Length)
			if m.width > 0 {
				col.X = rand.Intn(m.width)
			} else {
				col.X = 0
			}
			col.Speed = 0.1 + rand.Float64()*0.3
		}

		// Update character intensities
		for j := range col.Intensity {
			col.Intensity[j] = rand.Float64()
		}

		// Occasionally change characters
		if m.frame%10 == 0 {
			for j := range col.Chars {
				if rand.Float64() < 0.1 {
					newChars := generateMatrixChars(1)
					if len(newChars) > 0 {
						col.Chars[j] = newChars[0]
					}
				}
			}
		}
	}
}

// updateViewport refreshes the viewport content with formatted messages
func (m *Model) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		content.WriteString(m.formatMessage(msg))
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// formatMessage formats a message with cyberpunk styling
func (m *Model) formatMessage(msg CyberpunkMessage) string {
	timestamp := msg.Timestamp.Format("15:04:05")

	var roleStyle lipgloss.Style
	var prefix string

	switch msg.Role {
	case "user":
		roleStyle = lipgloss.NewStyle().
			Foreground(m.theme.Secondary).
			Bold(true)
		prefix = "▶ USER"
	case "assistant":
		if msg.IsError {
			roleStyle = lipgloss.NewStyle().
				Foreground(m.theme.Error).
				Bold(true)
			prefix = "⚠ ERROR"
		} else {
			roleStyle = lipgloss.NewStyle().
				Foreground(m.theme.Primary).
				Bold(true)
			prefix = "◀ AI"
		}
	}

	timeStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Faint(true)

	contentStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		PaddingLeft(2)

	header := fmt.Sprintf("%s [%s]",
		roleStyle.Render(prefix),
		timeStyle.Render(timestamp))

	if m.showStats && msg.Latency > 0 {
		header += fmt.Sprintf(" %s",
			timeStyle.Render(fmt.Sprintf("(%s)", msg.Latency.Round(time.Millisecond))))
	}

	content := contentStyle.Render(msg.Content)

	return header + "\n" + content
}

// View renders the TUI
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing cyberpunk interface..."
	}

	// Create the main layout
	var sections []string

	// Header with animated title
	header := m.renderHeader()
	sections = append(sections, header)

	// Main chat area with matrix background
	chatArea := m.renderChatArea()
	sections = append(sections, chatArea)

	// Input area
	inputArea := m.renderInputArea()
	sections = append(sections, inputArea)

	// Status bar
	statusBar := m.renderStatusBar()
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader creates the animated cyberpunk header
func (m Model) renderHeader() string {
	title := "GOLLM NEXUS"
	subtitle := "◢ HIGH-PERFORMANCE AI INTERFACE ◣"

	// Animated glitch effect
	if m.frame%20 < 3 {
		// Apply glitch
		runes := []rune(title)
		glitchCount := len(runes) / 4
		if glitchCount <= 0 {
			glitchCount = 1
		}
		for i := 0; i < glitchCount; i++ {
			if rand.Float64() < 0.3 && len(runes) > 0 && len(m.glitchChars) > 0 {
				runes[rand.Intn(len(runes))] = m.glitchChars[rand.Intn(len(m.glitchChars))]
			}
		}
		title = string(runes)
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.width)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary).
		Align(lipgloss.Center).
		Width(m.width).
		Faint(true)

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(1).
		Width(m.width - 2)

	content := titleStyle.Render(title) + "\n" + subtitleStyle.Render(subtitle)
	return borderStyle.Render(content)
}

// renderChatArea creates the main chat viewport with matrix effects
func (m Model) renderChatArea() string {
	// Render matrix rain background (simplified)
	matrixBg := m.renderMatrixBackground()

	// Chat viewport
	viewportStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Width(m.width - 2).
		Height(m.height - 12)

	viewport := m.viewport.View()

	// Overlay matrix background if no messages
	if len(m.messages) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(m.theme.TextMuted).
			Align(lipgloss.Center).
			Width(m.viewport.Width).
			Height(m.viewport.Height)

		emptyMsg := "▼ NEURAL LINK ESTABLISHED ▼\n\n" +
			"Enter your query to begin interaction...\n\n" +
			matrixBg

		viewport = emptyStyle.Render(emptyMsg)
	}

	return viewportStyle.Render(viewport)
}

// renderMatrixBackground creates a simplified matrix rain effect
func (m Model) renderMatrixBackground() string {
	if m.width < 20 {
		return ""
	}

	var lines []string
	for i := 0; i < 5; i++ {
		var line strings.Builder
		maxJ := m.width / 4
		if maxJ <= 0 {
			maxJ = 10
		}
		for j := 0; j < maxJ; j++ {
			if rand.Float64() < 0.1 {
				var char rune
				if len(m.glitchChars) > 0 {
					char = m.glitchChars[rand.Intn(len(m.glitchChars))]
				} else {
					char = '▓'
				}
				intensity := rand.Float64()
				if intensity > 0.7 {
					line.WriteString(lipgloss.NewStyle().
						Foreground(m.theme.Primary).
						Render(string(char)))
				} else if intensity > 0.4 {
					line.WriteString(lipgloss.NewStyle().
						Foreground(m.theme.Primary).
						Faint(true).
						Render(string(char)))
				} else {
					line.WriteString(lipgloss.NewStyle().
						Foreground(m.theme.TextMuted).
						Faint(true).
						Render(string(char)))
				}
			} else {
				line.WriteString(" ")
			}
		}
		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

// renderInputArea creates the input field with cyberpunk styling
func (m Model) renderInputArea() string {
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Padding(0, 1).
		Width(m.width - 2)

	input := m.textInput.View()

	if m.isLoading {
		spinner := m.spinner.View()
		input = fmt.Sprintf("%s %s Processing neural patterns...", spinner, input)
	}

	return inputStyle.Render(input)
}

// renderStatusBar creates the bottom status bar
func (m Model) renderStatusBar() string {
	left := fmt.Sprintf("Messages: %d | Session: %v",
		len(m.messages),
		time.Since(m.sessionStart).Round(time.Second))

	right := "F1:Stats F2:Debug F3:Multiline ESC:Exit"

	if m.showStats {
		right = fmt.Sprintf("Tokens: %d | %s", m.tokenCount, right)
	}

	leftStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	rightStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	padding := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return leftStyle.Render(left) + strings.Repeat(" ", padding) + rightStyle.Render(right)
}

// RunCyberpunkTUI starts the cyberpunk TUI interface
func RunCyberpunkTUI(cfg *config.Config, tuiConfig *Config) error {
	model := NewModel(cfg)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()
	return err
}
