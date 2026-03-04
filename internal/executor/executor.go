package executor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gollm/internal/biology"
	"github.com/yourusername/gollm/internal/core"
	"github.com/yourusername/gollm/internal/kinematics"
	"github.com/yourusername/gollm/internal/providers/anthropic"
	"github.com/yourusername/gollm/internal/providers/gemini"
	"github.com/yourusername/gollm/internal/router"
	"github.com/yourusername/gollm/internal/sandbox"
	"github.com/yourusername/gollm/internal/storage"
)

// Executor runs the assigned prompt on the correct model/sandbox based on the routing decision
type Executor struct {
	anthropicProvider core.Provider
	geminiProvider    core.Provider
	mathSandbox       *sandbox.MathVerifier
	store             *storage.Storage
	engine            *kinematics.Engine
	mito              *biology.Mitochondria // Phase 24/25 Integration

	// Circuit breakers for API stability
	anthropicCB *core.CircuitBreaker
	geminiCB    *core.CircuitBreaker

	// Phase 15: Web Execution Engine
	browserEngine *sandbox.BrowserEngine
}

// NewExecutor initializes LLM connections via API keys in the environment.
// If keys are missing, it operates in "Dry Run" mode for testing.
func NewExecutor(store *storage.Storage, eng *kinematics.Engine) *Executor {
	// Initialize Anthropic
	var antProvider core.Provider
	antKey := os.Getenv("ANTHROPIC_API_KEY")
	if antKey != "" {
		provider, err := anthropic.New(anthropic.Config{
			APIKey: antKey,
		})
		if err == nil {
			antProvider = provider // Wrap as interface if we want, or use directly
		}
	}

	// Initialize Gemini
	var gemProvider core.Provider
	gemKey := os.Getenv("GEMINI_API_KEY")
	if gemKey != "" {
		provider, err := gemini.New(gemini.Config{
			APIKey: gemKey,
		})
		if err == nil {
			gemProvider = provider
		}
	}

	// Initialize Phase 15 Web Engine. Can gracefully fail if chromium isn't downloaded yet.
	bEngine, err := sandbox.NewBrowserEngine()
	if err != nil {
		fmt.Printf("[WARNING] Could not initialize Web/JS Engine (Playwright): %v\n", err)
	}

	return &Executor{
		anthropicProvider: antProvider,
		geminiProvider:    gemProvider,
		mathSandbox:       sandbox.NewMathVerifier(),
		store:             store,
		engine:            eng,
		mito:              biology.NewMitochondria(100, 5, 10*time.Second), // Base limit
		anthropicCB:       core.NewCircuitBreaker(3, 5*time.Minute),        // 3 errors = Open for 5 minutes
		geminiCB:          core.NewCircuitBreaker(3, 5*time.Minute),
		browserEngine:     bEngine,
	}
}

