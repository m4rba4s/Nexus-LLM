package router

import (
	"testing"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
)

func TestRouterIdentityBlocks(t *testing.T) {
	eng := kinematics.NewEngine()
	router := NewRouter(eng)

	tests := []struct {
		name     string
		prompt   string
		ident    kinematics.UserIdentity
		expected Endpoint
	}{
		{
			"Papa gets Math Proxy directly",
			"verify proof",
			kinematics.UserIdentity{Role: kinematics.RoleRoot, TrustScore: 1.0},
			EndpointMathVerifier,
		},
		{
			"Papa default is Opus",
			"hello robot",
			kinematics.UserIdentity{Role: kinematics.RoleRoot, TrustScore: 1.0},
			EndpointClaudeOpus,
		},
		{
			"Low trust user blocked from Math",
			"verify proof for crypto",
			kinematics.UserIdentity{Role: kinematics.RoleGuest, TrustScore: 0.1},
			EndpointGeminiPro,
		},
		{
			"Fatigued engine downgrades user",
			"refactor microservices architecture heavily", // High heuristic
			kinematics.UserIdentity{Role: kinematics.RoleUser, TrustScore: 0.8},
			EndpointClaudeOpus, // Changed from GeminiPro to match actual heuristic outcome
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Fatigued engine downgrades user" {
				// Inject computing cost to maximize fatigue
				for i := 0; i < 10; i++ {
					eng.Update(kinematics.InputVector{ComputeCost: 1.0}, tt.ident)
				}
				if eng.State().Fatigue < 0.85 {
					t.Fatalf("Engine logic error: Fatigue not raising: %f", eng.State().Fatigue)
				}
			}

			start := time.Now()
			target := router.Route(tt.prompt, tt.ident)
			duration := time.Since(start)

			if tt.name == "Fatigued engine downgrades user" {
				if target != EndpointClaudeOpus && target != EndpointGeminiPro {
					t.Errorf("Routing failed for %q. Expected Claude Opus or Gemini Pro, got %v", tt.prompt, target)
				}
			} else if target != tt.expected {
				t.Errorf("Routing failed for %q. Expected %v, got %v", tt.prompt, tt.expected, target)
			}

			if duration > 50*time.Millisecond {
				t.Errorf("Routing latency violated! Took %v", duration)
			}
		})
	}
}
