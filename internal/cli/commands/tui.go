// Package commands provides CLI command implementations for GOLLM.
package commands

import (
    "fmt"
    "bufio"
    "os"
    "strconv"
    "strings"
    "context"
    "time"

    "github.com/spf13/cobra"

    "github.com/yourusername/gollm/internal/tui"
    "github.com/yourusername/gollm/internal/config"
    "github.com/yourusername/gollm/internal/core"
)

// TUIFlags contains flags specific to the TUI command.
type TUIFlags struct {
    Provider    string
    Model       string
    Profile     string
    Theme       string
    NoMatrix    bool
    Debug       bool
    Performance bool
    Simple      bool
    Select      bool
    System      string
    FreeOnly    bool
}

// NewTUICommand creates the cyberpunk TUI command.
func NewTUICommand() *cobra.Command {
	flags := &TUIFlags{}

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the cyberpunk-style TUI interface",
		Long: `Launch GOLLM's high-tech terminal user interface with cyberpunk aesthetics.

The TUI provides an immersive chat experience with:
• Real-time streaming responses with visual effects
• Matrix-style background animations
• Cyberpunk color scheme and glitch effects
• Interactive chat history and session management
• Live token counting and performance metrics
• Keyboard shortcuts for power users
• Multi-line input support

Features:
  🎮 Immersive cyberpunk interface with animations
  ⚡ Real-time streaming with visual feedback
  🎨 Customizable themes and color schemes
  📊 Live performance metrics and token usage
  🔥 Matrix rain background effects
  ⌨️  Power user keyboard shortcuts
  💾 Session persistence and chat history
  🌟 Glitch effects and sci-fi aesthetics

Keyboard Controls:
  • Enter:     Send message
  • Ctrl+C:    Exit application
  • Ctrl+L:    Clear chat history
  • F1:        Toggle statistics display
  • F2:        Toggle debug mode
  • F3:        Toggle multiline input
  • ESC:       Exit application

Examples:
  # Launch with default settings
  gollm tui

  # Launch with specific provider and model
  gollm tui --provider openai --model gpt-4

  # Launch with custom theme
  gollm tui --theme neon

  # Launch with performance monitoring
  gollm tui --performance --debug

  # Launch without matrix effects for better performance
  gollm tui --no-matrix

  # Launch simple terminal chat (no TUI framework)
  gollm tui --simple`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUICommand(cmd, args, flags)
		},
	}

	// Add command-specific flags
	addTUIFlags(cmd, flags)

	return cmd
}

// addTUIFlags adds flags specific to the TUI command.
func addTUIFlags(cmd *cobra.Command, flags *TUIFlags) {
	f := cmd.Flags()

	// Provider and model selection
	f.StringVar(&flags.Provider, "provider", "",
		"LLM provider to use (overrides config default)")
	f.StringVar(&flags.Model, "model", "",
		"model to use (overrides config default)")
	f.StringVar(&flags.Profile, "profile", "",
		"configuration profile to use")

	// Visual and theme options
	f.StringVar(&flags.Theme, "theme", "cyberpunk",
		"UI theme to use (cyberpunk, neon, matrix, minimal)")
	f.BoolVar(&flags.NoMatrix, "no-matrix", false,
		"disable matrix rain effects for better performance")

	// Debug and performance options
	f.BoolVar(&flags.Debug, "debug", false,
		"enable debug mode with detailed logging")
    f.BoolVar(&flags.Performance, "performance", false,
        "enable performance monitoring and metrics")
    f.BoolVar(&flags.Simple, "simple", false,
        "use simple terminal chat without TUI framework")
    f.BoolVar(&flags.Select, "select", false,
        "prompt to select provider/model before launching UI")
    f.BoolVar(&flags.FreeOnly, "free-only", false,
        "when selecting models, prefer/free-only models where available (OpenRouter)")
    f.StringVar(&flags.System, "system", "",
        "initial system instruction for the session")

	// Add completion for provider flag
	cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	cmd.RegisterFlagCompletionFunc("model", completeModels)
	cmd.RegisterFlagCompletionFunc("theme", completeThemes)
}

