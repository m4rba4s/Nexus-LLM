// Package version provides build information and version details for GOLLM.
package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Build information set via ldflags during compilation
var (
	// Version is the semantic version of the build
	Version = "dev"

	// Commit is the git commit hash of the build
	Commit = "unknown"

	// BuildTime is the timestamp when the binary was built
	BuildTime = "unknown"

	// GoVersion is the Go version used to build the binary
	GoVersion = runtime.Version()

	// Platform is the target platform for the build
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// BuildInfo contains all version and build information
type BuildInfo struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	BuildTime string `json:"build_time" yaml:"build_time"`
	GoVersion string `json:"go_version" yaml:"go_version"`
	Platform  string `json:"platform" yaml:"platform"`
}

// GetBuildInfo returns structured build information
func GetBuildInfo() *BuildInfo {
	return &BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// String returns a human-readable version string
func (b *BuildInfo) String() string {
	var parts []string

	if b.Version != "" && b.Version != "dev" {
		parts = append(parts, fmt.Sprintf("version %s", b.Version))
	} else {
		parts = append(parts, "development build")
	}

	if b.Commit != "" && b.Commit != "unknown" {
		if len(b.Commit) > 7 {
			parts = append(parts, fmt.Sprintf("commit %s", b.Commit[:7]))
		} else {
			parts = append(parts, fmt.Sprintf("commit %s", b.Commit))
		}
	}

	if b.BuildTime != "" && b.BuildTime != "unknown" {
		parts = append(parts, fmt.Sprintf("built %s", b.BuildTime))
	}

	if b.GoVersion != "" {
		parts = append(parts, fmt.Sprintf("with %s", b.GoVersion))
	}

	if b.Platform != "" {
		parts = append(parts, fmt.Sprintf("for %s", b.Platform))
	}

	return strings.Join(parts, ", ")
}

// Short returns a short version string suitable for CLI output
func (b *BuildInfo) Short() string {
	if b.Version != "" && b.Version != "dev" {
		return b.Version
	}

	if b.Commit != "" && b.Commit != "unknown" {
		if len(b.Commit) > 7 {
			return fmt.Sprintf("dev-%s", b.Commit[:7])
		}
		return fmt.Sprintf("dev-%s", b.Commit)
	}

	return "dev"
}

// Detailed returns a detailed multi-line version string
func (b *BuildInfo) Detailed() string {
	lines := []string{
		fmt.Sprintf("GOLLM %s", b.Short()),
		"",
	}

	if b.Version != "" {
		lines = append(lines, fmt.Sprintf("Version:    %s", b.Version))
	}

	if b.Commit != "" && b.Commit != "unknown" {
		lines = append(lines, fmt.Sprintf("Commit:     %s", b.Commit))
	}

	if b.BuildTime != "" && b.BuildTime != "unknown" {
		lines = append(lines, fmt.Sprintf("Build Time: %s", b.BuildTime))
	}

	if b.GoVersion != "" {
		lines = append(lines, fmt.Sprintf("Go Version: %s", b.GoVersion))
	}

	if b.Platform != "" {
		lines = append(lines, fmt.Sprintf("Platform:   %s", b.Platform))
	}

	return strings.Join(lines, "\n")
}

// IsRelease returns true if this is a release build (not dev)
func (b *BuildInfo) IsRelease() bool {
	return b.Version != "" && b.Version != "dev" && !strings.Contains(b.Version, "dev")
}

// IsDevelopment returns true if this is a development build
func (b *BuildInfo) IsDevelopment() bool {
	return !b.IsRelease()
}

// UserAgent returns a user agent string for HTTP requests
func (b *BuildInfo) UserAgent() string {
	version := b.Short()
	return fmt.Sprintf("gollm/%s (%s)", version, b.Platform)
}
