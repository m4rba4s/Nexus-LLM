package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	branding "github.com/m4rba4s/Nexus-LLM/internal/branding"
	"github.com/m4rba4s/Nexus-LLM/internal/config"
	"github.com/m4rba4s/Nexus-LLM/internal/core"
	"github.com/m4rba4s/Nexus-LLM/internal/modes/coder"

	// Import providers to register their factories via init()
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/anthropic"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/deepseek"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/gemini"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/mock"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/openai"
	_ "github.com/m4rba4s/Nexus-LLM/internal/providers/openrouter"
)

type CoderFlags struct {
	Provider       string
	Model          string
	Operator       string
	Prompt         string
	Files          []string
	Output         string
	NoStream       bool
	NonInteractive bool
	Diff           bool
	Apply          bool
	BackupExt      string
	ApplyTo        string
	Yes            bool
	Patch          bool
}

// NewCoderCommand exposes the Coder workflow as a CLI command.
func NewCoderCommand() *cobra.Command {
	flags := &CoderFlags{}

	cmd := &cobra.Command{
		Use:   "coder",
		Short: "Coder workflow (codegen, refactor, test, docs)",
		Long: `Run the Coder workflow without the full menu.

Examples:
  gollm coder --operator codegen
  gollm coder -p openai -m gpt-4 --operator refactor
  gollm coder -p anthropic --operator test
  gollm coder --operator docs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// Small startup logo for better UX
			branding.StartupLogo()

			// Load config (tolerant to partial/invalid entries)
			cfg, err := config.LoadWithOptions(config.LoadOptions{SkipValidation: true})
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Resolve provider name
			providerName := strings.TrimSpace(flags.Provider)
			if providerName == "" {
				providerName = cfg.DefaultProvider
			}

			// Fallback to openrouter if not configured or invalid
			if providerName == "" || cfg.Providers == nil {
				providerName = "openrouter"
				if cfg.Providers == nil {
					cfg.Providers = make(map[string]config.ProviderConfig)
				}
				if _, ok := cfg.Providers[providerName]; !ok {
					cfg.Providers[providerName] = config.ProviderConfig{Type: "openrouter", DefaultModel: "k2"}
				}
				if cfg.DefaultProvider == "" {
					cfg.DefaultProvider = providerName
				}
			}

			// Resolve provider config
			pconf, exists := cfg.Providers[providerName]
			if !exists || pconf.Type == "" {
				// Synthesize sane default for openrouter
				providerName = "openrouter"
				pconf = config.ProviderConfig{Type: "openrouter", DefaultModel: "k2"}
				cfg.Providers[providerName] = pconf
				if cfg.DefaultProvider == "" {
					cfg.DefaultProvider = providerName
				}
			}

			// Create provider instance from factory
			reg := core.GetGlobalRegistry()
			prov, err := reg.CreateProvider(strings.ToLower(pconf.Type), pconf.ToProviderConfig())
			if err != nil {
				// Graceful fallback to mock provider if API key missing or provider fails to init
				fmt.Fprintln(os.Stderr, "[WARN] ", err)
				fmt.Fprintln(os.Stderr, "[INFO] Falling back to mock provider. Set OPENROUTER_API_KEY or GOLLM_PROVIDERS_OPENROUTER_API_KEY to use real Kimi K2.")
				prov, err = reg.CreateProvider("mock", core.ProviderConfig{Type: "mock"})
				if err != nil {
					return fmt.Errorf("failed to create fallback mock provider: %w", err)
				}
				// If user asked for openrouter explicitly, keep model as 'k2' for context; mock ignores it.
			}

			// Resolve model
			model := strings.TrimSpace(flags.Model)
			if model == "" {
				model = pconf.DefaultModel
			}
			if model == "" {
				model = "k2"
			}

			// Map operator
			var op coder.Operator
			switch strings.ToLower(flags.Operator) {
			case "", "codegen", "gen", "generate":
				op = coder.OperatorCodegen
			case "refactor", "ref":
				op = coder.OperatorRefactor
			case "test", "tests":
				op = coder.OperatorTest
			case "docs", "doc", "documentation":
				op = coder.OperatorDocs
			default:
				return fmt.Errorf("unknown operator: %s (valid: codegen|refactor|test|docs)", flags.Operator)
			}

			// Headless mode if prompt/files provided or explicitly requested
			if flags.NonInteractive || flags.Prompt != "" || len(flags.Files) > 0 {
				return runCoderHeadless(ctx, prov, model, op, flags)
			}

			// Interactive workflow
			reader := bufio.NewReader(os.Stdin)
			return coder.RunFlow(ctx, reader, os.Stdout, prov, model, op)
		},
	}

	addCoderFlags(cmd, flags)
	return cmd
}

func addCoderFlags(cmd *cobra.Command, flags *CoderFlags) {
	f := cmd.Flags()
	f.StringVarP(&flags.Provider, "provider", "p", "", "LLM provider to use (defaults to config default_provider)")
	f.StringVarP(&flags.Model, "model", "m", "", "model to use (defaults to provider default_model)")
	// Avoid -o shorthand conflict with global --output (-o)
	f.StringVar(&flags.Operator, "operator", "codegen", "coder operator: codegen|refactor|test|docs")
	f.StringVarP(&flags.Prompt, "prompt", "r", "", "prompt text (non-interactive mode)")
	f.StringSliceVarP(&flags.Files, "files", "f", nil, "comma-separated files to include as context")
	f.StringVarP(&flags.Output, "output-file", "O", "", "write result to file instead of stdout")
	f.BoolVar(&flags.NoStream, "no-stream", false, "disable streaming output")
	f.BoolVar(&flags.NonInteractive, "non-interactive", false, "run without interactive prompts")
	f.BoolVar(&flags.Diff, "diff", false, "print unified diff against the first --files entry")
	f.BoolVar(&flags.Apply, "apply", false, "apply generated content to the first --files entry (with backup)")
	f.StringVar(&flags.BackupExt, "backup-ext", ".backup", "backup extension when using --apply")
	f.StringVar(&flags.ApplyTo, "apply-to", "", "target file to apply result (overrides first --files entry)")
	f.BoolVar(&flags.Yes, "yes", false, "assume yes for confirmations (use with --apply)")
	f.BoolVar(&flags.Patch, "patch", false, "emit patch format (diff-like) to stdout or --output-file")
}

// runCoderHeadless executes a coder request without interactive prompts.
func runCoderHeadless(ctx context.Context, prov core.Provider, model string, op coder.Operator, fl *CoderFlags) error {
	// Build system prompt with language context
	lang := detectLanguage(fl.Files)
	system := buildSystemPrompt(op, lang)

	// Aggregate file contexts
	var sb strings.Builder
	for _, path := range fl.Files {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		// Limit extremely large files to first 500 lines
		trimmed := trimToLines(string(content), 500)
		sb.WriteString("\n--- File: ")
		sb.WriteString(path)
		sb.WriteString(" ---\n")
		sb.WriteString(trimmed)
		sb.WriteString("\n")
	}

	// Compose user prompt with minimal operator guidance
	var user strings.Builder
	if fl.Prompt != "" {
		user.WriteString(fl.Prompt)
		user.WriteString("\n\n")
	}
	if sb.Len() > 0 {
		user.WriteString("Context:\n")
		user.WriteString(sb.String())
	}
	switch strings.ToLower(string(op)) {
	case "refactor":
		user.WriteString("\nRefactor the code per best practices; preserve behavior.\n")
	case "test":
		user.WriteString("\nWrite table-driven Go tests with edge cases.\n")
	case "docs":
		user.WriteString("\nProduce clear documentation (Markdown/godoc) as requested.\n")
	default:
		user.WriteString("\nGenerate clean, idiomatic Go code with error handling.\n")
	}

	temp := 0.7
	maxTok := 0 // provider default

	req := &core.CompletionRequest{
		Model: model,
		Messages: []core.Message{
			{Role: core.RoleSystem, Content: system},
			{Role: core.RoleUser, Content: user.String()},
		},
		Temperature: &temp,
		MaxTokens:   &maxTok,
		Stream:      !fl.NoStream,
	}

	// Execute
	if fl.NoStream {
		resp, err := prov.CreateCompletion(ctx, req)
		if err != nil {
			return err
		}
		var b strings.Builder
		for _, c := range resp.Choices {
			if c.Message.Content != "" {
				b.WriteString(c.Message.Content)
			}
		}
		result := extractCodeFromResponse(b.String())
		return postProcessResult(result, fl)
	}

	stream, err := prov.StreamCompletion(ctx, req)
	if err != nil {
		return err
	}
	var out strings.Builder
	for ch := range stream {
		if ch.Error != nil {
			return ch.Error
		}
		for _, c := range ch.Choices {
			var delta string
			if c.Delta != nil {
				delta = c.Delta.Content
			} else {
				delta = c.Message.Content
			}
			out.WriteString(delta)
			if fl.Output == "" {
				fmt.Fprint(os.Stdout, delta)
			}
		}
	}
	return postProcessResult(out.String(), fl)
}

func writeOutput(path, content string) error {
	if path == "" {
		fmt.Fprintln(os.Stdout)
		return nil
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	fmt.Fprintf(os.Stdout, "\n[SAVED] %s\n", path)
	return nil
}

func trimToLines(s string, max int) string {
	if max <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "\n") + "\n... (truncated)"
}

// extractCodeFromResponse pulls markdown code blocks if present, otherwise full text.
func extractCodeFromResponse(resp string) string {
	var blocks []string
	lines := strings.Split(resp, "\n")
	in := false
	var cur strings.Builder
	for _, ln := range lines {
		if strings.HasPrefix(ln, "```") {
			if in {
				blocks = append(blocks, cur.String())
				cur.Reset()
				in = false
			} else {
				in = true
			}
			continue
		}
		if in {
			cur.WriteString(ln)
			cur.WriteByte('\n')
		}
	}
	if len(blocks) == 0 {
		return resp
	}
	return strings.Join(blocks, "\n\n")
}