// runTUICommand executes the TUI command.
func runTUICommand(cmd *cobra.Command, args []string, flags *TUIFlags) error {
	// Load configuration
	cfg, err := getInjectedOrLoad()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply provider override if specified
	if flags.Provider != "" {
		if !cfg.HasProvider(flags.Provider) {
			available := cfg.ListProviders()
			return fmt.Errorf("provider %q not configured; available providers: %s",
				flags.Provider, joinStrings(available, ", "))
		}
	}

    // Validate profile if specified
    if flags.Profile != "" {
		// Profile validation would go here
		// For now, we'll just pass it to the TUI
	}

    // Interactive selection before launch if requested or not specified
    if flags.Select || flags.Provider == "" || flags.Model == "" {
        if err := selectProviderModel(cmd, cfg, flags); err != nil {
            return err
        }
    }

    // First-run friendliness: if no providers configured, fallback to Simple mode
    if len(cfg.Providers) == 0 {
		fmt.Println("ℹ No providers configured. Launching Simple mode with mock responses.")
		fmt.Println("   Tip: Run 'gollm config init' then 'gollm config set providers.openai.api_key sk-...'")
		fmt.Println()
		flags.Simple = true
	}

	// Initialize and run the cyberpunk TUI
	fmt.Println("🚀 Initializing GOLLM Cyberpunk Interface...")
	fmt.Println("▶ Loading neural pathways...")
	fmt.Println("▶ Establishing quantum entanglement...")
	fmt.Println("▶ Calibrating matrix projections...")
	fmt.Println("✨ Ready! Welcome to the future.")
	fmt.Println()

	// Create TUI configuration
    tuiConfig := &tui.Config{
        Provider:    flags.Provider,
        Model:       flags.Model,
        Profile:     flags.Profile,
        Theme:       flags.Theme,
        NoMatrix:    flags.NoMatrix,
        Debug:       flags.Debug,
        Performance: flags.Performance,
        SystemMessage: flags.System,
    }

	// Launch the appropriate TUI based on flags
	if flags.Simple {
		fmt.Println("🔧 Simple mode: Launching terminal chat...")
		return tui.RunSimpleChat(cfg, tuiConfig)
	} else if flags.Debug {
		fmt.Println("🔧 Debug mode: Launching minimal TUI...")
		return tui.RunMinimalTUI(cfg, tuiConfig)
	} else if flags.Theme == "professional" || flags.Theme == "pro" {
		fmt.Println("🎯 Launching GOLLM Professional Interface...")
		return tui.RunProfessionalTUI(cfg, tuiConfig)
	} else {
		return tui.RunCyberpunkTUI(cfg, tuiConfig)
	}
}

