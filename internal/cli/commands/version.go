// Package commands provides the version command for displaying build and system information.
//
// The version command displays comprehensive information about the GOLLM build,
// runtime environment, and system configuration. This information is useful
// for debugging, support, and understanding the current installation.
//
// The command provides different output formats:
// - Pretty format with colored, human-readable output
// - JSON format for programmatic consumption
// - Raw format for simple version string
//
// Usage:
//
//	gollm version
//	gollm version --detailed
//	gollm version --output json
//	gollm version --check-updates
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fatih/color"
	"github.com/yourusername/gollm/internal/display"
)

// VersionFlags holds all version command configuration.
type VersionFlags struct {
	OutputFormat  string
	Detailed      bool
	CheckUpdates  bool
	ShowBuildTags bool
	ShowDeps      bool
	Quiet         bool
	Short         bool
}

// VersionInfo contains comprehensive version and build information.
type VersionInfo struct {
	// Basic version information
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`

	// Build information
	BuildOS         string   `json:"build_os"`
	BuildArch       string   `json:"build_arch"`
	BuildTags       []string `json:"build_tags,omitempty"`
	BuildUser       string   `json:"build_user,omitempty"`
	BuildHost       string   `json:"build_host,omitempty"`
	CompilerVersion string   `json:"compiler_version"`

	// Runtime information
	RuntimeOS     string `json:"runtime_os"`
	RuntimeArch   string `json:"runtime_arch"`
	NumCPU        int    `json:"num_cpu"`
	NumGoroutines int    `json:"num_goroutines"`
	MemAllocated  uint64 `json:"mem_allocated_bytes"`
	MemSys        uint64 `json:"mem_sys_bytes"`
	MemHeapInUse  uint64 `json:"mem_heap_inuse_bytes"`
	NextGC        uint64 `json:"next_gc_bytes"`
	LastGC        string `json:"last_gc_time"`

	// Dependencies
	Dependencies []DependencyInfo `json:"dependencies,omitempty"`

	// Feature flags and capabilities
	Features []FeatureInfo `json:"features,omitempty"`

	// Performance metrics
	StartupTime    string `json:"startup_time,omitempty"`
	ConfigLoadTime string `json:"config_load_time,omitempty"`
}

// DependencyInfo contains information about a dependency.
type DependencyInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Sum     string `json:"sum,omitempty"`
	Replace string `json:"replace,omitempty"`
}

// FeatureInfo contains information about available features.
type FeatureInfo struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// UpdateInfo contains information about available updates.
type UpdateInfo struct {
	Available     bool   `json:"available"`
	LatestVersion string `json:"latest_version,omitempty"`
	ReleaseURL    string `json:"release_url,omitempty"`
	ReleaseNotes  string `json:"release_notes,omitempty"`
	PublishedAt   string `json:"published_at,omitempty"`
}

var (
	// Build information, set by ldflags during compilation
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
	BuildUser = "unknown"
	BuildHost = "unknown"
	BuildTags = ""

	// Startup metrics
	startupTime    time.Time
	configLoadTime time.Duration
)

func init() {
	startupTime = time.Now()
}

// NewVersionCommand creates the version command.
func NewVersionCommand() *cobra.Command {
	flags := &VersionFlags{}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version and build information",
		Long: `Display comprehensive version and build information for GOLLM.

Shows version, build details, runtime information, and system configuration.
Use different output formats for various use cases:

• Pretty format (default): Human-readable colored output
• JSON format: Machine-readable structured data
• Raw format: Simple version string for scripts

Examples:
  gollm version
  gollm version --detailed
  gollm version --output json
  gollm version --short
  gollm version --check-updates`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersionCommand(cmd.Context(), flags)
		},
	}

	addVersionFlags(cmd, flags)
	return cmd
}

// addVersionFlags adds flags to the version command.
func addVersionFlags(cmd *cobra.Command, flags *VersionFlags) {
	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "pretty", "Output format (pretty, json, raw)")
	cmd.Flags().BoolVarP(&flags.Detailed, "detailed", "d", false, "Show detailed build and runtime information")
	cmd.Flags().BoolVar(&flags.CheckUpdates, "check-updates", false, "Check for available updates")
	cmd.Flags().BoolVar(&flags.ShowBuildTags, "show-build-tags", false, "Show build tags used during compilation")
	cmd.Flags().BoolVar(&flags.ShowDeps, "show-deps", false, "Show dependency versions")
	cmd.Flags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Only show version number")
	cmd.Flags().BoolVarP(&flags.Short, "short", "s", false, "Show short version information")
}

// runVersionCommand executes the version command.
func runVersionCommand(ctx context.Context, flags *VersionFlags) error {
	// Handle simple cases first
	if flags.Quiet {
		fmt.Println(Version)
		return nil
	}

	// Create display renderer
	renderer := display.NewRenderer(display.Options{
		Colors:      true,
		Interactive: false,
		Format:      display.Format(flags.OutputFormat),
		Quiet:       flags.Quiet,
	})

	// Gather version information
	versionInfo := gatherVersionInfo(flags)

	// Check for updates if requested
	var updateInfo *UpdateInfo
	if flags.CheckUpdates {
		renderer.Info("Checking for updates...")
		updateInfo = checkForUpdates(ctx)
	}

	// Display version information
	switch flags.OutputFormat {
	case "json":
		displayVersionJSON(versionInfo, updateInfo)
	case "raw":
		displayVersionRaw(versionInfo)
	default:
		displayVersionPretty(versionInfo, updateInfo, flags, renderer)
	}

	return nil
}

// gatherVersionInfo collects comprehensive version and system information.
func gatherVersionInfo(flags *VersionFlags) *VersionInfo {
	info := &VersionInfo{
		Version:         Version,
		GitCommit:       GitCommit,
		BuildTime:       BuildTime,
		GoVersion:       runtime.Version(),
		RuntimeOS:       runtime.GOOS,
		RuntimeArch:     runtime.GOARCH,
		NumCPU:          runtime.NumCPU(),
		NumGoroutines:   runtime.NumGoroutine(),
		CompilerVersion: runtime.Version(),
	}

	// Add build information
	if BuildTags != "" {
		info.BuildTags = strings.Split(BuildTags, ",")
	}
	info.BuildUser = BuildUser
	info.BuildHost = BuildHost
	info.BuildOS = runtime.GOOS
	info.BuildArch = runtime.GOARCH

	// Gather memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	info.MemAllocated = memStats.Alloc
	info.MemSys = memStats.Sys
	info.MemHeapInUse = memStats.HeapInuse
	info.NextGC = memStats.NextGC
	if memStats.LastGC > 0 {
		info.LastGC = time.Unix(0, int64(memStats.LastGC)).Format(time.RFC3339)
	}

	// Add performance metrics if available
	if !startupTime.IsZero() {
		info.StartupTime = time.Since(startupTime).String()
	}
	if configLoadTime > 0 {
		info.ConfigLoadTime = configLoadTime.String()
	}

	// Gather dependency information if requested
	if flags.ShowDeps || flags.Detailed {
		info.Dependencies = gatherDependencyInfo()
	}

	// Gather feature information
	info.Features = gatherFeatureInfo()

	return info
}

// gatherDependencyInfo collects information about dependencies.
func gatherDependencyInfo() []DependencyInfo {
	var deps []DependencyInfo

	// Use build info to get module dependencies
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range buildInfo.Deps {
			depInfo := DependencyInfo{
				Name:    dep.Path,
				Version: dep.Version,
				Sum:     dep.Sum,
			}
			if dep.Replace != nil {
				depInfo.Replace = dep.Replace.Path + "@" + dep.Replace.Version
			}
			deps = append(deps, depInfo)
		}
	}

	return deps
}

// gatherFeatureInfo collects information about available features.
func gatherFeatureInfo() []FeatureInfo {
	features := []FeatureInfo{
		{
			Name:        "streaming",
			Enabled:     true,
			Description: "Streaming response support",
		},
		{
			Name:        "interactive",
			Enabled:     true,
			Description: "Interactive chat mode",
		},
		{
			Name:        "enhanced_interactive",
			Enabled:     true,
			Description: "Enhanced interactive mode with autocomplete",
		},
		{
			Name:        "profiles",
			Enabled:     true,
			Description: "Configuration profiles",
		},
		{
			Name:        "benchmarking",
			Enabled:     true,
			Description: "Performance benchmarking",
		},
		{
			Name:        "multi_provider",
			Enabled:     true,
			Description: "Multiple LLM provider support",
		},
		{
			Name:        "security_audit",
			Enabled:     true,
			Description: "Built-in security auditing",
		},
		{
			Name:        "plugin_system",
			Enabled:     false, // Not yet implemented
			Description: "Plugin system for extensibility",
		},
		{
			Name:        "mcp_support",
			Enabled:     false, // Not yet implemented
			Description: "Model Context Protocol support",
		},
	}

	// Check for CGO support (determined at build time)
	features = append(features, FeatureInfo{
		Name:        "cgo",
		Enabled:     false, // Would need build tags to determine this accurately
		Description: "CGO support for native libraries",
	})

	// Check build constraints
	buildTags := strings.Split(BuildTags, ",")
	for _, tag := range buildTags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			features = append(features, FeatureInfo{
				Name:        "build_tag_" + tag,
				Enabled:     true,
				Description: "Build tag: " + tag,
			})
		}
	}

	return features
}

// checkForUpdates checks for available updates (placeholder implementation).
func checkForUpdates(ctx context.Context) *UpdateInfo {
	// In a real implementation, this would:
	// 1. Query GitHub releases API
	// 2. Compare current version with latest
	// 3. Return update information

	// For now, return a placeholder
	return &UpdateInfo{
		Available: false,
	}
}

// displayVersionPretty displays version information in a human-readable format.
func displayVersionPretty(info *VersionInfo, updateInfo *UpdateInfo, flags *VersionFlags, renderer *display.Renderer) {
	if flags.Short {
		fmt.Printf("GOLLM %s\n", info.Version)
		return
	}

	// Display ASCII logo
	displayGOLLMLogo()
	fmt.Println()

	// Header
	renderer.Info("🚀 GOLLM - High-Performance LLM CLI")
	renderer.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Basic version information
	fmt.Printf("Version:      %s\n", info.Version)
	fmt.Printf("Git Commit:   %s\n", info.GitCommit)
	fmt.Printf("Build Time:   %s\n", info.BuildTime)
	fmt.Printf("Go Version:   %s\n", info.GoVersion)

	// Build information
	if flags.Detailed {
		fmt.Println("\n📦 Build Information:")
		fmt.Printf("  OS/Arch:    %s/%s\n", info.BuildOS, info.BuildArch)
		if info.BuildUser != "unknown" {
			fmt.Printf("  Built by:   %s@%s\n", info.BuildUser, info.BuildHost)
		}
		if len(info.BuildTags) > 0 {
			fmt.Printf("  Build Tags: %s\n", strings.Join(info.BuildTags, ", "))
		}

		// Runtime information
		fmt.Println("\n⚙️  Runtime Information:")
		fmt.Printf("  OS/Arch:     %s/%s\n", info.RuntimeOS, info.RuntimeArch)
		fmt.Printf("  CPUs:        %d\n", info.NumCPU)
		fmt.Printf("  Goroutines:  %d\n", info.NumGoroutines)

		// Memory information
		fmt.Println("\n💾 Memory Usage:")
		fmt.Printf("  Allocated:   %s\n", formatBytes(info.MemAllocated))
		fmt.Printf("  System:      %s\n", formatBytes(info.MemSys))
		fmt.Printf("  Heap InUse:  %s\n", formatBytes(info.MemHeapInUse))
		if info.NextGC > 0 {
			fmt.Printf("  Next GC:     %s\n", formatBytes(info.NextGC))
		}
		if info.LastGC != "" {
			fmt.Printf("  Last GC:     %s\n", info.LastGC)
		}

		// Performance metrics
		if info.StartupTime != "" || info.ConfigLoadTime != "" {
			fmt.Println("\n⚡ Performance:")
			if info.StartupTime != "" {
				fmt.Printf("  Startup:     %s\n", info.StartupTime)
			}
			if info.ConfigLoadTime != "" {
				fmt.Printf("  Config Load: %s\n", info.ConfigLoadTime)
			}
		}
	}

	// Features
	if flags.Detailed && len(info.Features) > 0 {
		fmt.Println("\n✨ Features:")
		enabledFeatures := 0
		for _, feature := range info.Features {
			if feature.Enabled {
				enabledFeatures++
			}
			status := "❌"
			if feature.Enabled {
				status = "✅"
			}
			fmt.Printf("  %s %s", status, feature.Name)
			if feature.Description != "" {
				fmt.Printf(" - %s", feature.Description)
			}
			fmt.Println()
		}
		fmt.Printf("\nEnabled: %d/%d features\n", enabledFeatures, len(info.Features))
	}

	// Dependencies
	if flags.ShowDeps && len(info.Dependencies) > 0 {
		fmt.Printf("\n📚 Dependencies (%d):\n", len(info.Dependencies))
		for _, dep := range info.Dependencies {
			fmt.Printf("  %s %s\n", dep.Name, dep.Version)
			if dep.Replace != "" {
				fmt.Printf("    └─ replaced by %s\n", dep.Replace)
			}
		}
	}

	// Update information
	if updateInfo != nil {
		fmt.Println("\n🔄 Updates:")
		if updateInfo.Available {
			renderer.Warning(fmt.Sprintf("Update available: %s → %s", info.Version, updateInfo.LatestVersion))
			if updateInfo.ReleaseURL != "" {
				fmt.Printf("  Release: %s\n", updateInfo.ReleaseURL)
			}
		} else {
			renderer.Success("You are running the latest version")
		}
	}

	fmt.Println()
}

// displayVersionJSON displays version information in JSON format.
func displayVersionJSON(info *VersionInfo, updateInfo *UpdateInfo) {
	output := map[string]interface{}{
		"version": info,
	}

	if updateInfo != nil {
		output["updates"] = updateInfo
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

// displayVersionRaw displays just the version string.
func displayVersionRaw(info *VersionInfo) {
	fmt.Println(info.Version)
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// displayGOLLMLogo displays the GOLLM ASCII logo with colors.
func displayGOLLMLogo() {
	logo := `░▒▓███████▓▒░░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓███████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓██████▓▒░  ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░░▒▓███████▓▒░`

	// Apply gradient colors if color is supported
	if shouldUseColors() {
		lines := strings.Split(logo, "\n")
		colors := []*color.Color{
			color.New(color.FgHiCyan),    // Top line
			color.New(color.FgCyan),      // Second line
			color.New(color.FgHiBlue),    // Third line
			color.New(color.FgBlue),      // Middle line (main focus)
			color.New(color.FgHiMagenta), // Fifth line
			color.New(color.FgMagenta),   // Sixth line
			color.New(color.FgHiRed),     // Bottom line
		}

		for i, line := range lines {
			if i < len(colors) {
				fmt.Println(colors[i].Sprint(line))
			} else {
				fmt.Println(line)
			}
		}

		// Add tagline
		tagline := "\n              High-Performance CLI for Large Language Models\n            🚀 Lightning Fast • 🔗 Multi-Provider • 🎯 Enterprise Ready"
		fmt.Println(color.New(color.FgHiWhite, color.Bold).Sprint(tagline))
	} else {
		fmt.Println(logo)
		fmt.Println("\n              High-Performance CLI for Large Language Models")
		fmt.Println("            🚀 Lightning Fast • 🔗 Multi-Provider • 🎯 Enterprise Ready")
	}
}

// shouldUseColors determines if colors should be used based on environment and terminal support.
func shouldUseColors() bool {
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for FORCE_COLOR environment variable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Use fatih/color's built-in detection
	return !color.NoColor
}

// SetStartupMetrics sets startup performance metrics.
func SetStartupMetrics(startup time.Time, configLoad time.Duration) {
	startupTime = startup
	configLoadTime = configLoad
}
