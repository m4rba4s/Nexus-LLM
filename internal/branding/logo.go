// Package cli provides ASCII logo and branding for GOLLM CLI.
//
// This package contains the GOLLM ASCII logo and related branding functions
// for consistent display across different commands and contexts.
package branding

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

const (
	// ASCII logo for GOLLM - main branding
	asciiLogo = `░▒▓███████▓▒░░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓███████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓██████▓▒░  ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░░▒▓███████▓▒░`

	// Compact logo for smaller displays
	compactLogo = `  ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░
  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░
  ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░
  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░  ░▒▓█▓▒░
  ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░ ░▒▓██▓▒░`

	// Mini logo for very compact displays
	miniLogo = `GOLLM ░▒▓█▓▒░`
)

// LogoOptions configures how the logo is displayed
type LogoOptions struct {
	ShowTagline   bool   // Show tagline below logo
	ShowVersion   bool   // Show version info
	Colored       bool   // Use colors (auto-detected if not specified)
	Compact       bool   // Use compact version
	Mini          bool   // Use mini version (overrides Compact)
	CenterAlign   bool   // Center align the logo
	Width         int    // Terminal width for centering (0 = auto-detect)
	CustomTagline string // Custom tagline instead of default
}

// DefaultLogoOptions returns sensible defaults for logo display
func DefaultLogoOptions() LogoOptions {
	return LogoOptions{
		ShowTagline:   true,
		ShowVersion:   false,
		Colored:       true, // Will be auto-detected
		Compact:       false,
		Mini:          false,
		CenterAlign:   false,
		Width:         0,
		CustomTagline: "",
	}
}

// DisplayLogo prints the GOLLM logo with the specified options
func DisplayLogo(opts LogoOptions) {
	fmt.Print(GetLogo(opts))
}

// GetLogo returns the GOLLM logo as a string with the specified options
func GetLogo(opts LogoOptions) string {
	var result strings.Builder

	// Select logo variant
	var logo string
	switch {
	case opts.Mini:
		logo = miniLogo
	case opts.Compact:
		logo = compactLogo
	default:
		logo = asciiLogo
	}

	// Apply colors if enabled
	if opts.Colored && shouldUseColor() {
		logo = colorizeGOLLMLogo(logo)
	}

	// Center align if requested
	if opts.CenterAlign {
		logo = centerAlign(logo, opts.Width)
	}

	result.WriteString(logo)

	// Add tagline
	if opts.ShowTagline {
		tagline := opts.CustomTagline
		if tagline == "" {
			tagline = getDefaultTagline(opts.Mini)
		}

		if opts.Colored && shouldUseColor() {
			tagline = colorizeTagline(tagline)
		}

		if opts.CenterAlign {
			tagline = centerAlign(tagline, opts.Width)
		}

		result.WriteString("\n")
		result.WriteString(tagline)
	}

	// Add version info
	if opts.ShowVersion {
		// This would integrate with version info when available
		versionInfo := "\n                      v1.0.0 - High-Performance LLM CLI"
		if opts.Colored && shouldUseColor() {
			versionInfo = color.New(color.FgHiBlack).Sprint(versionInfo)
		}
		if opts.CenterAlign {
			versionInfo = centerAlign(strings.TrimLeft(versionInfo, "\n"), opts.Width)
			versionInfo = "\n" + versionInfo
		}
		result.WriteString(versionInfo)
	}

	return result.String()
}

// GetPlainLogo returns the logo without any colors or formatting
func GetPlainLogo() string {
	return asciiLogo
}

// GetCompactLogo returns the compact version without colors
func GetCompactLogo() string {
	return compactLogo
}

// GetMiniLogo returns the mini version without colors
func GetMiniLogo() string {
	return miniLogo
}

// colorizeGOLLMLogo applies colors to the main GOLLM logo
func colorizeGOLLMLogo(logo string) string {
	// Use gradient-like colors for the logo
	lines := strings.Split(logo, "\n")
	var coloredLines []string

	colors := []*color.Color{
		color.New(color.FgHiCyan),    // Top line - bright cyan
		color.New(color.FgCyan),      // Second line - cyan
		color.New(color.FgHiBlue),    // Third line - bright blue
		color.New(color.FgBlue),      // Middle line - blue (main focus)
		color.New(color.FgHiMagenta), // Fifth line - bright magenta
		color.New(color.FgMagenta),   // Sixth line - magenta
		color.New(color.FgHiRed),     // Bottom line - bright red
	}

	for i, line := range lines {
		if i < len(colors) {
			coloredLines = append(coloredLines, colors[i].Sprint(line))
		} else {
			// Fallback to default color
			coloredLines = append(coloredLines, color.New(color.FgHiCyan).Sprint(line))
		}
	}

	return strings.Join(coloredLines, "\n")
}

