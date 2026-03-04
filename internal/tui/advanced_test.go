package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAdvancedModel_ExecuteMenuItem_Help(t *testing.T) {
	m := NewAdvancedModel()
	mp := &m

	// Execute Help action directly
	mp.executeMenuItem(MenuItem{Action: "help"})

	if assert.Greater(t, len(mp.messages), 0, "expected at least one system message after help") {
		last := mp.messages[len(mp.messages)-1]
		assert.Equal(t, "system", last.Role)
		assert.Contains(t, last.Content, "HELP & COMMANDS")
	}
}

func TestAdvancedModel_HandleCommand_Clear(t *testing.T) {
	m := NewAdvancedModel()
	mp := &m

	// Seed with messages
	mp.addUserMessage("hi")
	mp.addAIMessage("hello")
	assert.GreaterOrEqual(t, len(mp.messages), 2)

	// Clear command
	mp.handleCommand("clear")
	assert.Equal(t, 0, len(mp.messages))
}

func TestAdvancedModel_ExecuteCommand_PreparesWhenAutoExecDisabled(t *testing.T) {
	m := NewAdvancedModel()
	mp := &m
	mp.autoExecute = false

	mp.executeCommand("echo test")

	if assert.Greater(t, len(mp.messages), 0) {
		last := mp.messages[len(mp.messages)-1]
		assert.Equal(t, "system", last.Role)
		assert.Contains(t, last.Content, "Command ready")
	}
}

func TestAdvancedModel_SetTheme(t *testing.T) {
	m := NewAdvancedModel()
	mp := &m

	mp.SetTheme("dark")
	assert.Equal(t, DarkTheme.Primary, mp.theme.Primary)

	mp.SetTheme("light")
	assert.Equal(t, LightTheme.Primary, mp.theme.Primary)

	mp.SetTheme("matrix")
	assert.Equal(t, MatrixTheme.Primary, mp.theme.Primary)
}

func TestAdvancedModel_UpdateViewport_RendersMessages(t *testing.T) {
	m := NewAdvancedModel()
	mp := &m

	mp.addUserMessage("Hello")
	mp.addAIMessage("Hi there")

	// ensure timestamps differ a bit for stability
	time.Sleep(10 * time.Millisecond)

	mp.updateViewport()
	view := mp.viewport.View()

	assert.Contains(t, view, "Hello")
	assert.Contains(t, view, "Hi there")
}
