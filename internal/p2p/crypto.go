package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Identity represents a cryptographic identity of a Swarm Node.
type Identity struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	NodeID     string // Hex representation of the public key
}

// SignedMessage represents a generic payload signed by a Node.
type SignedMessage struct {
	SenderID  string `json:"sender_id"`
	Payload   []byte `json:"payload"`
	Signature []byte `json:"signature"`
}

// GenerateIdentity creates a new Zero-Trust Ed25519 keypair for the node.
func GenerateIdentity() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 keys: %w", err)
	}

	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
		NodeID:     hex.EncodeToString(pub),
	}, nil
}

// LoadIdentityFromHex loads an identity if private key is known (e.g. from env)
func LoadIdentityFromHex(privHex string) (*Identity, error) {
	privBytes, err := hex.DecodeString(privHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex private key: %w", err)
	}

	if len(privBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: got %d, want %d", len(privBytes), ed25519.PrivateKeySize)
	}

	priv := ed25519.PrivateKey(privBytes)
	pub := priv.Public().(ed25519.PublicKey)

	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
		NodeID:     hex.EncodeToString(pub),
	}, nil
}

// Sign payload with node's private key.
func (id *Identity) Sign(payload []byte) *SignedMessage {
	sig := ed25519.Sign(id.PrivateKey, payload)
	return &SignedMessage{
		SenderID:  id.NodeID,
		Payload:   payload,
		Signature: sig,
	}
}

// Verify checks a generic SignedMessage using the sender's embedded public key hex (SenderID).
// In a true Zero-Trust system, the NodeID must ALSO be present in a local TrustStore (Whitelist),
// otherwise anyone can sign valid messages. That check happens at the application layer.
func VerifyMessage(msg *SignedMessage) bool {
	pubBytes, err := hex.DecodeString(msg.SenderID)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(pubBytes), msg.Payload, msg.Signature)
}

// PayloadToJSON is a helper to encode SignedMessage to transmittable bytes
func (msg *SignedMessage) ToJSON() ([]byte, error) {
	return json.Marshal(msg)
}

// ParseSignedMessage is a helper to decode bytes back to SignedMessage
func ParseSignedMessage(data []byte) (*SignedMessage, error) {
	var msg SignedMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