// Execute handles the blocking execution.
// Automatically enriches prompts with Long-Term Memory (RAG) using pgvector.
func (e *Executor) Execute(ctx context.Context, endpoint router.Endpoint, systemPrompt, userPrompt, messengerID string) (string, error) {
	// 1. Math Sandbox Execution
	if endpoint == router.EndpointMathVerifier {
		fmt.Println("[SECURITY] Routing to Math Sandbox Container...")
		return e.mathSandbox.Execute(ctx, userPrompt)
	}

	// 2. Anthropic (Claude Opus) Execution
	if endpoint == router.EndpointClaudeOpus {
		// Phase 25: Charge ATP for expensive models
		if err := e.mito.Consume(20); err != nil {
			fmt.Println("[BIOLOGY] ATP Depleted! Penalizing Claude Opus synapse weight and falling back to Gemini.")
			e.engine.UpdateSynapse(string(router.EndpointClaudeOpus), -0.05) // Pain modifier
			return e.Execute(ctx, router.EndpointGeminiPro, systemPrompt, userPrompt, messengerID)
		}

		if !e.anthropicCB.Allow() {
			fmt.Println("[CIRCUIT BREAKER] Anthropic is OPEN/Failing. Falling back to Gemini.")
			// Fallback routing
			return e.Execute(ctx, router.EndpointGeminiPro, systemPrompt, userPrompt, messengerID)
		}

		ragContext, promptEmb := e.enrichWithRAG(ctx, messengerID, userPrompt)
		if ragContext != "" {
			systemPrompt += ragContext
		}

		if e.anthropicProvider == nil {
			return "[DRY RUN - ANTHROPIC_API_KEY NOT SET]\nSimulated response from Claude 4.6.", nil
		}

		fmt.Println("[NETWORK] Calling Claude 4.6 Opus...")
		req := &core.CompletionRequest{
			Model: "claude-3-opus-20240229",
			Messages: []core.Message{
				{Role: core.RoleSystem, Content: systemPrompt},
				{Role: core.RoleUser, Content: userPrompt},
			},
		}

		resp, err := e.anthropicProvider.CreateCompletion(ctx, req)
		if err != nil {
			e.anthropicCB.ReportFailure()
			e.engine.UpdateSynapse(string(router.EndpointClaudeOpus), -0.1) // Pain penalty
			return "", err
		}
		e.anthropicCB.ReportSuccess()
		e.engine.UpdateSynapse(string(router.EndpointClaudeOpus), +0.02) // Success reinforcement

		if len(resp.Choices) > 0 {
			content := resp.Choices[0].Message.Content
			e.asyncSaveMemory(messengerID, userPrompt, content, promptEmb)
			return content, nil
		}
		return "Empty Response from Anthropic", nil
	}

	// 3. Gemini Execution
	if endpoint == router.EndpointGeminiPro {
		// Phase 25: Charge low ATP for cheap models
		if err := e.mito.Consume(5); err != nil {
			fmt.Println("[BIOLOGY] Total ATP Exhaustion. Request denied.")
			e.engine.UpdateSynapse(string(router.EndpointGeminiPro), -0.05)
			return "", err // Complete system exhaust
		}

		if !e.geminiCB.Allow() {
			return "", fmt.Errorf("gemini circuit breaker is OPEN. Complete Failure")
		}

		ragContext, promptEmb := e.enrichWithRAG(ctx, messengerID, userPrompt)
		if ragContext != "" {
			systemPrompt += ragContext
		}

		if e.geminiProvider == nil {
			return "[DRY RUN - GEMINI_API_KEY NOT SET]\nSimulated response from Gemini 3.1 Pro High.", nil
		}

		fmt.Println("[NETWORK] Calling Gemini 3.1 Pro High...")
		req := &core.CompletionRequest{
			Model: "gemini-3.1-pro-high",
			Messages: []core.Message{
				{Role: core.RoleSystem, Content: systemPrompt},
				{Role: core.RoleUser, Content: userPrompt},
			},
		}

		resp, err := e.geminiProvider.CreateCompletion(ctx, req)
		if err != nil {
			e.geminiCB.ReportFailure()
			e.engine.UpdateSynapse(string(router.EndpointGeminiPro), -0.1) // Pain penalty
			return "", err
		}
		e.geminiCB.ReportSuccess()
		e.engine.UpdateSynapse(string(router.EndpointGeminiPro), +0.02) // Success reinforcement

		if len(resp.Choices) > 0 {
			content := resp.Choices[0].Message.Content
			e.asyncSaveMemory(messengerID, userPrompt, content, promptEmb)
			return content, nil
		}
		return "Empty Response from Gemini", nil
	}

	// 4. Autonomous Web Execution Engine
	if endpoint == router.EndpointWebNavigator {
		if e.browserEngine == nil {
			return "", fmt.Errorf("web engine disabled, cannot fulfill browsing request")
		}

		fmt.Println("[WEB ENGINE] Extracting dynamic DOM content...")
		// Hardcoded target for MVP demonstration. Real version extracts URL via prompt parsing/LLM tools
		scrapedText, err := e.browserEngine.NavigateAndExtract("https://example.com", "body")
		if err != nil {
			fmt.Printf("[WARNING] Web extraction failed: %v\n", err)
			scrapedText = "Failed to load dynamic context: " + err.Error()
			e.engine.UpdateSynapse(string(router.EndpointWebNavigator), -0.05)
		} else {
			e.engine.UpdateSynapse(string(router.EndpointWebNavigator), +0.01)
		}

		// Inject scraped DOM text back into the LLM context for final analysis
		systemPrompt += "\n\n<WEB_CONTEXT>\n" + scrapedText + "\n</WEB_CONTEXT>\n"
		systemPrompt += "Analyze the provided Web Context and answer the user's scraping query."

		if e.geminiProvider == nil {
			return "[DRY RUN] Web Agent completed scrape. Gemini missing for final analysis.", nil
		}

		req := &core.CompletionRequest{
			Model: "gemini-3.1-pro-high",
			Messages: []core.Message{
				{Role: core.RoleSystem, Content: systemPrompt},
				{Role: core.RoleUser, Content: userPrompt},
			},
		}

		resp, err := e.geminiProvider.CreateCompletion(ctx, req)
		if err != nil {
			return "", err
		}

		if len(resp.Choices) > 0 {
			return resp.Choices[0].Message.Content, nil
		}
		return "Empty Web Analysis Response", nil
	}

	return "", fmt.Errorf("unknown routing endpoint: %s", endpoint)
}