// colorizeTagline applies colors to the tagline
func colorizeTagline(tagline string) string {
	// Make tagline stand out with bold white
	return color.New(color.FgHiWhite, color.Bold).Sprint(tagline)
}

// getDefaultTagline returns the default tagline based on logo type
func getDefaultTagline(mini bool) string {
	if mini {
		return "High-Performance LLM CLI"
	}
	return `
              High-Performance CLI for Large Language Models
            🚀 Lightning Fast • 🔗 Multi-Provider • 🎯 Enterprise Ready`
}

// centerAlign centers text within the specified width
func centerAlign(text string, width int) string {
	if width <= 0 {
		// Try to detect terminal width, fallback to 80
		width = 80 // This could be enhanced with terminal width detection
	}

	lines := strings.Split(text, "\n")
	var centeredLines []string

	for _, line := range lines {
		// Remove ANSI codes for length calculation
		plainLine := stripAnsiCodes(line)
		lineLen := len(plainLine)

		if lineLen >= width {
			centeredLines = append(centeredLines, line)
			continue
		}

		padding := (width - lineLen) / 2
		centeredLine := strings.Repeat(" ", padding) + line
		centeredLines = append(centeredLines, centeredLine)
	}

	return strings.Join(centeredLines, "\n")
}

// stripAnsiCodes removes ANSI escape codes for length calculation
func stripAnsiCodes(text string) string {
	// Simple ANSI code stripping for length calculation
	// This is a basic implementation - could be enhanced
	result := text

	// Remove common ANSI escape sequences
	ansiPatterns := []string{
		"\x1b[0m",  // Reset
		"\x1b[1m",  // Bold
		"\x1b[31m", // Red
		"\x1b[32m", // Green
		"\x1b[33m", // Yellow
		"\x1b[34m", // Blue
		"\x1b[35m", // Magenta
		"\x1b[36m", // Cyan
		"\x1b[37m", // White
		"\x1b[91m", // Bright Red
		"\x1b[92m", // Bright Green
		"\x1b[93m", // Bright Yellow
		"\x1b[94m", // Bright Blue
		"\x1b[95m", // Bright Magenta
		"\x1b[96m", // Bright Cyan
		"\x1b[97m", // Bright White
	}

	for _, pattern := range ansiPatterns {
		result = strings.ReplaceAll(result, pattern, "")
	}

	return result
}

// shouldUseColor determines if colors should be used
func shouldUseColor() bool {
	// Check for NO_COLOR environment variable
	if noColor := getEnv("NO_COLOR"); noColor != "" {
		return false
	}

	// Check for FORCE_COLOR environment variable
	if forceColor := getEnv("FORCE_COLOR"); forceColor != "" {
		return true
	}

	// Auto-detect color support
	return color.NoColor == false
}

// getEnv is a helper to get environment variables
func getEnv(key string) string {
	return os.Getenv(key)
}

// WelcomeBanner displays a complete welcome banner with logo
func WelcomeBanner(version string) {
	opts := LogoOptions{
		ShowTagline:   true,
		ShowVersion:   true,
		Colored:       true,
		CenterAlign:   true,
		CustomTagline: "",
	}

	fmt.Println()
	DisplayLogo(opts)
	fmt.Println()

	if shouldUseColor() {
		fmt.Printf("   %s\n",
			color.New(color.FgHiGreen).Sprintf("✅ Ready to revolutionize your LLM workflow!"))
		fmt.Printf("   %s\n",
			color.New(color.FgHiBlue).Sprintf("🔗 Multiple providers • ⚡ Sub-100ms startup • 🎯 Production ready"))
		fmt.Println()
	} else {
		fmt.Println("   ✅ Ready to revolutionize your LLM workflow!")
		fmt.Println("   🔗 Multiple providers • ⚡ Sub-100ms startup • 🎯 Production ready")
		fmt.Println()
	}
}

// StartupLogo displays a minimal startup logo
func StartupLogo() {
	opts := LogoOptions{
		ShowTagline: false,
		ShowVersion: false,
		Colored:     true,
		Mini:        true,
	}
	DisplayLogo(opts)
}

// InteractiveBanner displays banner for interactive mode
func InteractiveBanner() {
	opts := LogoOptions{
		ShowTagline:   true,
		Colored:       true,
		Compact:       true,
		CustomTagline: "🎮 Interactive Mode • Type /help for commands • /quit to exit",
	}

	fmt.Println()
	DisplayLogo(opts)
	fmt.Println()
}