// postProcessResult optionally prints diff and applies changes to first file.
func postProcessResult(result string, fl *CoderFlags) error {
	// If no file target and only stdout desired
	if !fl.Diff && !fl.Apply && fl.Output == "" {
		// Just ensure newline for clean output
		fmt.Fprintln(os.Stdout)
		return nil
	}

	// If diff/apply requested, need at least one file
	if (fl.Diff || fl.Apply || fl.Patch) && len(fl.Files) == 0 && fl.ApplyTo == "" {
		if fl.Output != "" {
			return writeOutput(fl.Output, result)
		}
		fmt.Fprintln(os.Stdout)
		return nil
	}

	// Target file
	target := strings.TrimSpace(fl.ApplyTo)
	if target == "" && len(fl.Files) > 0 {
		target = strings.TrimSpace(fl.Files[0])
	}
	if target == "" {
		if fl.Output != "" {
			return writeOutput(fl.Output, result)
		}
		fmt.Fprintln(os.Stdout)
		return nil
	}
	oldBytes, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read target file %s: %w", target, err)
	}
	old := string(oldBytes)
	newc := extractCodeFromResponse(result)

	if fl.Diff || fl.Patch {
		diff := unifiedDiff(target, old, newc)
		if fl.Patch {
			patch := generatePatch(target, diff)
			if fl.Output != "" {
				return writeOutput(fl.Output, patch)
			}
			fmt.Fprint(os.Stdout, patch)
		} else {
			fmt.Fprint(os.Stdout, diff)
		}
	}
	if fl.Apply {
		if !fl.Yes {
			// Confirm destructive change
			fmt.Fprintf(os.Stdout, "\nThis will overwrite %s. Proceed? (yes/no): ", target)
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans != "yes" && ans != "y" {
				fmt.Fprintln(os.Stdout, "[CANCELLED] Apply aborted")
				return nil
			}
		}
		// Backup
		backup := target + fl.BackupExt
		if err := os.WriteFile(backup, oldBytes, 0644); err != nil {
			return fmt.Errorf("failed to write backup: %w", err)
		}
		if err := os.WriteFile(target, []byte(newc), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Fprintf(os.Stdout, "\n[APPLIED] %s (backup: %s)\n", target, backup)
	}

	if fl.Output != "" && !fl.Patch {
		if err := writeOutput(fl.Output, newc); err != nil {
			return err
		}
	}
	return nil
}

