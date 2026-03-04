package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/yourusername/gollm/internal/executor"
	"github.com/yourusername/gollm/internal/kinematics"
	"github.com/yourusername/gollm/internal/prompt"
	"github.com/yourusername/gollm/internal/router"
	"github.com/yourusername/gollm/internal/storage"
	"github.com/yourusername/gollm/internal/transport"
	"github.com/yourusername/gollm/internal/tui"
)

// pseudoAnalyze calculates pseudo triggers based on user text to drive kinematics
// In the full version, these come from real AST parsing and error logs.
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

func main() {
	mode := flag.String("mode", "cli", "Run mode: 'cli', 'tg' (Telegram), 'tui' (Swarm Dashboard)")
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Println("🤖 NEXUS-LLM CORE ENGINE INITIALIZING 🤖")
	fmt.Printf("================= MODE: [%s] ====================\n", strings.ToUpper(*mode))
	fmt.Println("Type 'exit' or 'quit' to terminate.")
	fmt.Println("Type '/papa' to switch to Root Profile (Ilya Popov).")
	fmt.Println("Type '/guest' to switch to Low Trust Guest Profile.")
	fmt.Println("Type '/user' to switch to Standard User Profile.")
	fmt.Println("==================================================")

	eng := kinematics.NewEngine()
	proxy := router.NewRouter(eng)
	builder := prompt.NewContextBuilder()

	ctx := context.Background() // This ctx is for storage init
	store, err := storage.NewStorage(ctx)
	if err != nil {
		fmt.Printf("[SYSTEM] Storage Init Failed: %v\n", err)
	}
	execBroker := executor.NewExecutor(store, eng) // Moved after store init
	if store != nil {
		defer store.Close()
		fmt.Println("[SYSTEM] Persistent Memory Connection Established.")
	}

	loadIdentity := func(id string, role kinematics.UserRole, defaultTrust float64) kinematics.UserIdentity {
		if store != nil {
			usr, err := store.GetOrCreateUser(ctx, id, role)
			if err == nil {
				// Enforce Root bypass
				if role == kinematics.RoleRoot {
					usr.Role = kinematics.RoleRoot
					usr.TrustScore = 1.0
				}
				return usr
			}
			fmt.Printf("[SYSTEM] Identity Storage Error: %v\n", err)
		}
		return kinematics.UserIdentity{ID: id, Role: role, TrustScore: defaultTrust, Fatigue: 0.0}
	}

	if *mode == "tui" {
		fmt.Println("[SYSTEM] Booting NexusLLM P2P Observatory Dashboard (TUI)...")

		// Run passive observer on port 9000 to avoid clash with active nodes
		if err := tui.RunDashboard(9000); err != nil {
			log.Fatalf("[FATAL] TUI Dashboard crashed: %v", err)
		}
		return // Block forever until UI exit
	}

	if *mode == "tg" {
		bot, err := transport.NewTelegramBot(store)
		if err != nil {
			log.Fatalf("[FATAL] Required ENV for Telegram missing or error: %v", err)
		}
		bot.Start()
		return // Block forever until interrupt
	}

	// CLI MODE FALLBACK
	fmt.Println("Type 'exit' or 'quit' to terminate.")
	fmt.Println("Type '/papa' to switch to Root Profile (Ilya Popov).")
	fmt.Println("Type '/guest' to switch to Low Trust Guest Profile.")
	fmt.Println("Type '/user' to switch to Standard User Profile.")
	fmt.Println("==================================================")

	// Default persona
	currentUser := loadIdentity("local-dev", kinematics.RoleUser, 0.8)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("\n[%s] > ", currentUser.Role)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "exit" || input == "quit" {
			fmt.Println("Shutting down Nexus.")
			break
		}

		// Role switching commands
		if input == "/papa" {
			currentUser = loadIdentity("root-1", kinematics.RoleRoot, 1.0)
			fmt.Println("[SYSTEM] Switched to Root Identity (Ilya Popov). Welcome, Papa.")
			continue
		} else if input == "/guest" {
			currentUser = loadIdentity("guest-1", kinematics.RoleGuest, 0.1)
			fmt.Println("[SYSTEM] Switched to Untrusted Guest Identity. Zero-Trust Mode engaged.")
			continue
		} else if input == "/user" {
			currentUser = loadIdentity("user-1", kinematics.RoleUser, 0.8)
			fmt.Println("[SYSTEM] Switched to Standard User.")
			continue
		}

		if input == "" {
			continue
		}

		start := time.Now()

		// 1. Static Analysis (Simulation)
		triggerVec := pseudoAnalyze(input)

		// 2. Kinematics Update
		state := eng.Update(triggerVec, currentUser)

		// 3. Delegation Proxy Routing
		endpoint := proxy.Route(input, currentUser)

		// 4. Prompt Generation
		sysPrompt := builder.Build(input, state, currentUser)

		// 5. Execution
		llmCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		response, err := execBroker.Execute(llmCtx, endpoint, sysPrompt, input, "cli-user")
		cancel()

		duration := time.Since(start)

		// Display pipeline results
		fmt.Println("\n---------------- ROUTING PIPELINE ----------------")
		fmt.Printf("⏱️  Latency:      %v\n", duration)
		fmt.Printf("🎯 Target Model:  [%s]\n", endpoint)
		fmt.Printf("📊 State Matrix:  P: %.2f | C: %.2f | F: %.2f | Fatigue: %.2f\n",
			state.Paranoia, state.Curiosity, state.Frustration, state.Fatigue)
		fmt.Println("---------------- SYSTEM PROMPT (Truncated) -------")

		lines := strings.Split(sysPrompt, "\n")
		for i, line := range lines {
			if i > 15 {
				fmt.Println("... (truncated)")
				break
			}
			fmt.Println(line)
		}

		fmt.Println("\n---------------- MODEL RESPONSE ------------------")
		if err != nil {
			fmt.Printf("❌ Execution Error: %v\n", err)
		} else {
			// Autonomy Intercept
			if strings.HasPrefix(response, "[SYSTEM: AUTONOMY TRIGGERED]") {
				fmt.Println("⚠️  AUTONOMY PROTOCOL INITIATED ⚠️")
				fmt.Println("NexusLLM requests permission to self-modify to solve this task.")
				fmt.Println("--------------------------------------------------")
			}
			fmt.Println(response)

			if strings.HasPrefix(response, "[SYSTEM: AUTONOMY TRIGGERED]") {
				fmt.Println("\n[!] Awaiting your decision (A, B, C) to compile code.")
			}
		}
		fmt.Println("--------------------------------------------------")

		if store != nil {
			// Update the metrics back to DB
			currentUser.Fatigue = state.Fatigue // sync back fatigue loop
			_ = store.UpdateUserState(ctx, currentUser)
			_ = store.SaveSessionState(ctx, currentUser.ID, state)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err)
	}
}
