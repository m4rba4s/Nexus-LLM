// Package display provides beautiful terminal output with colors, progress bars, and formatting.
//
// This package handles all visual output for GOLLM, providing:
// - Colored terminal output with graceful degradation
// - Progress bars for streaming responses
// - Formatted display of models, configurations, and responses
// - Raw output mode for scripting integration
// - Interactive elements and prompts
//
// Example usage:
//
//	renderer := display.NewRenderer(display.Options{
//		Colors:      true,
//		Interactive: true,
//		Format:      display.FormatPretty,
//	})
//
//	renderer.Success("Configuration saved successfully")
//	renderer.StreamingResponse(response, func(chunk string) {
//		// Handle streaming chunk
//	})
package display

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Format represents the output format type.
type Format string

const (
	// FormatPretty provides colored, human-readable output
	FormatPretty Format = "pretty"
	// FormatJSON provides JSON output for scripting
	FormatJSON Format = "json"
	// FormatRaw provides raw text output
	FormatRaw Format = "raw"
	// FormatTable provides tabular output
	FormatTable Format = "table"
)

// Options configures the display renderer behavior.
type Options struct {
	// Colors enables colored output (auto-detected if not set)
	Colors bool
	// Interactive enables interactive elements like progress bars
	Interactive bool
	// Format specifies the output format
	Format Format
	// Output specifies the output writer (defaults to os.Stdout)
	Output io.Writer
	// ErrorOutput specifies the error output writer (defaults to os.Stderr)
	ErrorOutput io.Writer
	// Quiet suppresses non-essential output
	Quiet bool
	// Verbose enables verbose output
	Verbose bool
}

// Renderer provides methods for displaying various types of content.
type Renderer struct {
	opts     Options
	mu       sync.RWMutex
	progress *progressbar.ProgressBar

	// Color functions
	success  func(a ...interface{}) string
	error    func(a ...interface{}) string
	warning  func(a ...interface{}) string
	info     func(a ...interface{}) string
	debug    func(a ...interface{}) string
	emphasis func(a ...interface{}) string
	dim      func(a ...interface{}) string
	bold     func(a ...interface{}) string
}

// NewRenderer creates a new display renderer with the specified options.
func NewRenderer(opts Options) *Renderer {
	// Set defaults
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.ErrorOutput == nil {
		opts.ErrorOutput = os.Stderr
	}
	if opts.Format == "" {
		opts.Format = FormatPretty
	}

	// Auto-detect color support if not explicitly set
	if !opts.Colors {
		opts.Colors = isColorTerminal()
	}

	r := &Renderer{opts: opts}
	r.initColors()
	return r
}

// initColors initializes color functions based on options.
func (r *Renderer) initColors() {
	if r.opts.Colors && r.opts.Format == FormatPretty {
		r.success = color.New(color.FgGreen, color.Bold).SprintFunc()
		r.error = color.New(color.FgRed, color.Bold).SprintFunc()
		r.warning = color.New(color.FgYellow, color.Bold).SprintFunc()
		r.info = color.New(color.FgCyan).SprintFunc()
		r.debug = color.New(color.FgWhite, color.Faint).SprintFunc()
		r.emphasis = color.New(color.FgBlue, color.Bold).SprintFunc()
		r.dim = color.New(color.Faint).SprintFunc()
		r.bold = color.New(color.Bold).SprintFunc()
	} else {
		// No-color functions
		noop := func(a ...interface{}) string {
			return fmt.Sprint(a...)
		}
		r.success = noop
		r.error = noop
		r.warning = noop
		r.info = noop
		r.debug = noop
		r.emphasis = noop
		r.dim = noop
		r.bold = noop
	}
}

// Success displays a success message.
func (r *Renderer) Success(msg string) {
	if r.opts.Quiet {
		return
	}

	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(map[string]interface{}{
			"level":   "success",
			"message": msg,
			"time":    time.Now().Format(time.RFC3339),
		})
	case FormatRaw:
		fmt.Fprintln(r.opts.Output, msg)
	default:
		fmt.Fprintf(r.opts.Output, "%s %s\n", r.success("✓"), msg)
	}
}

// Error displays an error message.
func (r *Renderer) Error(msg string) {
	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(map[string]interface{}{
			"level":   "error",
			"message": msg,
			"time":    time.Now().Format(time.RFC3339),
		})
	case FormatRaw:
		fmt.Fprintln(r.opts.ErrorOutput, msg)
	default:
		fmt.Fprintf(r.opts.ErrorOutput, "%s %s\n", r.error("✗"), msg)
	}
}