// GetCircuitStates returns human-readable states for telemetry
func (e *Executor) GetCircuitStates() (anthropicState, geminiState int) {
	return e.anthropicCB.State(), e.geminiCB.State()
}

// GetBPFAlerts returns the Ring-0 isolation alert channel
func (e *Executor) GetBPFAlerts() <-chan string {
	return e.mathSandbox.GetBPFAlerts()
}

// enrichWithRAG performs Vector Database semantic search
func (e *Executor) enrichWithRAG(ctx context.Context, messengerID, userPrompt string) (string, []float32) {
	if e.store == nil || e.geminiProvider == nil || messengerID == "" {
		return "", nil
	}

	embedder, ok := e.geminiProvider.(core.Embedder)
	if !ok {
		return "", nil
	}

	// 1. Embed the user's prompt
	emb, err := embedder.EmbedText(ctx, userPrompt)
	if err != nil || len(emb) == 0 {
		return "", nil
	}

	// 2. Fetch Top-5 nearest neighbors
	memories, err := e.store.SearchSimilarMemories(ctx, messengerID, emb, 5)
	if err != nil || len(memories) == 0 {
		return "", emb
	}

	// 3. Format context injected into System Prompt
	ragContext := "\n\n<RELEVANT_LONG_TERM_MEMORY>\n"
	ragContext += "Below are relevant memories from past conversations with this specific user. Use them to maintain context and adapt to their preferences (Do not explicitly mention searching memory unless asked):\n"
	for _, m := range memories {
		ragContext += fmt.Sprintf("[%s] (Distance: %.2f): %s\n", m.Role, m.Distance, m.Content)
	}
	ragContext += "</RELEVANT_LONG_TERM_MEMORY>\n"

	return ragContext, emb
}

// asyncSaveMemory computes response embeddings and commits both prompt and output to the database
func (e *Executor) asyncSaveMemory(messengerID, reqPrompt, respContent string, promptEmb []float32) {
	if e.store == nil || len(promptEmb) == 0 {
		return
	}

	// Fork into background goroutine to not block the LLM delivery
	go func() {
		// Time limit for downstream DB operations
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// 1. Save User Prompt
		_ = e.store.SaveMemory(ctx, messengerID, "user", reqPrompt, promptEmb)

		// 2. Embed & Save Assistant Response
		if embedder, ok := e.geminiProvider.(core.Embedder); ok {
			respEmb, err := embedder.EmbedText(ctx, respContent)
			if err == nil && len(respEmb) > 0 {
				_ = e.store.SaveMemory(ctx, messengerID, "assistant", respContent, respEmb)
			}
		}
	}()
}

// BrowserNavigateAndExtract executes stealth DOM extraction via Playwright.
func (e *Executor) BrowserNavigateAndExtract(ctx context.Context, url, selector string) (string, error) {
	if e.browserEngine == nil {
		return "", fmt.Errorf("web execution engine is not initialized or failed to start")
	}
	fmt.Printf("[WEB ENGINE] Navigating to: %s\n", url)
	return e.browserEngine.NavigateAndExtract(url, selector)
}

// BrowserEvaluateJS safely executes a JS script via Goja locally.
func (e *Executor) BrowserEvaluateJS(ctx context.Context, script string) (string, error) {
	if e.browserEngine == nil {
		return "", fmt.Errorf("web execution engine is not initialized or failed to start")
	}
	fmt.Printf("[JS ENGINE] Evaluating local JavaScript snippet...\n")
	return e.browserEngine.EvaluateJS(ctx, script)
}
