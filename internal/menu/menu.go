package menu

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "strconv"
    "strings"
    "time"

    "github.com/m4rba4s/Nexus-LLM/internal/config"
    "github.com/m4rba4s/Nexus-LLM/internal/core"
    "github.com/m4rba4s/Nexus-LLM/internal/modes/coder"
    "github.com/m4rba4s/Nexus-LLM/internal/modes/operator"
    branding "github.com/m4rba4s/Nexus-LLM/internal/branding"

	// Import providers to register them
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/anthropic"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/deepseek"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/gemini"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/openai"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/openrouter"
)

// Settings holds current menu settings
type Settings struct {
    Provider string
    Model    string
    CoderOp  coder.Operator
}

// Options allows preselecting initial menu state from CLI flags.
type Options struct {
    Provider string
    Model    string
    CoderOp  string // codegen|refactor|test|docs
}

// Run starts the main menu loop
func Run(ctx context.Context, in io.Reader, out io.Writer) error {
    return RunWithOptions(ctx, in, out, Options{})
}

// RunWithOptions starts the menu loop with preselected options.
func RunWithOptions(ctx context.Context, in io.Reader, out io.Writer, opts Options) error {
    // Load config, don't fail if not exists
    cfg, err := config.Load()
    if err != nil {
        cfg = &config.Config{
            DefaultProvider: "openrouter",
            Settings: config.GlobalSettings{
                OutputFormat: "text",
            },
        }
        cfg.Settings.ApplyDefaults()
    }

    // Initialize settings from config
    state := Settings{
        Provider: cfg.DefaultProvider,
        CoderOp:  coder.OperatorCodegen,
    }

    // Ensure provider config exists and is valid; fallback to openrouter+k2 (free)
    if cfg.Providers == nil {
        cfg.Providers = make(map[string]config.ProviderConfig)
    }
    // If default provider is empty or not configured properly, set openrouter
    if state.Provider == "" || !cfg.HasProvider(state.Provider) || cfg.Providers[state.Provider].Type == "" {
        state.Provider = "openrouter"
        pc := cfg.Providers[state.Provider]
        pc.Type = "openrouter"
        if pc.DefaultModel == "" {
            pc.DefaultModel = "k2" // Free Kimi K2 default
        }
        cfg.Providers[state.Provider] = pc
        cfg.DefaultProvider = state.Provider
    }
    // If default provider is openai but no API key set, prefer openrouter+k2
    if strings.EqualFold(state.Provider, "openai") {
        if p, ok := cfg.Providers[state.Provider]; ok {
            if p.APIKey.IsEmpty() {
                state.Provider = "openrouter"
                pc := cfg.Providers[state.Provider]
                pc.Type = "openrouter"
                if pc.DefaultModel == "" { pc.DefaultModel = "k2" }
                cfg.Providers[state.Provider] = pc
                cfg.DefaultProvider = state.Provider
            }
        }
    }

    // Try to get model from config
    if cfg.Providers != nil && cfg.Providers[state.Provider].DefaultModel != "" {
        state.Model = cfg.Providers[state.Provider].DefaultModel
    } else {
        // Prefer free Kimi K2 on OpenRouter by default
        if state.Provider == "openrouter" {
            state.Model = "k2"
        } else {
            state.Model = "k2"
        }
    }

    // Apply preselected options if provided
    if opts.Provider != "" {
        state.Provider = opts.Provider
    }
    if opts.Model != "" {
        state.Model = opts.Model
    }
    if opts.CoderOp != "" {
        switch strings.ToLower(opts.CoderOp) {
        case "codegen", "gen", "generate":
            state.CoderOp = coder.OperatorCodegen
        case "refactor", "ref":
            state.CoderOp = coder.OperatorRefactor
        case "test", "tests":
            state.CoderOp = coder.OperatorTest
        case "docs", "doc", "documentation":
            state.CoderOp = coder.OperatorDocs
        }
    }

    r := bufio.NewReader(in)

    // Clear screen and show compact colored logo
    fmt.Fprintf(out, "\033[2J\033[H")
    branding.DisplayLogo(branding.LogoOptions{
        ShowTagline:   true,
        Colored:       true,
        Compact:       true,
        CenterAlign:   false,
        CustomTagline: "⚙️  Operator • 🧠 Coder • 🔗 Multi‑Provider",
    })

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Show header
		fmt.Fprintln(out, "\n╔═══════════════════════════════════════════╗")
		fmt.Fprintln(out, "║            GOLLM OPERATOR                 ║")
		fmt.Fprintln(out, "╚═══════════════════════════════════════════╝")
		fmt.Fprintf(out, "│ Provider: %-10s Model: %-15s\n", state.Provider, state.Model)
		fmt.Fprintf(out, "│ Coder Op: %-10s\n", state.CoderOp)
		fmt.Fprintln(out, "├───────────────────────────────────────────")
		fmt.Fprintln(out, "│ 1) PC-Operator  - System operations")
		fmt.Fprintln(out, "│ 2) Coder        - Development tasks")
		fmt.Fprintln(out, "│ 3) Settings     - Configure provider/model")
		fmt.Fprintln(out, "│ 0) Exit")
		fmt.Fprintln(out, "└───────────────────────────────────────────")
		fmt.Fprint(out, "Select > ")

		line, err := r.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		choice := strings.TrimSpace(line)

		switch choice {
        case "1":
            if err := runPCOperator(ctx, r, out, state); err != nil {
                fmt.Fprintf(out, "\n[ERROR] %v\n", err)
            }

        case "2":
            if err := runCoder(ctx, r, out, state, cfg); err != nil {
                fmt.Fprintf(out, "\n[ERROR] %v\n", err)
            }

		case "3":
			if err := settings(ctx, r, out, &state, cfg); err != nil {
				fmt.Fprintf(out, "\n[ERROR] %v\n", err)
			} else {
				// Save config
				if cfg.Providers == nil {
					cfg.Providers = make(map[string]config.ProviderConfig)
				}
				cfg.DefaultProvider = state.Provider
				if pc, exists := cfg.Providers[state.Provider]; exists {
					pc.DefaultModel = state.Model
					cfg.Providers[state.Provider] = pc
				}
				// Save coder operator to config extra
				if cfg.Extra == nil {
					cfg.Extra = make(map[string]interface{})
				}
				cfg.Extra["coder_operator"] = string(state.CoderOp)

				_ = config.Save(cfg) // Ignore errors for now
			}

		case "0", "q", "quit", "exit":
			fmt.Fprintln(out, "\nGoodbye!")
			return nil

		default:
			fmt.Fprintln(out, "\n[ERROR] Invalid selection")
		}
	}
}