// unifiedDiff creates a simple unified diff view of two texts.
func unifiedDiff(filename, old, newc string) string {
	const contextN = 3
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(newc, "\n")

	// find first and last differing regions
	start := 0
	for start < len(oldLines) && start < len(newLines) && oldLines[start] == newLines[start] {
		start++
	}
	endOld := len(oldLines) - 1
	endNew := len(newLines) - 1
	for endOld >= start && endNew >= start && oldLines[endOld] == newLines[endNew] {
		endOld--
		endNew--
	}

	// compute hunk ranges with context
	hStart := max(0, start-contextN)
	hEndOld := min(len(oldLines)-1, endOld+contextN)
	hEndNew := min(len(newLines)-1, endNew+contextN)

	var b strings.Builder
	b.WriteString("--- ")
	b.WriteString(filename)
	b.WriteByte('\n')
	b.WriteString("+++ ")
	b.WriteString(filename)
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hStart+1, (hEndOld-hStart)+1, hStart+1, (hEndNew-hStart)+1))

	i := hStart
	j := hStart
	for i <= hEndOld || j <= hEndNew {
		if i <= endOld && j <= endNew {
			if i <= hEndOld && j <= hEndNew && oldLines[i] == newLines[j] {
				b.WriteString(" ")
				b.WriteString(oldLines[i])
				b.WriteByte('\n')
				i++
				j++
				continue
			}
		}
		if i <= endOld && (j > endNew || (i <= hEndOld && (j > hEndNew || oldLines[i] != newLines[j]))) {
			b.WriteString("-")
			b.WriteString(oldLines[i])
			b.WriteByte('\n')
			i++
			continue
		}
		if j <= hEndNew {
			b.WriteString("+")
			b.WriteString(newLines[j])
			b.WriteByte('\n')
			j++
			continue
		}
		// context-only beyond changed region
		if i <= hEndOld {
			b.WriteString(" ")
			b.WriteString(oldLines[i])
			b.WriteByte('\n')
			i++
		} else if j <= hEndNew {
			b.WriteString(" ")
			b.WriteString(newLines[j])
			b.WriteByte('\n')
			j++
		} else {
			break
		}
	}
	return b.String()
}

