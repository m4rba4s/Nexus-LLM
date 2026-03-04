package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/rivo/tview"
)

// AppState holds the thread-safe telemetry context for the TUI Dashboard.
// It acts as a passive observer collecting events from the P2P network.
type AppState struct {
	mu           sync.RWMutex
	Nodes        map[string]*NodeInfo
	GossipEvents []string
	DetailsText  string

	app *tview.Application
	ui  *DashboardUI
}

type NodeInfo struct {
	ID       string
	IP       string
	LastSeen time.Time
}

// NewAppState initializes the central state bucket used by the TUI.
func NewAppState(app *tview.Application) *AppState {
	return &AppState{
		Nodes:        make(map[string]*NodeInfo),
		GossipEvents: make([]string, 0),
		app:          app,
	}
}

// AddOrUpdateNode refreshes a node's heartbeat in the UI state.
func (s *AppState) AddOrUpdateNode(id string, ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Nodes[id] = &NodeInfo{
		ID:       id,
		IP:       ip,
		LastSeen: time.Now(),
	}

	// Trigger a redraw
	if s.app != nil && s.ui != nil {
		s.app.QueueUpdateDraw(func() {
			s.ui.RefreshNodes(s.Nodes)
		})
	}
}

// LogEvent appends a gossip or alert log to the UI.
func (s *AppState) LogEvent(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts := time.Now().Format("15:04:05")
	formattedMsg := fmt.Sprintf("[%s] %s", ts, msg)
	s.GossipEvents = append(s.GossipEvents, formattedMsg)

	// Keep only the last 100 events to prevent memory blowup
	if len(s.GossipEvents) > 100 {
		s.GossipEvents = s.GossipEvents[len(s.GossipEvents)-100:]
	}

	if s.app != nil && s.ui != nil {
		s.app.QueueUpdateDraw(func() {
			s.ui.RefreshLogs(s.GossipEvents)
		})
	}
}

// SetDetails displays detailed JSON or hex dumps in the details pane.
func (s *AppState) SetDetails(detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DetailsText = detail
	if s.app != nil && s.ui != nil {
		s.app.QueueUpdateDraw(func() {
			s.ui.RefreshDetails(s.DetailsText)
		})
	}
}
