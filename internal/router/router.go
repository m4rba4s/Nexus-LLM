package router

import (
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
)

type Endpoint string

const (
	EndpointMathVerifier Endpoint = "Math_Verifier_Agent"
	EndpointClaudeOpus   Endpoint = "Claude_4_6_Opus"
	EndpointGeminiPro    Endpoint = "Gemini_3_1_Pro"
	EndpointWebNavigator Endpoint = "Web_Navigator_Agent"
)

type Router struct {
	mathRegex *regexp.Regexp
	engine    *kinematics.Engine
}

func NewRouter(eng *kinematics.Engine) *Router {
	rand.Seed(time.Now().UnixNano())
	return &Router{
		mathRegex: regexp.MustCompile(`(?i)([\$\\]\w+|proof|verify|sym|integral)`),
		engine:    eng,
	}
}

// Route decides the endpoint based on prompt text, emotional state, and Neural Plasticity (Synapses).
func (r *Router) Route(prompt string, ident kinematics.UserIdentity) Endpoint {
	state := r.engine.State()

	// 1. Root User ("Papa") Override
	if ident.Role == kinematics.RoleRoot {
		if r.mathRegex.MatchString(prompt) || strings.Contains(strings.ToLower(prompt), "crypto") {
			return EndpointMathVerifier
		}
		return EndpointClaudeOpus
	}

	// 2. Trust Blocks (Hard limits remain deterministic)
	if ident.TrustScore < 0.3 {
		return EndpointGeminiPro
	}

	// 3. Autonomous Web Execution Logic (Phase 15)
	if checkWebBrowsingHeuristic(prompt) {
		return EndpointWebNavigator
	}

	// 4. Neural Plasticity Routing (Phase 25)
	// Instead of heuristics, we roll probabilistically based on Synaptic Weights
	synapses := r.engine.Synapses()

	// Math operations heavily bias the Math Verifier synapse
	if r.mathRegex.MatchString(prompt) || strings.Contains(strings.ToLower(prompt), "crypto") {
		return r.probabilisticSelect(map[Endpoint]float64{
			EndpointMathVerifier: synapses.MathVerifier * 3.0, // Temporary contextual spike
			EndpointClaudeOpus:   synapses.ClaudeOpus * 0.2,
			EndpointGeminiPro:    synapses.GeminiPro * 0.1,
		})
	}

	// Fatigue depresses heavy-compute synapses
	fatiguePenalty := 1.0
	if state.Fatigue >= 0.8 {
		fatiguePenalty = 0.5 // Heavy hit to expensive models
	}

	// Standard Probabilistic Route
	return r.probabilisticSelect(map[Endpoint]float64{
		EndpointClaudeOpus: synapses.ClaudeOpus * fatiguePenalty,
		EndpointGeminiPro:  synapses.GeminiPro, // Baseline is unaffected by fatigue
	})
}

// probabilisticSelect choses an endpoint based on the relative weights simulating a synaptic firing.
func (r *Router) probabilisticSelect(weights map[Endpoint]float64) Endpoint {
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	roll := rand.Float64() * totalWeight
	current := 0.0

	for ep, w := range weights {
		current += w
		if roll <= current {
			return ep
		}
	}

	// Fallback
	return EndpointGeminiPro
}

func calculateHeuristicScore(prompt string) float64 {
	length := float64(len(prompt))
	h1 := length / 1000.0

	codeChars := strings.Count(prompt, "{") + strings.Count(prompt, "}") + strings.Count(prompt, "def ") + strings.Count(prompt, "func ")
	var h2 float64
	if length > 0 {
		h2 = float64(codeChars*10) / length
	}

	archTerms := []string{"refactor", "microservices", "consensus", "architecture", "state machine"}
	archCount := 0
	lowerPrompt := strings.ToLower(prompt)
	for _, term := range archTerms {
		if strings.Contains(lowerPrompt, term) {
			archCount++
		}
	}
	h3 := float64(archCount) * 0.1

	score := h1 + h2 + h3
	if score > 1.0 {
		return 1.0
	}
	return score
}

// checkWebBrowsingHeuristic evaluates if the prompt requires headless browser extraction or JS execution
func checkWebBrowsingHeuristic(prompt string) bool {
	lowerPrompt := strings.ToLower(prompt)
	webKeywords := []string{
		"openclaw", "scrape", "browse", "fetch", "job", "bounty", "network", "html", "dom", "DOM", "SPA", "react", "vue",
	}

	for _, keyword := range webKeywords {
		if strings.Contains(lowerPrompt, keyword) {
			return true
		}
	}
	return false
}
