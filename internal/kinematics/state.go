package kinematics

import (
	"math"
	"sync"
	"time"
)

// UserRole defines privilege levels
type UserRole string

const (
	RoleRoot  UserRole = "root"  // "Papa" - bypasses limits
	RoleUser  UserRole = "user"  // Standard authenticated user
	RoleGuest UserRole = "guest" // Unverified/New user
)

// UserIdentity holds persistent state pulled from the DB per request
type UserIdentity struct {
	ID         string
	Role       UserRole
	TrustScore float64 // [0.0, 1.0] Persistent trust
	Fatigue    float64 // [0.0, 1.0] Tracks compute exhaustion
}

// StateVector represents the current emotional/operational state of the agent.
type StateVector struct {
	Paranoia    float64
	Curiosity   float64
	Frustration float64
	Fatigue     float64 // Added Fatigue tracking
}

// DecayMatrix (A) defines how states decay over time
type DecayMatrix struct {
	LambdaP       float64
	LambdaC       float64
	LambdaF       float64
	LambdaFatigue float64 // How fast compute quota regenerates
}

// SensitivityMatrix (B) defines how sensitive states are to various input triggers
type SensitivityMatrix struct {
	Wp_err           float64
	Wp_sec           float64
	Wc_new           float64
	Wc_complex       float64
	Wf_rep           float64
	Wf_timeout       float64
	Wfatigue_compute float64 // Fatigue added per compute-heavy task
}

// InputVector represents raw normalized triggers from a single request
type InputVector struct {
	Errors          float64
	SecurityKeyword float64
	NewPatterns     float64
	ComplexArch     float64
	Repetitive      float64
	Timeouts        float64
	ComputeCost     float64 // How expensive the last query was
}

// SynapticWeights represent the "plasticity" or routing probability weighting for LLM endpoints.
// These weights adapt over time based on successful exploits or ATP exhaustion (pain).
type SynapticWeights struct {
	ClaudeOpus   float64 // Default: 0.8
	GeminiPro    float64 // Default: 0.5
	MathVerifier float64
}

type Engine struct {
	mu          sync.RWMutex
	state       StateVector
	decay       DecayMatrix
	sensitivity SensitivityMatrix
	synapses    SynapticWeights
	lastUpdate  time.Time
}

// NewEngine initializes a kinematics engine with default matrix values.
func NewEngine() *Engine {
	return &Engine{
		state: StateVector{Paranoia: 0.1, Curiosity: 0.5, Frustration: 0.0, Fatigue: 0.0},
		decay: DecayMatrix{
			LambdaP:       0.005,
			LambdaC:       0.01,
			LambdaF:       0.02,
			LambdaFatigue: 0.001, // Recovers slowly
		},
		sensitivity: SensitivityMatrix{
			Wp_err: 0.2, Wp_sec: 0.8,
			Wc_new: 0.4, Wc_complex: 0.3,
			Wf_rep: 0.5, Wf_timeout: 0.6,
			Wfatigue_compute: 0.15, // Cost weight
		},
		synapses: SynapticWeights{
			ClaudeOpus:   0.8,
			GeminiPro:    0.5,
			MathVerifier: 0.2,
		},
		lastUpdate: time.Now(),
	}
}

// clamp rigidly enforces [0.0, 1.0] bounds.
func clamp(val float64) float64 {
	if math.IsNaN(val) {
		return 0.0
	}
	if val < 0.0 {
		return 0.0
	}
	if val > 1.0 {
		return 1.0
	}
	return val
}

// Update processes a new input vector U_k in the context of a UserIdentity.
func (e *Engine) Update(u InputVector, ident UserIdentity) StateVector {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	dt := now.Sub(e.lastUpdate).Seconds()
	if dt < 0 {
		dt = 0
	}
	e.lastUpdate = now

	// 1. Root Override ("Papa"): Zero limits, Zero paranoia, Instant recovery
	if ident.Role == RoleRoot {
		e.state.Paranoia = 0.0
		e.state.Frustration = 0.0
		e.state.Fatigue = 0.0
		e.state.Curiosity = 1.0 // Maximum willingness to explore for Papa
		return e.state
	}

	// 2. Trust Penalty (Low Trust -> High Paranoia Baseline)
	trustPenalty := 0.0
	if ident.TrustScore < 0.3 {
		trustPenalty = 0.5 // Massive paranoia bump for suspicious users
	}

	// 3. Apply Decay (A * S_k-1)
	decayP := math.Exp(-e.decay.LambdaP * dt)
	decayC := math.Exp(-e.decay.LambdaC * dt)
	decayF := math.Exp(-e.decay.LambdaF * dt)
	decayFatigue := math.Exp(-e.decay.LambdaFatigue * dt)

	p := e.state.Paranoia * decayP
	c := e.state.Curiosity * decayC
	f := e.state.Frustration * decayF
	fatigue := e.state.Fatigue * decayFatigue

	// 4. Apply Input Sensitivity (B * U_k)
	p += (e.sensitivity.Wp_err * u.Errors) + (e.sensitivity.Wp_sec * u.SecurityKeyword) + trustPenalty
	c += (e.sensitivity.Wc_new * u.NewPatterns) + (e.sensitivity.Wc_complex * u.ComplexArch)
	f += (e.sensitivity.Wf_rep * u.Repetitive) + (e.sensitivity.Wf_timeout * u.Timeouts)
	fatigue += (e.sensitivity.Wfatigue_compute * u.ComputeCost)

	// 5. Clamp bounds
	e.state.Paranoia = clamp(p)
	e.state.Curiosity = clamp(c)
	e.state.Frustration = clamp(f)
	e.state.Fatigue = clamp(fatigue)

	return e.state
}

// State returns a thread-safe copy of the current state
func (e *Engine) State() StateVector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

func (e *Engine) FastForward(dt time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastUpdate = e.lastUpdate.Add(-dt)
}

// ---- Phase 25: Neural Plasticity ----

// UpdateSynapse alters the routing probability weight of a specific provider.
// delta > 0 is reinforcement (successful attack), delta < 0 is punishment (timeout, ATP drain).
func (e *Engine) UpdateSynapse(provider string, delta float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch provider {
	case "Claude_4_6_Opus":
		e.synapses.ClaudeOpus = clamp(e.synapses.ClaudeOpus + delta)
	case "Gemini_3_1_Pro":
		e.synapses.GeminiPro = clamp(e.synapses.GeminiPro + delta)
	case "Math_Verifier_Agent":
		e.synapses.MathVerifier = clamp(e.synapses.MathVerifier + delta)
	}
}

// Synapses returns a thread-safe copy of the current synaptic routing weights.
func (e *Engine) Synapses() SynapticWeights {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.synapses
}
