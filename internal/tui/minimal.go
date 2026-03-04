// Package tui provides a minimal terminal user interface for GOLLM
// for debugging and basic functionality testing.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/gollm/internal/config"
)

// MinimalModel represents the minimal TUI state
type MinimalModel struct {
	textInput textinput.Model
	viewport  viewport.Model
	messages  []string
	ready     bool
	width     int
	height    int
}

// responseMsg represents a response from the AI
type responseMsg struct {
	content string
	err     error
}

// NewMinimalModel creates a new minimal TUI model
func NewMinimalModel() *MinimalModel {
	// Create text input
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 50

	// Create viewport
	vp := viewport.New(78, 17)

	return &MinimalModel{
		textInput: ti,
		viewport:  vp,
		messages:  []string{},
		ready:     false,
	}
}

// Init initializes the minimal TUI
func (m MinimalModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m MinimalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handle window resize
		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-7)
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 7
		}
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.textInput.Value() != "" {
				return m.handleUserInput()
			}
		}

	case responseMsg:
		return m.handleResponse(msg)
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

// handleUserInput processes user input
func (m MinimalModel) handleUserInput() (MinimalModel, tea.Cmd) {
	input := m.textInput.Value()
	m.textInput.SetValue("")

	// Add user message
	userMsg := fmt.Sprintf("[%s] You: %s", time.Now().Format("15:04"), input)
	m.messages = append(m.messages, userMsg)
	m.updateViewport()

	// Send to AI (mock response for now)
	return m, m.sendToAI(input)
}

// sendToAI sends message to AI and returns response
func (m MinimalModel) sendToAI(input string) tea.Cmd {
	return func() tea.Msg {
		// Simulate AI processing
		time.Sleep(500 * time.Millisecond)

		// Mock responses based on input
		var response string
		switch strings.ToLower(input) {
		case "hello", "hi":
			response = "Hello! I'm your AI assistant. How can I help you today?"
		case "test":
			response = "Test successful! The minimal TUI is working correctly."
		case "quit", "exit":
			response = "Goodbye! Thanks for using GOLLM."
		default:
			response = fmt.Sprintf("I received: '%s'. This is a minimal TUI test response.", input)
		}

		return responseMsg{
			content: response,
			err:     nil,
		}
	}
}

// handleResponse processes AI response
func (m MinimalModel) handleResponse(msg responseMsg) (MinimalModel, tea.Cmd) {
	if msg.err != nil {
		errorMsg := fmt.Sprintf("[%s] Error: %s", time.Now().Format("15:04"), msg.err.Error())
		m.messages = append(m.messages, errorMsg)
	} else {
		aiMsg := fmt.Sprintf("[%s] AI: %s", time.Now().Format("15:04"), msg.content)
		m.messages = append(m.messages, aiMsg)
	}

	m.updateViewport()
	return m, nil
}

// updateViewport refreshes the viewport with current messages
func (m *MinimalModel) updateViewport() {
	content := strings.Join(m.messages, "\n\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// View renders the minimal TUI
func (m MinimalModel) View() string {
	if !m.ready {
		return "Initializing minimal TUI..."
	}

	// Create header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff41")).
		Render("🤖 GOLLM Minimal Chat Interface")

	// Create viewport section
	viewportStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Padding(1)

	chatArea := viewportStyle.Render(m.viewport.View())

	// Create input section
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#0066ff")).
		Padding(0, 1)

	inputArea := inputStyle.Render(m.textInput.View())

	// Create footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("Press ESC or Ctrl+C to exit • Enter to send message")

	// Join all sections
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		header,
		chatArea,
		inputArea,
		footer)
}

// RunMinimalTUI starts the minimal TUI interface
func RunMinimalTUI(cfg *config.Config, tuiConfig *Config) error {
	model := NewMinimalModel()

	// Add welcome message
	model.messages = append(model.messages,
		fmt.Sprintf("[%s] System: Welcome to GOLLM Minimal TUI!", time.Now().Format("15:04")),
		"",
		"This is a debug version without animations.",
		"Type 'test' to verify functionality.",
		"Type 'hello' for a greeting.",
		"Press ESC to exit.",
		"",
	)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