func settings(ctx context.Context, r *bufio.Reader, out io.Writer, s *Settings, cfg *config.Config) error {
	// Clear screen
	fmt.Fprintf(out, "\033[2J\033[H")
	fmt.Fprint(out, "\n=== SETTINGS ===\n")

	// 1. Select Provider
	registry := core.GetGlobalRegistry()
	providers := registry.List()
	if len(providers) == 0 {
		// Fallback to hardcoded list if registry is empty
		providers = []string{"openai", "anthropic", "deepseek", "gemini", "openrouter"}
	}

	fmt.Fprintln(out, "Available Providers:")
	for i, p := range providers {
		fmt.Fprintf(out, "  %d) %s\n", i+1, p)
	}
	fmt.Fprint(out, "\nSelect provider (1-", len(providers), "): ")

	pidx, err := readIndex(r, len(providers))
	if err != nil {
		return err
	}
	s.Provider = providers[pidx]

	// 2. Select Model
	fmt.Fprintln(out, "\nFetching models...")

	// Get provider from registry or config
	var models []core.Model
	if registry != nil {
		if prov, err := registry.Get(s.Provider); err == nil {
			if modelProvider, ok := prov.(core.ModelProvider); ok {
				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				if modelList, err := modelProvider.ListModels(ctx); err == nil {
					models = modelList
				}
			}
		}
	}

	// Fallback to predefined models if API call fails
	if len(models) == 0 {
		models = getDefaultModels(s.Provider)
	}

	fmt.Fprintln(out, "\nAvailable Models:")
	for i, m := range models {
		fmt.Fprintf(out, "  %d) %s\n", i+1, m.ID)
		if m.Description != "" {
			fmt.Fprintf(out, "     %s\n", m.Description)
		}
	}
	fmt.Fprint(out, "\nSelect model (1-", len(models), "): ")

	midx, err := readIndex(r, len(models))
	if err != nil {
		return err
	}
	s.Model = models[midx].ID

	// 3. Select Coder Operator
	operators := []coder.Operator{
		coder.OperatorCodegen,
		coder.OperatorRefactor,
		coder.OperatorTest,
		coder.OperatorDocs,
	}

	fmt.Fprintln(out, "\nCoder Operators:")
	fmt.Fprintln(out, "  1) Codegen  - Generate new code")
	fmt.Fprintln(out, "  2) Refactor - Improve existing code")
	fmt.Fprintln(out, "  3) Test     - Write unit tests")
	fmt.Fprintln(out, "  4) Docs     - Generate documentation")
	fmt.Fprint(out, "\nSelect operator (1-4): ")

	oidx, err := readIndex(r, len(operators))
	if err != nil {
		return err
	}
	s.CoderOp = operators[oidx]

	fmt.Fprintln(out, "\n[INFO] Settings updated successfully")
	fmt.Fprintln(out, "Press Enter to continue...")
	r.ReadString('\n')

	return nil
}

