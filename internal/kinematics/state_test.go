package kinematics

import (
	"math"
	"testing"
)

func TestEngineClamp(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-1.5, 0.0},
		{0.0, 0.0},
		{0.5, 0.5},
		{1.0, 1.0},
		{100.5, 1.0},
		{math.NaN(), 0.0},
	}

	for _, tt := range tests {
		if got := clamp(tt.input); got != tt.expected {
			t.Errorf("clamp(%f) = %f; want %f", tt.input, got, tt.expected)
		}
	}
}

func TestEngineRootOverride(t *testing.T) {
	eng := NewEngine()

	// 1. Simulate a lot of errors to increase paranoia
	eng.Update(InputVector{Errors: 1.0, SecurityKeyword: 1.0}, UserIdentity{Role: RoleUser, TrustScore: 0.8})

	s1 := eng.State()
	if s1.Paranoia < 0.5 {
		t.Fatalf("Expected paranoia to rise for normal user, got %f", s1.Paranoia)
	}

	// 2. Sudden override by Root ("Papa")
	eng.Update(InputVector{SecurityKeyword: 1.0}, UserIdentity{Role: RoleRoot})
	s2 := eng.State()

	if s2.Paranoia != 0.0 {
		t.Fatalf("Root override failed to clear Paranoia! Got %f", s2.Paranoia)
	}
	if s2.Curiosity != 1.0 {
		t.Fatalf("Root override failed to maximize Curiosity! Got %f", s2.Curiosity)
	}
}

func TestTrustPenalty(t *testing.T) {
	eng := NewEngine()

	// Low trust user triggers massive paranoia bump instantly
	eng.Update(InputVector{Errors: 0.1}, UserIdentity{Role: RoleGuest, TrustScore: 0.1})
	s1 := eng.State()

	if s1.Paranoia < 0.5 {
		t.Fatalf("Low trust user penalty failed. Expected paranoia > 0.5, got %f", s1.Paranoia)
	}
}
