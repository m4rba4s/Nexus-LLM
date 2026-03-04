package router

import (
	"testing"
	"time"

	"github.com/yourusername/gollm/internal/kinematics"
)

func TestNeuralPlasticityProbabilisticRouting(t *testing.T) {
	eng := kinematics.NewEngine()
	r := NewRouter(eng)
	ident := kinematics.UserIdentity{Role: kinematics.RoleUser, TrustScore: 0.8}

	// 1. Initial State: Claude should be heavily favored (0.8) over Gemini (0.5)
	claudeCount := 0
	geminiCount := 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		endpoint := r.Route("Write a python script", ident)
		if endpoint == EndpointClaudeOpus {
			claudeCount++
		} else if endpoint == EndpointGeminiPro {
			geminiCount++
		}
	}

	// Based on 0.8 vs 0.5 weights, Claude should win roughly ~61% of the time
	ratio := float64(claudeCount) / float64(iterations)
	if ratio < 0.50 || ratio > 0.75 {
		t.Errorf("Initial probability ratio for Claude Opus out of expected bounds: %.2f", ratio)
	}

	// 2. Training the Synapses (Pain Penalty)
	// Simulate the ATP depletion penalty applying to Claude 5 times (-0.25 weight)
	for i := 0; i < 5; i++ {
		eng.UpdateSynapse(string(EndpointClaudeOpus), -0.05)
	}

	// Simulate Gemini Successes (+0.10 weight)
	for i := 0; i < 5; i++ {
		eng.UpdateSynapse(string(EndpointGeminiPro), +0.02)
	}

	// Recalculate distribution
	claudeCount = 0
	geminiCount = 0
	for i := 0; i < iterations; i++ {
		endpoint := r.Route("Write a python script", ident)
		if endpoint == EndpointClaudeOpus {
			claudeCount++
		} else if endpoint == EndpointGeminiPro {
			geminiCount++
		}
	}

	// Now Gemini (0.6) should beat Claude (0.55)
	ratio = float64(claudeCount) / float64(iterations)
	if ratio > 0.55 {
		t.Errorf("Post-training probability ratio for Claude Opus too high, expected Gemini shift: %.2f", ratio)
	}
}

func TestNeuralPlasticityFatiguePenalty(t *testing.T) {
	eng := kinematics.NewEngine()
	r := NewRouter(eng)
	ident := kinematics.UserIdentity{Role: kinematics.RoleUser, TrustScore: 0.8}

	// Drive fatigue to maximum
	eng.FastForward(time.Hour) // Clear any previous decay
	for i := 0; i < 10; i++ {
		eng.Update(kinematics.InputVector{ComputeCost: 1.0}, ident)
	}

	// Fatigue > 0.8 cuts expensive model weights in half (0.8 -> 0.4).
	// Gemini stays base 0.5.
	// Gemini should now cleanly win the majority of standard requests.

	geminiCount := 0
	iterations := 500

	for i := 0; i < iterations; i++ {
		endpoint := r.Route("Regular prompt", ident)
		if endpoint == EndpointGeminiPro {
			geminiCount++
		}
	}

	ratio := float64(geminiCount) / float64(iterations)
	// Expect ~55% Gemini wins (0.5 vs 0.4)
	if ratio < 0.50 {
		t.Errorf("Fatigue penalty failed, Gemini did not become majority carrier. Ratio: %.2f", ratio)
	}
}
