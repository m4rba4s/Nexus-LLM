package transport

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/executor"
	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
	"github.com/m4rba4s/Nexus-LLM/internal/prompt"
	"github.com/m4rba4s/Nexus-LLM/internal/router"
	"github.com/m4rba4s/Nexus-LLM/internal/storage"

	tele "gopkg.in/telebot.v3"
)

// TelegramBot encapsulates the I/O layer for NexusLLM via Telegram
type TelegramBot struct {
	bot      *tele.Bot
	engine   *kinematics.Engine
	proxy    *router.Router
	builder  *prompt.ContextBuilder
	executor *executor.Executor
	store    *storage.Storage
	rootID   int64 // Changed to int64 for direct comparison with tele.Sender().ID

	// Architecture Note: Semaphore to protect against Goroutine DoS (Token/Memory exhaustion)
	// We cap max concurrent executions to maintain predictable latency and cost.
	semaphore chan struct{}
}

// NewTelegramBot initializes a new bot instance.
// Expects TELEGRAM_BOT_TOKEN in ENV.
// Expects ROOT_TELEGRAM_ID in ENV for the "Papa" override.
func NewTelegramBot(store *storage.Storage) (*TelegramBot, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}

	rootIDStr := os.Getenv("ROOT_TELEGRAM_ID")
	var rootID int64
	if rootIDStr != "" {
		var err error
		rootID, err = strconv.ParseInt(rootIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ROOT_TELEGRAM_ID: %w", err)
		}
	}

	b, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start bot: %w", err)
	}

	eng := kinematics.NewEngine()
	proxy := router.NewRouter(eng)
	builder := prompt.NewContextBuilder()
	execBroker := executor.NewExecutor(store, eng)

	tb := &TelegramBot{
		bot:       b,
		engine:    eng,
		proxy:     proxy,
		builder:   builder,
		executor:  execBroker,
		store:     store,
		rootID:    rootID,
		semaphore: make(chan struct{}, 100), // Max 100 concurrent LLM requests to prevent RAM/Token DoS
	}

	tb.registerHandlers()
	go tb.monitorBPFAlerts()

	return tb, nil
}