func runPCOperator(ctx context.Context, r *bufio.Reader, out io.Writer, s Settings) error {
	for {
		// Clear screen
		fmt.Fprintf(out, "\033[2J\033[H")
		fmt.Fprint(out, "\n=== PC-OPERATOR MODE ===\n")
		fmt.Fprintln(out, "1) Run command       - Execute shell command")
		fmt.Fprintln(out, "2) Edit file         - Modify file content")
		fmt.Fprintln(out, "3) Manage service    - Start/stop services")
		fmt.Fprintln(out, "4) Install package   - Package management")
		fmt.Fprintln(out, "5) Request sudo      - Elevate privileges")
		fmt.Fprintln(out, "0) Back to main menu")
		fmt.Fprint(out, "\nSelect > ")

		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}

		choice := strings.TrimSpace(line)

		switch choice {
		case "1":
			if err := operator.RunCommandFlow(ctx, r, out); err != nil {
				fmt.Fprintf(out, "\n[ERROR] %v\n", err)
				fmt.Fprintln(out, "Press Enter to continue...")
				r.ReadString('\n')
			}

		case "2":
			if err := operator.EditFileFlow(ctx, r, out); err != nil {
				fmt.Fprintf(out, "\n[ERROR] %v\n", err)
				fmt.Fprintln(out, "Press Enter to continue...")
				r.ReadString('\n')
			}

		case "3":
			if err := operator.ServiceFlow(ctx, r, out); err != nil {
				fmt.Fprintf(out, "\n[ERROR] %v\n", err)
				fmt.Fprintln(out, "Press Enter to continue...")
				r.ReadString('\n')
			}

		case "4":
			if err := operator.PackageFlow(ctx, r, out); err != nil {
				fmt.Fprintf(out, "\n[ERROR] %v\n", err)
				fmt.Fprintln(out, "Press Enter to continue...")
				r.ReadString('\n')
			}

		case "5":
			ok, err := operator.RequestElevation(ctx, "User requested admin privileges")
			if err != nil {
				fmt.Fprintf(out, "\n[ERROR] Elevation failed: %v\n", err)
			} else if ok {
				fmt.Fprintln(out, "\n[INFO] Admin privileges granted for this session")
			} else {
				fmt.Fprintln(out, "\n[INFO] Admin privileges denied")
			}
			fmt.Fprintln(out, "Press Enter to continue...")
			r.ReadString('\n')

		case "0":
			return nil

		default:
			fmt.Fprintln(out, "\n[ERROR] Invalid selection")
		}
	}
}

func runCoder(ctx context.Context, r *bufio.Reader, out io.Writer, s Settings, cfg *config.Config) error {
	// Get provider from registry
	registry := core.GetGlobalRegistry()
	if registry == nil {
		return fmt.Errorf("provider registry not initialized")
	}

	// Check if provider is configured
	if cfg.Providers == nil || cfg.Providers[s.Provider].APIKey.IsEmpty() {
		return fmt.Errorf("provider %s not configured, please set API key in config", s.Provider)
	}

	// Create provider config
	provConfig := core.ProviderConfig{
		Type:         s.Provider,
		APIKey:       cfg.Providers[s.Provider].APIKey.Value(),
		BaseURL:      cfg.Providers[s.Provider].BaseURL,
		Organization: cfg.Providers[s.Provider].Organization,
		MaxRetries:   cfg.Providers[s.Provider].MaxRetries,
		Timeout:      cfg.Providers[s.Provider].Timeout,
	}

	// Get or create provider
	provider, err := registry.Get(s.Provider)
	if err != nil {
		// Try to create provider
		provider, err = registry.CreateProvider(s.Provider, provConfig)
		if err != nil {
			return fmt.Errorf("failed to create provider: %w", err)
		}
		// Register it
		if err := registry.Register(s.Provider, provider); err != nil {
			return fmt.Errorf("failed to register provider: %w", err)
		}
	}

	// Run coder flow
	return coder.RunFlow(ctx, r, out, provider, s.Model, s.CoderOp)
}

func readIndex(r *bufio.Reader, maxIdx int) (int, error) {
	for attempts := 0; attempts < 3; attempts++ {
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}

		line = strings.TrimSpace(line)
		idx, err := strconv.Atoi(line)
		if err == nil && idx >= 1 && idx <= maxIdx {
			return idx - 1, nil
		}

		fmt.Printf("Invalid input. Please enter a number between 1 and %d: ", maxIdx)
	}
	return 0, fmt.Errorf("too many invalid attempts")
}

func getDefaultModels(provider string) []core.Model {
	switch provider {
	case "openai":
		return []core.Model{
			{ID: "gpt-4-turbo-preview", Description: "Most capable, latest GPT-4"},
			{ID: "gpt-4", Description: "High intelligence model"},
			{ID: "gpt-3.5-turbo", Description: "Fast and efficient"},
		}
	case "anthropic":
		return []core.Model{
			{ID: "claude-3-opus-20240229", Description: "Most capable"},
			{ID: "claude-3-sonnet-20240229", Description: "Balanced performance"},
			{ID: "claude-3-haiku-20240307", Description: "Fast and efficient"},
		}
	case "deepseek":
		return []core.Model{
			{ID: "deepseek-chat", Description: "General purpose chat"},
			{ID: "deepseek-coder", Description: "Optimized for code"},
		}
	case "gemini":
		return []core.Model{
			{ID: "gemini-pro", Description: "Google's advanced model"},
			{ID: "gemini-pro-vision", Description: "Multimodal capabilities"},
		}
	default:
		return []core.Model{
			{ID: "default", Description: "Provider default"},
		}
	}
}
