package p2p

import (
	"context"
	"log"

	"github.com/yourusername/gollm/internal/autonomy"
)

// Coordinator connects the Gossip network to the Defensive Ring-0 compiler.
// It applies received Threat Intelligence (eBPF) instantly ensuring swarm immunity.
type Coordinator struct {
	node     *GossipNode
	reloader autonomy.SandboxHotReloader
}

// NewCoordinator creates the bridging layer for Swarm Intelligence.
func NewCoordinator(node *GossipNode, reloader autonomy.SandboxHotReloader) *Coordinator {
	return &Coordinator{
		node:     node,
		reloader: reloader,
	}
}

// Start listens endlessly for incoming DefensePlans and pushes them to the kernel.
func (c *Coordinator) Start(ctx context.Context) {
	log.Println("[SWARM-COORD] Activating P2P to eBPF integration loop...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[SWARM-COORD] Shutting down P2P defense relay.")
			return
		case plan := <-c.node.IntelCh:
			// Ensure it represents an eBPF mitigation payload
			if plan.DefenseType == "ebpf" || plan.Type == "eBPF" || plan.Type == "ebpf" {
				log.Printf("[SWARM-COORD] Received critical zero-day Mitigation (Target: %s, Confidence: %s). Instructing Kernel loader...", plan.SyscallTarget, plan.Confidence)

				// Zero-Trust was verified at the Gossip TCP layer (Ed25519 signature).
				// Therefore we trust this payload enough to attempt JIT clang compilation.
				if err := c.reloader.LoadAndAttach(ctx, plan.BPFCode); err != nil {
					log.Printf("[SWARM-ERROR] Kernel rejected dynamic Swarm defense patch: %v", err)
				} else {
					log.Println("[SWARM-COORD] SUCCESS! Hardware successfully patched kernel from Swarm Intelligence!")
				}
			}
		}
	}
}
