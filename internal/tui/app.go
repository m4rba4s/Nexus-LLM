package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// DashboardUI structures the physical terminal interface components.
type DashboardUI struct {
	app         *tview.Application
	nodeTable   *tview.Table
	logView     *tview.TextView
	detailsView *tview.TextView
	grid        *tview.Grid
}

// NewDashboardUI initializes the visual tree.
func NewDashboardUI(app *tview.Application) *DashboardUI {
	ui := &DashboardUI{
		app: app,
	}

	// 1. Swarm Nodes Topology (Left Pane)
	ui.nodeTable = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	ui.nodeTable.SetTitle("[ Swarm Topology ]").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderColor(tcell.ColorGreen)

	ui.setupTableHeader()

	// 2. Event Stream / Gossip Log (Top Right Pane)
	ui.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetScrollable(true)

	// Stick to bottom
	ui.logView.SetChangedFunc(func() {
		ui.app.Draw()
	})

	ui.logView.SetTitle("[ Threat Intelligence Event Stream ]").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderColor(tcell.ColorBlue)

	// 3. Detail/Payload Inspection (Bottom Right Pane)
	ui.detailsView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)

	ui.detailsView.SetTitle("[ Defense Payload Inspector ]").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderColor(tcell.ColorOrange)

	// Layout Grid Configuration
	// Divide screen: Left 30% (Nodes), Right 70% (Logs + Details)
	ui.grid = tview.NewGrid().
		SetRows(0, 0).
		SetColumns(40, 0).
		SetBorders(false).
		AddItem(ui.nodeTable, 0, 0, 2, 1, 0, 0, true).
		AddItem(ui.logView, 0, 1, 1, 1, 0, 0, false).
		AddItem(ui.detailsView, 1, 1, 1, 1, 0, 0, false)

	return ui
}

func (ui *DashboardUI) setupTableHeader() {
	ui.nodeTable.Clear()
	headers := []string{"Node ID (Ed25519)", "IP Adress", "Status"}
	for col, header := range headers {
		ui.nodeTable.SetCell(0, col,
			tview.NewTableCell(header).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter).
				SetSelectable(false))
	}
}

// RefreshNodes draws the active node list
func (ui *DashboardUI) RefreshNodes(nodes map[string]*NodeInfo) {
	ui.setupTableHeader()

	// Sort by ID to keep the list stable
	var peerIDs []string
	for id := range nodes {
		peerIDs = append(peerIDs, id)
	}
	sort.Strings(peerIDs)

	for row, id := range peerIDs {
		n := nodes[id]

		// Format short ID (first 8 chars)
		shortID := id
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		// Calculate health/status
		status := "🟢 ACTV"
		if time.Since(n.LastSeen) > 10*time.Second {
			status = "🟡 STAL" // Stale
		}
		if time.Since(n.LastSeen) > 30*time.Second {
			status = "🔴 DEAD"
		}

		ui.nodeTable.SetCell(row+1, 0, tview.NewTableCell(shortID).SetTextColor(tcell.ColorLightCyan))
		ui.nodeTable.SetCell(row+1, 1, tview.NewTableCell(n.IP).SetTextColor(tcell.ColorWhite))
		ui.nodeTable.SetCell(row+1, 2, tview.NewTableCell(status).SetAlign(tview.AlignCenter))
	}
}

// RefreshLogs writes the gossip events
func (ui *DashboardUI) RefreshLogs(events []string) {
	ui.logView.SetText(strings.Join(events, "\n"))
	ui.logView.ScrollToEnd()
}

// RefreshDetails displays the raw JSON or YARA rule
func (ui *DashboardUI) RefreshDetails(details string) {
	ui.detailsView.SetText(details)
}

// Run executes the application blocking loop
func (ui *DashboardUI) Run() error {
	return ui.app.SetRoot(ui.grid, true).EnableMouse(true).Run()
}
