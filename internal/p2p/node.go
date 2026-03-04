package p2p

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

// SwarmNode represents a single instance of NexusLLM in the P2P network.
type SwarmNode struct {
	Identity         *Identity
	Port             int
	Peers            map[string]*PeerInfo
	OnPeerDiscovered func(id string, ip string) // TUI Hook
}

// PeerInfo holds connection and identity data about discovered nodes.
type PeerInfo struct {
	NodeID string
	Host   string
	Port   int
}

// NewSwarmNode initializes the local node, generating a new cryptographic identity.
func NewSwarmNode(port int) (*SwarmNode, error) {
	id, err := GenerateIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize node identity: %w", err)
	}

	return &SwarmNode{
		Identity: id,
		Port:     port,
		Peers:    make(map[string]*PeerInfo),
	}, nil
}

// StartDiscovery launches mDNS broadcasting (so others can find us)
// and mDNS querying (so we can find others).
func (sn *SwarmNode) StartDiscovery(ctx context.Context) error {
	log.Printf("[SWARM] Node %s booting P2P Discovery on port %d...", sn.Identity.NodeID[:8], sn.Port)

	// 1. Broadcast ourselves to the local network
	host, _ := os.Hostname()
	info := []string{"nexus_node", fmt.Sprintf("id=%s", sn.Identity.NodeID)}

	service, err := mdns.NewMDNSService(host, "_nexus._udp", "", "", sn.Port, nil, info)
	if err != nil {
		return fmt.Errorf("mdns service creation failed: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return fmt.Errorf("mdns server failed: %w", err)
	}

	// 2. Listen for other nodes
	entriesCh := make(chan *mdns.ServiceEntry, 10)
	go func() {
		for entry := range entriesCh {
			sn.handleDiscoveredPeer(entry)
		}
	}()

	// Background loop to poll for peers constantly
	go func() {
		defer server.Shutdown()
		defer close(entriesCh)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Initial sweep
		mdns.Lookup("_nexus._udp", entriesCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mdns.Lookup("_nexus._udp", entriesCh)
			}
		}
	}()

	return nil
}

func (sn *SwarmNode) handleDiscoveredPeer(entry *mdns.ServiceEntry) {
	// Extract NodeID from TXT records: ["nexus_node", "id=abc123hex..."]
	var nodeID string
	for _, info := range entry.InfoFields {
		if strings.HasPrefix(info, "id=") {
			nodeID = strings.TrimPrefix(info, "id=")
			break
		}
	}

	if nodeID == "" || nodeID == sn.Identity.NodeID {
		return // Ignore self or malformed records
	}

	if _, exists := sn.Peers[nodeID]; !exists {
		log.Printf("[SWARM] Discovered new peer: %s at %s:%d", nodeID[:8], entry.AddrV4, entry.Port)
		sn.Peers[nodeID] = &PeerInfo{
			NodeID: nodeID,
			Host:   entry.AddrV4.String(),
			Port:   entry.Port,
		}

		// Fire the hook if registered
		if sn.OnPeerDiscovered != nil {
			sn.OnPeerDiscovered(nodeID, entry.AddrV4.String())
		}
	}
}