// selectProviderModel interactively selects provider, model and system message
func selectProviderModel(cmd *cobra.Command, cfg *config.Config, flags *TUIFlags) error {
    // If provider is already set and valid, we can skip provider selection
    providerName := flags.Provider
    if providerName == "" {
        names := cfg.ListProviders()
        if len(names) == 0 {
            return fmt.Errorf("no providers configured; run 'gollm config init' and set API keys")
        }
        fmt.Fprintln(cmd.OutOrStdout(), "Select provider:")
        for i, name := range names {
            fmt.Fprintf(cmd.OutOrStdout(), "  %d) %s\n", i+1, name)
        }
        fmt.Fprint(cmd.OutOrStdout(), "Enter number: ")
        idx, err := readNumberInRange(os.Stdin, 1, len(names))
        if err != nil {
            return err
        }
        providerName = names[idx-1]
        flags.Provider = providerName
    } else if !cfg.HasProvider(providerName) {
        return fmt.Errorf("provider %q not configured", providerName)
    }

    // Resolve models list
    pc, _, err := cfg.GetProvider(providerName)
    if err != nil {
        return fmt.Errorf("failed to get provider config: %w", err)
    }
    models := append([]string{}, pc.Models...)

    // If no models in config, try provider's GetModels
    if len(models) == 0 {
        // Build a temporary provider to list models
        prov, err := core.CreateProviderFromConfig(pc.Type, pc.ToProviderConfig())
        if err == nil {
            if lister, ok := prov.(core.ModelLister); ok {
                timeout := cfg.Settings.Timeout
                if timeout <= 0 { timeout = 15 * time.Second }
                ctx, cancel := context.WithTimeout(context.Background(), timeout)
                defer cancel()
                if list, e := lister.GetModels(ctx); e == nil {
                    models = make([]string, 0, len(list))
                    for _, m := range list {
                        // Optionally filter free-only for openrouter
                        if flags.FreeOnly && providerName == "openrouter" {
                            if (m.InputCostPer1K != nil && *m.InputCostPer1K > 0) || (m.OutputCostPer1K != nil && *m.OutputCostPer1K > 0) {
                                continue
                            }
                        }
                        models = append(models, m.ID)
                    }
                }
            }
        }
    }

    // Nudge kimi/k2 to top if openrouter
    if providerName == "openrouter" && len(models) > 0 {
        prioritized := make([]string, 0, len(models))
        others := make([]string, 0, len(models))
        for _, m := range models {
            lm := strings.ToLower(m)
            if strings.Contains(lm, "kimi") || strings.Contains(lm, "/k2") || strings.Contains(lm, "-k2") {
                prioritized = append(prioritized, m)
            } else {
                others = append(others, m)
            }
        }
        models = append(prioritized, others...)
    }

    // If still empty, use default model if present
    if len(models) == 0 && pc.DefaultModel != "" {
        models = []string{pc.DefaultModel}
    }

    // Determine model
    if flags.Model == "" {
        if len(models) == 0 {
            // Ask user to type a model ID
            fmt.Fprintln(cmd.OutOrStdout(), "Enter model ID (no list available):")
            fmt.Fprint(cmd.OutOrStdout(), "Model> ")
            m, err := readLine(os.Stdin)
            if err != nil {
                return err
            }
            flags.Model = strings.TrimSpace(m)
        } else {
            fmt.Fprintf(cmd.OutOrStdout(), "Select model for %s:\n", providerName)
            limit := 50
            if len(models) < limit { limit = len(models) }
            for i := 0; i < limit; i++ {
                fmt.Fprintf(cmd.OutOrStdout(), "  %d) %s\n", i+1, models[i])
            }
            if len(models) > limit {
                fmt.Fprintf(cmd.OutOrStdout(), "  ... and %d more. Type number or enter model id.\n", len(models)-limit)
            }
            fmt.Fprint(cmd.OutOrStdout(), "Enter number or model id: ")
            text, err := readLine(os.Stdin)
            if err != nil {
                return err
            }
            text = strings.TrimSpace(text)
            if n, convErr := strconv.Atoi(text); convErr == nil && n >= 1 && n <= len(models) {
                flags.Model = models[n-1]
            } else if text != "" {
                flags.Model = text
            } else {
                // fallback to default
                if pc.DefaultModel != "" {
                    flags.Model = pc.DefaultModel
                } else {
                    flags.Model = models[0]
                }
            }
        }
    }

    // Ask for system message if not set
    if flags.System == "" {
        fmt.Fprintln(cmd.OutOrStdout(), "Optional: enter system instructions (empty to skip):")
        fmt.Fprint(cmd.OutOrStdout(), "System> ")
        s, err := readLine(os.Stdin)
        if err == nil {
            flags.System = strings.TrimSpace(s)
        }
    }

    return nil
}

func readNumberInRange(r *os.File, min, max int) (int, error) {
    line, err := readLine(r)
    if err != nil { return 0, err }
    n, err := strconv.Atoi(strings.TrimSpace(line))
    if err != nil || n < min || n > max {
        return 0, fmt.Errorf("invalid selection")
    }
    return n, nil
}

func readLine(r *os.File) (string, error) {
    br := bufio.NewReader(r)
    s, err := br.ReadString('\n')
    if err != nil && err.Error() != "EOF" {
        return s, err
    }
    return strings.TrimRight(s, "\r\n"), nil
}

// completeThemes provides shell completion for theme names.
func completeThemes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	themes := []string{
		"cyberpunk",
		"neon",
		"matrix",
		"minimal",
		"retro",
		"dark",
		"light",
		"professional",
		"pro",
	}
	return themes, cobra.ShellCompDirectiveNoFileComp
}

// joinStrings joins a slice of strings with a separator (helper function).
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
