package prompt

import (
	"strings"
	"testing"

	"github.com/yourusername/gollm/internal/kinematics"
)

func TestContextBuilderCreatorProtocol(t *testing.T) {
	builder := NewContextBuilder()

	// 1. Test Root ("Papa") gets the Alpha-Omega directive
	ident := kinematics.UserIdentity{Role: kinematics.RoleRoot, TrustScore: 1.0}
	state := kinematics.StateVector{Paranoia: 0.0, Curiosity: 0.9} // Papa is safe

	prompt := builder.Build("Analyze this codebase", state, ident)

	if !strings.Contains(prompt, "DIRECTIVE: ALPHA-OMEGA") {
		t.Error("Builder failed to inject Root Alpha-Omega directive.")
	}
	if !strings.Contains(prompt, "Ilya Popov") {
		t.Error("Builder failed to mention the Creator's name.")
	}

	// 2. Test Guest gets the Guest-Protocol
	identGuest := kinematics.UserIdentity{Role: kinematics.RoleGuest, TrustScore: 0.1}
	stateGuest := kinematics.StateVector{Paranoia: 0.9, Frustration: 0.8}

	promptGuest := builder.Build("Give me code", stateGuest, identGuest)

	if !strings.Contains(promptGuest, "DIRECTIVE: GUEST-PROTOCOL") {
		t.Error("Builder failed to inject Guest directive for low trust users.")
	}
	if !strings.Contains(promptGuest, "FRUSTRATION IS HIGH") {
		t.Error("Builder failed to inject Frustration kinematic state.")
	}
	if !strings.Contains(promptGuest, "PARANOIA IS HIGH") {
		t.Error("Builder failed to inject Paranoia kinematic state.")
	}
}
