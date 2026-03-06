package executor

import (
	"context"
	"testing"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/biology"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
	"github.com/m4rba4s/Nexus-LLM/internal/providers/mock"
	"github.com/m4rba4s/Nexus-LLM/internal/router"
	"github.com/m4rba4s/Nexus-LLM/internal/sandbox"
)

// newTestExecutor builds an Executor with mock providers and no DB/storage.
func newTestExecutor(antProvider, gemProvider core.Provider) *Executor {
	return &Executor{
		anthropicProvider: antProvider,
		geminiProvider:    gemProvider,
		mathSandbox:       sandbox.NewMathVerifier(),
		engine:            kinematics.NewEngine(),
		mito:              biology.NewMitochondria(100, 5, 10*time.Second),
		anthropicCB:       core.NewCircuitBreaker(3, 5*time.Minute),
		geminiCB:          core.NewCircuitBreaker(3, 5*time.Minute),
		// store and browserEngine intentionally nil for unit tests
	}
}

func mockProvider(response string) *mock.Provider {
	cfg := mock.DefaultConfig()
	p := mock.New(cfg)
	p.SetGlobalResponse(response)
	return p
}

// --- Tests ---

func TestExecute_DryRun_Anthropic(t *testing.T) {
	t.Parallel()

	exec := newTestExecutor(nil, nil)
	ctx := context.Background()

	resp, err := exec.Execute(ctx, router.EndpointClaudeOpus, "system", "hello", "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty dry run response")
	}

	const want = "[DRY RUN - ANTHROPIC_API_KEY NOT SET]"
	if len(resp) < len(want) || resp[:len(want)] != want {
		t.Errorf("dry run response = %q, want prefix %q", resp, want)
	}
}

func TestExecute_DryRun_Gemini(t *testing.T) {
	t.Parallel()

	exec := newTestExecutor(nil, nil)
	ctx := context.Background()

	resp, err := exec.Execute(ctx, router.EndpointGeminiPro, "system", "hello", "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const want = "[DRY RUN - GEMINI_API_KEY NOT SET]"
	if len(resp) < len(want) || resp[:len(want)] != want {
		t.Errorf("dry run response = %q, want prefix %q", resp, want)
	}
}

func TestExecute_MockAnthropic_Success(t *testing.T) {
	t.Parallel()

	p := mockProvider("Hello from Claude!")
	exec := newTestExecutor(p, nil)
	ctx := context.Background()

	resp, err := exec.Execute(ctx, router.EndpointClaudeOpus, "system", "test prompt", "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Hello from Claude!" {
		t.Errorf("response = %q, want %q", resp, "Hello from Claude!")
	}
}

func TestExecute_MockGemini_Success(t *testing.T) {
	t.Parallel()

	p := mockProvider("Hello from Gemini!")
	exec := newTestExecutor(nil, p)
	ctx := context.Background()

	resp, err := exec.Execute(ctx, router.EndpointGeminiPro, "system", "test prompt", "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "Hello from Gemini!" {
		t.Errorf("response = %q, want %q", resp, "Hello from Gemini!")
	}
}

func TestExecute_CircuitBreaker_FallbackToGemini(t *testing.T) {
	t.Parallel()

	// Anthropic provider fails 3 times → circuit breaker opens → falls back to Gemini.
	antP := mockProvider("")
	antP.SetGlobalError(core.ErrTimeout)

	gemP := mockProvider("Gemini fallback response")

	exec := newTestExecutor(antP, gemP)
	ctx := context.Background()

	// Trip the circuit breaker
	for i := 0; i < 3; i++ {
		_, _ = exec.Execute(ctx, router.EndpointClaudeOpus, "sys", "fail me", "user1")
	}

	// Next call to Claude should fall back to Gemini
	resp, err := exec.Execute(ctx, router.EndpointClaudeOpus, "system", "hello", "user1")
	if err != nil {
		t.Fatalf("fallback should succeed, got error: %v", err)
	}
	if resp != "Gemini fallback response" {
		t.Errorf("expected Gemini fallback, got %q", resp)
	}
}

func TestExecute_ATP_Depletion_Fallback(t *testing.T) {
	t.Parallel()

	gemP := mockProvider("Gemini ATP fallback")
	// Create executor with very low ATP so Claude path exhausts it quickly
	exec := &Executor{
		anthropicProvider: mockProvider("never see this"),
		geminiProvider:    gemP,
		mathSandbox:       sandbox.NewMathVerifier(),
		engine:            kinematics.NewEngine(),
		mito:              biology.NewMitochondria(15, 5, 10*time.Second), // Only 15 ATP
		anthropicCB:       core.NewCircuitBreaker(3, 5*time.Minute),
		geminiCB:          core.NewCircuitBreaker(3, 5*time.Minute),
	}

	ctx := context.Background()

	// Claude costs 20 ATP, we only have 15 → falls back to Gemini
	resp, err := exec.Execute(ctx, router.EndpointClaudeOpus, "sys", "expensive", "user1")
	if err != nil {
		t.Fatalf("ATP fallback to Gemini should succeed, got error: %v", err)
	}
	if resp != "Gemini ATP fallback" {
		t.Errorf("expected Gemini fallback, got %q", resp)
	}
}

func TestExecute_MathSandbox(t *testing.T) {
	t.Parallel()

	exec := newTestExecutor(nil, nil)
	ctx := context.Background()

	resp, err := exec.Execute(ctx, router.EndpointMathVerifier, "", "2+2", "user1")
	if err != nil {
		// Math sandbox requires Linux namespaces; skip in unprivileged environments
		t.Skipf("math sandbox unavailable (expected in CI/non-root): %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty math response")
	}
}

func TestExecute_WebNavigator_Disabled(t *testing.T) {
	t.Parallel()

	exec := newTestExecutor(nil, nil) // browserEngine is nil
	ctx := context.Background()

	_, err := exec.Execute(ctx, router.EndpointWebNavigator, "sys", "scrape", "user1")
	if err == nil {
		t.Fatal("expected error when browser engine is nil")
	}
}

func TestGetCircuitStates(t *testing.T) {
	t.Parallel()

	exec := newTestExecutor(nil, nil)
	antState, gemState := exec.GetCircuitStates()

	// Both should be Closed (0) initially
	if antState != 0 {
		t.Errorf("initial anthropic CB state = %d, want 0 (Closed)", antState)
	}
	if gemState != 0 {
		t.Errorf("initial gemini CB state = %d, want 0 (Closed)", gemState)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	t.Parallel()

	p := mockProvider("")
	p.SetGlobalLatency(5 * time.Second)

	exec := newTestExecutor(p, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := exec.Execute(ctx, router.EndpointClaudeOpus, "sys", "slow", "user1")
	if err == nil {
		t.Fatal("expected timeout/cancellation error")
	}
}