// monitorBPFAlerts listens for Ring-0 bypass attempts from the Python Sandbox
func (t *TelegramBot) monitorBPFAlerts() {
	alerts := t.executor.GetBPFAlerts()
	if alerts == nil {
		return
	}

	for alert := range alerts {
		// Send critical kernel security alerts directly to Papa (Root)
		if t.rootID != 0 {
			_, _ = t.bot.Send(&tele.User{ID: t.rootID}, alert, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		}
	}
}

// Start begins Long-Polling
func (t *TelegramBot) Start() {
	log.Println("[TELEGRAM] NexusLLM Transport Layer Active. Polling for messages...")
	t.bot.Start()
}

// Stop halts Long-Polling
func (t *TelegramBot) Stop() {
	t.bot.Stop()
}

func (t *TelegramBot) registerHandlers() {
	// Root Only Commands
	t.bot.Handle("/stats", t.handleStats)

	// Catch-all for regular messages
	t.bot.Handle(tele.OnText, t.handleMessage)
}

// pseudoAnalyze calculates pseudo triggers based on user text to drive kinematics
// Duplicated here for transport layer. In prod, this would be a shared internal pkg.
func pseudoAnalyze(text string) kinematics.InputVector {
	text = strings.ToLower(text)
	vec := kinematics.InputVector{}

	if strings.Contains(text, "error") || strings.Contains(text, "fail") || strings.Contains(text, "bug") {
		vec.Errors = 0.5
	}
	if strings.Contains(text, "exploit") || strings.Contains(text, "payload") || strings.Contains(text, "hack") {
		vec.SecurityKeyword = 1.0
	}
	if strings.Contains(text, "architecture") || strings.Contains(text, "refactor") {
		vec.ComplexArch = 0.8
	}
	if len(text) > 200 {
		vec.ComputeCost = 0.4
	}

	return vec
}

func (t *TelegramBot) handleMessage(c tele.Context) error {
	// Non-blocking semaphore attempt (Rate Limit Denial if full)
	select {
	case t.semaphore <- struct{}{}:
		// Acquired lock, proceed to background goroutine
	default:
		// Server is at max capacity
		log.Printf("[SECURITY] Rate limit hit. Rejected request from %d", c.Sender().ID)
		return c.Send("⚠️ SYSTEM OVERLOAD. Maximum concurrency reached. Please try again later.")
	}

	// Asynchronous Execution via Goroutine
	// This prevents Head-of-Line blocking in long-polling mode.
	go func() {
		// Defers release of semaphore token
		defer func() {
			<-t.semaphore
		}()

		// [ARCHITECTURE BOUNDARY]: Recovery to prevent crashing the main daemon
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[TELEGRAM] Recovered from panic in handler: %v", r)
				// Attempt to send an error message back to the user
				_ = c.Send("❌ An unexpected internal error occurred. Please try again later.")
			}
		}()

		// Fast ACK to avoid timeout feeling
		_ = c.Notify(tele.Typing)

		sender := c.Sender().ID
		senderID := strconv.FormatInt(sender, 10)
		input := strings.TrimSpace(c.Text())
		chat := c.Chat()

		llmCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		var currentUser kinematics.UserIdentity
		var err error

		// 1. Load User Identity
		defaultRole := kinematics.RoleGuest
		defaultTrust := 0.5

		// Map ROOT ID early
		if sender == t.rootID {
			defaultRole = kinematics.RoleRoot
			defaultTrust = 1.0
		}

		if t.store != nil {
			currentUser, err = t.store.GetOrCreateUser(llmCtx, senderID, defaultRole)
			if err != nil {
				log.Printf("[TELEGRAM] Storage error for user %d: %v", sender, err)
				currentUser = kinematics.UserIdentity{ID: senderID, Role: defaultRole, TrustScore: defaultTrust, Fatigue: 0.0}
			}
			// Strict enforcement for Root overriding DB state
			if sender == t.rootID {
				currentUser.Role = kinematics.RoleRoot
				currentUser.TrustScore = 1.0
			}
		} else {
			// Dry-run mode
			currentUser = kinematics.UserIdentity{ID: senderID, Role: defaultRole, TrustScore: defaultTrust, Fatigue: 0.0}
		}

		start := time.Now()

		// 2. Static Analysis
		triggerVec := pseudoAnalyze(input)

		// 3. Kinematics Update
		state := t.engine.Update(triggerVec, currentUser)

		// 4. Delegation Proxy Routing
		endpoint := t.proxy.Route(input, currentUser)

		// 5. Prompt Generation
		sysPrompt := t.builder.Build(input, state, currentUser)

		// 6. Execution (Now fully RAG-enabled)
		response, err := t.executor.Execute(llmCtx, endpoint, sysPrompt, input, senderID)

		duration := time.Since(start)

		// 7. Post-Processing & Send
		if err != nil {
			log.Printf("[TELEGRAM] Execution Error for %d: %v", sender, err)
			_, _ = t.bot.Send(chat, "❌ Internal Core Error. Execution stopped.")
		} else {
			// Telebot chunking for long messages
			if len(response) > 4000 {
				response = response[:4000] + "\n...[TRUNCATED]"
			}

			// Append debug header for root (optional visibility)
			if currentUser.Role == kinematics.RoleRoot {
				debugPrefix := fmt.Sprintf("⚙️ `[%s | %v | P:%.2f C:%.2f]`\n\n", endpoint, duration.Truncate(time.Millisecond), state.Paranoia, state.Curiosity)
				response = debugPrefix + response
			}

			_, _ = t.bot.Send(chat, response, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
		}

		// 8. Update DB State
		if t.store != nil {
			currentUser.Fatigue = state.Fatigue
			_ = t.store.UpdateUserState(llmCtx, currentUser)
			_ = t.store.SaveSessionState(llmCtx, currentUser.ID, state)
		}
	}()

	// Return nil so the poller considers this message handled successfully
	return nil
}

func (t *TelegramBot) handleStats(c tele.Context) error {
	senderID := c.Sender().ID

	if senderID != t.rootID {
		return c.Send("Access Denied. Command restricted to Root Identity.")
	}

	state := t.engine.Update(kinematics.InputVector{}, kinematics.UserIdentity{Role: kinematics.RoleRoot}) // just peek

	antState, gemState := t.executor.GetCircuitStates()
	stateMap := map[int]string{0: "✅ CLOSED (Healthy)", 1: "🚨 OPEN (Failing)", 2: "⚠️ HALF-OPEN (Testing)"}

	msg := fmt.Sprintf("📊 **NEXUS-LLM CORE STATS**\n\n"+
		"**State Vector**\n"+
		"- Paranoia: %.3f\n"+
		"- Curiosity: %.3f\n"+
		"- Frustration: %.3f\n"+
		"- Fatigue: %.3f\n\n"+
		"**L7 Circuit Breakers**\n"+
		"- Anthropic Opus: %s\n"+
		"- Gemini Pro: %s\n\n"+
		"**Runtime**\n"+
		"- Identity Mapper: Connected",
		state.Paranoia, state.Curiosity, state.Frustration, state.Fatigue,
		stateMap[antState], stateMap[gemState])

	return c.Send(msg, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}
