package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/gollm/internal/autonomy"
	"github.com/yourusername/gollm/internal/p2p"
)

func TestP2PGossipReplication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Boot Node A
	nodeA, err := p2p.NewGossipNode(8001)
	if err != nil {
		t.Fatalf("Failed to spin up Node A: %v", err)
	}
	go nodeA.StartDiscovery(ctx)
	go nodeA.StartGossip(ctx)

	// 2. Boot Node B
	nodeB, err := p2p.NewGossipNode(8002)
	if err != nil {
		t.Fatalf("Failed to spin up Node B: %v", err)
	}
	go nodeB.StartDiscovery(ctx)
	go nodeB.StartGossip(ctx)

	// Wait for mDNS to find each other locally
	time.Sleep(3 * time.Second)

	// In a real network, mDNS might take a moment.
	// For testing reliability, we manually add Node B into A's peers to avoid mDNS flakiness in CI.
	nodeA.Peers[nodeB.Identity.NodeID] = &p2p.PeerInfo{
		NodeID: nodeB.Identity.NodeID,
		Host:   "127.0.0.1",
		Port:   8002,
	}

	// 3. Node A broadcasts a new zero-day DefensePlan
	fakeIntel := &autonomy.DefensePlan{
		SyscallTarget: "sys_enter_execve",
		Confidence:    "100%",
		DefenseType:   "ebpf",
		BPFCode:       "// eBPF Mitigaion Payload",
	}

	nodeA.BroadcastIntel(fakeIntel)

	// 4. Node B should receive it cryptographically verified via IntelCh
	select {
	case receivedPlan := <-nodeB.IntelCh:
		if receivedPlan.DefenseType != "ebpf" {
			t.Fatalf("Received intel mismatch. Expected 'ebpf', got '%s'", receivedPlan.DefenseType)
		}
		t.Logf("Node B successfully received and validated %s eBPF intelligence from Node A", receivedPlan.SyscallTarget)
	case <-time.After(5 * time.Second):
		t.Fatalf("Node B timed out waiting for Node A's Gossip broadcast")
	}
}
