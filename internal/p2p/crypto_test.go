package p2p_test

import (
	"testing"

	"github.com/m4rba4s/Nexus-LLM/internal/p2p"
)

func TestEd25519Cryptography(t *testing.T) {
	// 1. Generate new identity
	identity, err := p2p.GenerateIdentity()
	if err != nil {
		t.Fatalf("Failed to generate identity: %v", err)
	}

	if len(identity.NodeID) != 64 { // 32 bytes hex encoded = 64 chars
		t.Fatalf("Expected NodeID to be 64 characters, got %d", len(identity.NodeID))
	}

	payload := []byte("secret_eBPF_intel")

	// 2. Sign payload
	signedMsg := identity.Sign(payload)

	// 3. Verify signature mathematically
	if !p2p.VerifyMessage(signedMsg) {
		t.Fatalf("Signature verification failed for authentic message")
	}

	// 4. Test forgery resistance
	fakeIdentity, _ := p2p.GenerateIdentity()
	forgedMsg := &p2p.SignedMessage{
		SenderID:  fakeIdentity.NodeID,
		Payload:   payload,
		Signature: signedMsg.Signature, // Copied signature from original identity
	}

	if p2p.VerifyMessage(forgedMsg) {
		t.Fatalf("Signature verification ALLOWED a forged message (Critical Security Flaw)")
	}

	// 5. Test JSON serialization
	data, err := signedMsg.ToJSON()
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	parsedMsg, err := p2p.ParseSignedMessage(data)
	if err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}

	if !p2p.VerifyMessage(parsedMsg) {
		t.Fatalf("Signature verification failed after JSON roundtrip")
	}

	t.Logf("Ed25519 Cryptography is robust. Zero Trust verification passed.")
}
