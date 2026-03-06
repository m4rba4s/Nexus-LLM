package tui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rivo/tview"
	"github.com/m4rba4s/Nexus-LLM/internal/autonomy"
	"github.com/m4rba4s/Nexus-LLM/internal/p2p"
)

// RunDashboard boots a passive Swarm node and attaches the TUI to its event streams.
func RunDashboard(port int) error {
	app := tview.NewApplication()
	state := NewAppState(app)
	ui := NewDashboardUI(app)

	// 1. Initialize passive gossip node
	gn, err := p2p.NewGossipNode(port)
	if err != nil {
		return fmt.Errorf("failed to start TUI observer node: %w", err)
	}

	// 2. Register TUI Bridge Hooks (Zero eBPF Injection overhead here)
	gn.OnPeerDiscovered = func(id string, ip string) {
		message := fmt.Sprintf("System Log: Discovered new peer %s at %s", id[:8], ip)
		state.LogEvent(message)
		state.AddOrUpdateNode(id, ip)
	}

	gn.OnIntelReceived = func(plan *autonomy.DefensePlan, senderID string) {
		message := fmt.Sprintf("⚠️ Threat Intel Downloaded [%s] (Confidence: %s) from %s", plan.DefenseType, plan.Confidence, senderID[:8])
		state.LogEvent(message)

		// Format DefensePlan payload into pretty JSON for the details pane
		details, _ := json.MarshalIndent(plan, "", "  ")
		state.SetDetails(string(details))

		// Note: The UI layer never passes this to `SandboxHotReloader`
		// Ensuring strict isolation of the Observer layout.
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Start Swarm Background Daemons
	if err := gn.StartDiscovery(ctx); err != nil {
		return fmt.Errorf("failed starting mDNS: %w", err)
	}
	gn.StartGossip(ctx)

	state.LogEvent(fmt.Sprintf("NexusLLM Observer initialized on UDP/TCP port %d. Waiting for Swarm gossip...", port))

	// 4. Run UI Loop (this blocks until Ctrl+C)
	return ui.Run()
}