// generatePatch wraps a unified diff into a basic patch format.
func generatePatch(filename, diff string) string {
	var b strings.Builder
	b.WriteString("diff --git a/")
	b.WriteString(filename)
	b.WriteString(" b/")
	b.WriteString(filename)
	b.WriteByte('\n')
	b.WriteString("index 0000000..0000000 100644\n")
	b.WriteString("--- a/")
	b.WriteString(filename)
	b.WriteByte('\n')
	b.WriteString("+++ b/")
	b.WriteString(filename)
	b.WriteByte('\n')
	// diff already contains @@ hunk and +/- lines
	// Remove possible leading ---/+++ to avoid duplication
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// detectLanguage infers primary language from file extensions.
func detectLanguage(files []string) string {
	if len(files) == 0 {
		return "Go"
	}
	// pick first non-empty
	var f string
	for _, p := range files {
		if strings.TrimSpace(p) != "" {
			f = p
			break
		}
	}
	if f == "" {
		return "Go"
	}
	ext := strings.ToLower(filepath.Ext(f))
	switch ext {
	case ".go":
		return "Go"
	case ".py":
		return "Python"
	case ".ts":
		return "TypeScript"
	case ".js":
		return "JavaScript"
	case ".java":
		return "Java"
	case ".rb":
		return "Ruby"
	case ".rs":
		return "Rust"
	case ".c":
		return "C"
	case ".h":
		return "C"
	case ".cpp", ".cc", ".hpp":
		return "C++"
	case ".cs":
		return "C#"
	case ".sh":
		return "Shell"
	case ".sql":
		return "SQL"
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".json":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".proto":
		return "Protocol Buffers"
	default:
		return "Go"
	}
}

// buildSystemPrompt crafts a focused system prompt including operator and language.
func buildSystemPrompt(op coder.Operator, lang string) string {
	base := "You are an expert software engineer. Produce high-quality, production-ready results.\n"
	langLine := "Target language: " + lang + "\n"
	switch op {
	case coder.OperatorCodegen:
		return base + langLine + "Focus on idiomatic code, error handling, performance, security, and testability. Output only the code unless asked otherwise."
	case coder.OperatorRefactor:
		return base + langLine + "Refactor for readability, maintainability, and testability. Preserve behavior. Output full refactored code or minimal diff as appropriate."
	case coder.OperatorTest:
		return base + langLine + "Write comprehensive, table-driven tests with clear names and edge cases. Output test code only."
	case coder.OperatorDocs:
		return base + langLine + "Write clear documentation with examples. Use appropriate format (Markdown/godoc)."
	default:
		return base + langLine
	}
}
