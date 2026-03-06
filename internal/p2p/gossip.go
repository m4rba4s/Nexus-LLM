package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/autonomy"
)

// GossipNode extends SwarmNode with C2 capabilities.
type GossipNode struct {
	*SwarmNode
	Listener        net.Listener
	IntelCh         chan *autonomy.DefensePlan
	OnIntelReceived func(plan *autonomy.DefensePlan, senderID string) // TUI Hook
}

// NewGossipNode initializes a functional C2 P2P overlay.
func NewGossipNode(port int) (*GossipNode, error) {
	node, err := NewSwarmNode(port)
	if err != nil {
		return nil, err
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed binding gossip port %d: %w", port, err)
	}

	return &GossipNode{
		SwarmNode: node,
		Listener:  l,
		IntelCh:   make(chan *autonomy.DefensePlan, 50),
	}, nil
}

// StartGossip spins up the TCP listener to receive intelligence from the Swarm.
func (gn *GossipNode) StartGossip(ctx context.Context) {
	go func() {
		<-ctx.Done()
		gn.Listener.Close()
	}()

	for {
		conn, err := gn.Listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return // Context canceled
			}
			log.Printf("[SWARM-WARN] Failed accepting peer connection: %v", err)
			continue
		}

		go gn.handleIncomingPeer(conn)
	}
}

func (gn *GossipNode) handleIncomingPeer(conn net.Conn) {
	defer conn.Close()

	data, err := io.ReadAll(conn)
	if err != nil {
		log.Printf("[SWARM-WARN] Read error from peer: %v", err)
		return
	}

	// 1. Unpack cryptographic wrapper
	signedMsg, err := ParseSignedMessage(data)
	if err != nil {
		log.Printf("[SWARM-WARN] Invalid message envelope from peer")
		return
	}

	// 2. Validate Ed25519 Cryptographic Signature (Zero Trust Layer)
	if !VerifyMessage(signedMsg) {
		log.Printf("[SWARM-CRITICAL] Dropped malformed/forged intel from %s. Signature invalid!", signedMsg.SenderID)
		return
	}

	// 3. Deserialize authentic payload into DefensePlan
	var plan autonomy.DefensePlan
	if err := json.Unmarshal(signedMsg.Payload, &plan); err != nil {
		log.Printf("[SWARM-WARN] Failed to decode valid signature payload to DefensePlan")
		return
	}

	log.Printf("[SWARM] Received validated Threat Intel (DefenseType: %s) from Peer %s", plan.DefenseType, signedMsg.SenderID[:8])

	// Fire the optional observer hook for TUI
	if gn.OnIntelReceived != nil {
		gn.OnIntelReceived(&plan, signedMsg.SenderID)
	}

	// Pass to core integration loop (eBPF loader)
	gn.IntelCh <- &plan
}

// BroadcastIntel serializes a DefensePlan, signs it, and pushes to all known Swarm peers.
func (gn *GossipNode) BroadcastIntel(plan *autonomy.DefensePlan) {
	payload, err := json.Marshal(plan)
	if err != nil {
		log.Printf("[SWARM-ERROR] Failed to marshal plan for broadcast: %v", err)
		return
	}

	// Cryptographically sign the intel
	signedMsg := gn.Identity.Sign(payload)

	frame, err := signedMsg.ToJSON()
	if err != nil {
		return
	}

	for id, peer := range gn.Peers {
		go func(pID string, p *PeerInfo) {
			addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.Port))
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				log.Printf("[SWARM-DEBUG] Peer %s unreachable: %v", pID[:8], err)
				return // Peer went offline
			}
			defer conn.Close()

			_, _ = conn.Write(frame)
		}(id, peer)
	}

	log.Printf("[SWARM] Broadcasted Threat Intel to %d peers.", len(gn.Peers))
}