// Warning displays a warning message.
func (r *Renderer) Warning(msg string) {
	if r.opts.Quiet {
		return
	}

	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(map[string]interface{}{
			"level":   "warning",
			"message": msg,
			"time":    time.Now().Format(time.RFC3339),
		})
	case FormatRaw:
		fmt.Fprintln(r.opts.Output, msg)
	default:
		fmt.Fprintf(r.opts.Output, "%s %s\n", r.warning("⚠"), msg)
	}
}

// Info displays an informational message.
func (r *Renderer) Info(msg string) {
	if r.opts.Quiet {
		return
	}

	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(map[string]interface{}{
			"level":   "info",
			"message": msg,
			"time":    time.Now().Format(time.RFC3339),
		})
	case FormatRaw:
		fmt.Fprintln(r.opts.Output, msg)
	default:
		fmt.Fprintf(r.opts.Output, "%s %s\n", r.info("ℹ"), msg)
	}
}

// Debug displays a debug message (only if verbose mode is enabled).
func (r *Renderer) Debug(msg string) {
	if !r.opts.Verbose || r.opts.Quiet {
		return
	}

	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(map[string]interface{}{
			"level":   "debug",
			"message": msg,
			"time":    time.Now().Format(time.RFC3339),
		})
	case FormatRaw:
		fmt.Fprintln(r.opts.Output, msg)
	default:
		fmt.Fprintf(r.opts.Output, "%s %s\n", r.debug("🔧"), msg)
	}
}

// StartProgress starts a progress bar with the given description.
func (r *Renderer) StartProgress(description string) {
	if !r.opts.Interactive || r.opts.Quiet || r.opts.Format != FormatPretty {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.progress = progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(r.opts.ErrorOutput),
		progressbar.OptionEnableColorCodes(r.opts.Colors),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}

// UpdateProgress updates the current progress.
func (r *Renderer) UpdateProgress() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.progress != nil {
		r.progress.Add(1)
	}
}

// FinishProgress finishes the current progress bar.
func (r *Renderer) FinishProgress() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.progress != nil {
		r.progress.Finish()
		r.progress = nil
	}
}

// StreamingResponse handles streaming response display with real-time updates.
func (r *Renderer) StreamingResponse(onChunk func() (string, bool)) {
	if r.opts.Format == FormatJSON {
		r.streamingJSON(onChunk)
		return
	}

	r.StartProgress("Receiving response...")

	var response strings.Builder
	for {
		chunk, more := onChunk()
		if !more {
			break
		}

		response.WriteString(chunk)

		// Print chunk immediately for streaming experience
		if r.opts.Format == FormatPretty {
			fmt.Fprint(r.opts.Output, chunk)
		}

		r.UpdateProgress()
	}

	r.FinishProgress()

	// For raw format, print the complete response
	if r.opts.Format == FormatRaw {
		fmt.Fprint(r.opts.Output, response.String())
	}

	// Add newline for pretty format
	if r.opts.Format == FormatPretty {
		fmt.Fprintln(r.opts.Output)
	}
}

// ModelsList displays a formatted list of available models.
func (r *Renderer) ModelsList(models []ModelInfo) {
	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(map[string]interface{}{
			"models": models,
			"count":  len(models),
		})
	case FormatRaw:
		for _, model := range models {
			fmt.Fprintln(r.opts.Output, model.Name)
		}
	case FormatTable:
		r.printModelsTable(models)
	default:
		r.printModelsPretty(models)
	}
}

// ConfigInfo displays configuration information.
func (r *Renderer) ConfigInfo(config interface{}) {
	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(config)
	case FormatRaw:
		// For raw format, print key-value pairs
		r.printConfigRaw(config)
	default:
		r.printConfigPretty(config)
	}
}

// BenchmarkResult displays benchmark results.
func (r *Renderer) BenchmarkResult(result BenchmarkResult) {
	switch r.opts.Format {
	case FormatJSON:
		r.printJSON(result)
	case FormatRaw:
		fmt.Fprintf(r.opts.Output, "Provider: %s\nModel: %s\nLatency: %v\nTokens/sec: %.2f\nSuccess Rate: %.2f%%\n",
			result.Provider, result.Model, result.AvgLatency, result.TokensPerSecond, result.SuccessRate*100)
	default:
		r.printBenchmarkPretty(result)
	}
}

