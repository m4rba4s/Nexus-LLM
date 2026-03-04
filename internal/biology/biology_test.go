package biology

import (
	"bytes"
	"testing"
	"time"

	"github.com/yourusername/gollm/internal/autonomy"
	"github.com/yourusername/gollm/internal/kinematics"
)

func TestGeneticEncoder_EncodeDecode(t *testing.T) {
	enc := NewGeneticEncoder()

	plan := &autonomy.DefensePlan{
		SyscallTarget: "test_call",
		DefenseType:   "yara",
		BPFCode:       "rule Test { condition: true }",
	}

	raw, err := enc.Encode(plan)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify Codons
	if !bytes.HasPrefix(raw, []byte{0x41, 0x54, 0x47}) {
		t.Errorf("Missing ATG Start codon")
	}
	if !bytes.HasSuffix(raw, []byte{0x54, 0x41, 0x47}) {
		t.Errorf("Missing TAG Stop codon")
	}

	decoded, err := enc.Decode(raw)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.SyscallTarget != plan.SyscallTarget {
		t.Errorf("Mismatch in decoded payload. Got %s", decoded.SyscallTarget)
	}
}

func TestMitochondria_ATPConsumption(t *testing.T) {
	// 50 max, regen 10 ATP per second
	mito := NewMitochondria(50, 10, 1*time.Second)
	defer mito.Shutdown()

	if err := mito.Consume(40); err != nil {
		t.Fatalf("Failed to consume initial ATP: %v", err)
	}

	if err := mito.Consume(20); err == nil {
		t.Errorf("Expected ATP depletion error, but it succeeded")
	}

	// Wait for 2 regeneration cycles (20 ATP)
	time.Sleep(2100 * time.Millisecond)

	if err := mito.Consume(20); err != nil {
		t.Errorf("Failed to consume regenerated ATP after rest: %v", err)
	}
}

func TestNervousSystem_ApoptosisTrigger(t *testing.T) {
	eng := kinematics.NewEngine()
	user := kinematics.UserIdentity{ID: "test-user", Role: kinematics.RoleUser, Fatigue: 0.0}

	ns := NewNervousSystem(eng, user)

	// Inflict non-fatal pain
	ns.InducePain(40)

	select {
	case <-ns.WaitForDeath():
		t.Fatalf("Apoptosis triggered too early!")
	default:
		// Healthy
	}

	// Inflict fatal pain crossing the threshold (100)
	ns.InducePain(70)

	select {
	case <-ns.WaitForDeath():
		// Success: System died.
	case <-time.After(1 * time.Second):
		t.Fatalf("System survived fatal trauma. Apoptosis failed.")
	}
}