// Helper types for display data
type ModelInfo struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Description string `json:"description,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
}

type BenchmarkResult struct {
	Provider        string        `json:"provider"`
	Model          string        `json:"model"`
	AvgLatency     time.Duration `json:"avg_latency"`
	TokensPerSecond float64       `json:"tokens_per_second"`
	SuccessRate    float64       `json:"success_rate"`
	TotalRequests  int           `json:"total_requests"`
}

// Helper methods

func (r *Renderer) printJSON(data interface{}) {
	encoder := json.NewEncoder(r.opts.Output)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}

func (r *Renderer) streamingJSON(onChunk func() (string, bool)) {
	fmt.Fprint(r.opts.Output, `{"content":"`)

	for {
		chunk, more := onChunk()
		if !more {
			break
		}

		// Escape JSON special characters
		escaped := strings.ReplaceAll(chunk, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		fmt.Fprint(r.opts.Output, escaped)
	}

	fmt.Fprintln(r.opts.Output, `"}`)
}

func (r *Renderer) printModelsTable(models []ModelInfo) {
	if len(models) == 0 {
		r.Info("No models available")
		return
	}

	// Print header
	fmt.Fprintf(r.opts.Output, "%-30s %-15s %s\n", "NAME", "PROVIDER", "DESCRIPTION")
	fmt.Fprintf(r.opts.Output, "%-30s %-15s %s\n",
		strings.Repeat("-", 30),
		strings.Repeat("-", 15),
		strings.Repeat("-", 40))

	// Print models
	for _, model := range models {
		desc := model.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fmt.Fprintf(r.opts.Output, "%-30s %-15s %s\n", model.Name, model.Provider, desc)
	}

	fmt.Fprintf(r.opts.Output, "\nTotal models: %d\n", len(models))
}

func (r *Renderer) printModelsPretty(models []ModelInfo) {
	if len(models) == 0 {
		r.Info("No models available")
		return
	}

	fmt.Fprintf(r.opts.Output, "\n%s Available Models (%d total):\n\n", r.emphasis("📋"), len(models))

	currentProvider := ""
	for _, model := range models {
		if model.Provider != currentProvider {
			currentProvider = model.Provider
			fmt.Fprintf(r.opts.Output, "%s %s\n", r.bold("▶"), r.info(currentProvider))
		}

		fmt.Fprintf(r.opts.Output, "  %s %s", r.dim("•"), model.Name)
		if model.Description != "" {
			fmt.Fprintf(r.opts.Output, " - %s", r.dim(model.Description))
		}
		fmt.Fprintln(r.opts.Output)
	}

	fmt.Fprintln(r.opts.Output)
}

func (r *Renderer) printConfigPretty(config interface{}) {
	fmt.Fprintf(r.opts.Output, "\n%s Current Configuration:\n\n", r.emphasis("⚙️"))

	// Convert to JSON for pretty printing
	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		r.Error("Failed to format configuration")
		return
	}

	// Add syntax highlighting for JSON (basic)
	lines := strings.Split(string(jsonBytes), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && strings.Contains(line, `"`) {
			// Highlight keys
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				fmt.Fprintf(r.opts.Output, "%s:%s\n", r.bold(parts[0]), parts[1])
				continue
			}
		}
		fmt.Fprintln(r.opts.Output, line)
	}

	fmt.Fprintln(r.opts.Output)
}

func (r *Renderer) printConfigRaw(config interface{}) {
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		fmt.Fprintf(r.opts.ErrorOutput, "Error: %v\n", err)
		return
	}
	fmt.Fprintln(r.opts.Output, string(jsonBytes))
}

func (r *Renderer) printBenchmarkPretty(result BenchmarkResult) {
	fmt.Fprintf(r.opts.Output, "\n%s Benchmark Results:\n\n", r.emphasis("⚡"))

	fmt.Fprintf(r.opts.Output, "%s %s\n", r.bold("Provider:"), result.Provider)
	fmt.Fprintf(r.opts.Output, "%s %s\n", r.bold("Model:"), result.Model)
	fmt.Fprintf(r.opts.Output, "%s %v\n", r.bold("Average Latency:"), result.AvgLatency)
	fmt.Fprintf(r.opts.Output, "%s %.2f tokens/sec\n", r.bold("Throughput:"), result.TokensPerSecond)

	successColor := r.success
	if result.SuccessRate < 0.95 {
		successColor = r.warning
	}
	if result.SuccessRate < 0.8 {
		successColor = r.error
	}

	fmt.Fprintf(r.opts.Output, "%s %s\n", r.bold("Success Rate:"), successColor(fmt.Sprintf("%.2f%%", result.SuccessRate*100)))
	fmt.Fprintf(r.opts.Output, "%s %d\n", r.bold("Total Requests:"), result.TotalRequests)

	fmt.Fprintln(r.opts.Output)
}

// isColorTerminal detects if the current terminal supports colors.
func isColorTerminal() bool {
	// Check common environment variables
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for FORCE_COLOR environment variable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Basic terminal type detection
	colorTerms := []string{"xterm", "xterm-256color", "screen", "tmux"}
	for _, colorTerm := range colorTerms {
		if strings.Contains(term, colorTerm) {
			return true
		}
	}

	return false
}
