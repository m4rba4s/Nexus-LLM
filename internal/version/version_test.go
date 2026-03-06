// Package version provides build information and version details for GOLLM.
package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetBuildInfo(t *testing.T) {
	t.Parallel()

	buildInfo := GetBuildInfo()

	// Validate that we get a non-nil BuildInfo
	if buildInfo == nil {
		t.Fatal("GetBuildInfo() returned nil")
	}

	// Validate that all fields are populated with some value
	if buildInfo.Version == "" {
		t.Error("Version should not be empty")
	}

	if buildInfo.Commit == "" {
		t.Error("Commit should not be empty")
	}

	if buildInfo.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}

	if buildInfo.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	if buildInfo.Platform == "" {
		t.Error("Platform should not be empty")
	}

	// Validate that GoVersion matches runtime.Version()
	if buildInfo.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %s, want %s", buildInfo.GoVersion, runtime.Version())
	}
}

func TestBuildInfo_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     BuildInfo
		contains []string // strings that should be present in the output
		excludes []string // strings that should NOT be present in the output
	}{
		{
			name: "release version with full info",
			info: BuildInfo{
				Version:   "1.0.0",
				Commit:    "abcdef123456",
				BuildTime: "2024-01-15T10:30:00Z",
				GoVersion: "go1.21.5",
				Platform:  "linux/amd64",
			},
			contains: []string{"version 1.0.0", "commit abcdef1", "built 2024-01-15T10:30:00Z", "with go1.21.5", "for linux/amd64"},
		},
		{
			name: "development build",
			info: BuildInfo{
				Version:   "dev",
				Commit:    "xyz789",
				BuildTime: "unknown",
				GoVersion: "go1.21.5",
				Platform:  "darwin/arm64",
			},
			contains: []string{"development build", "commit xyz789", "with go1.21.5", "for darwin/arm64"},
			excludes: []string{"version dev", "built unknown"},
		},
		{
			name: "minimal info",
			info: BuildInfo{
				Version:   "",
				Commit:    "",
				BuildTime: "",
				GoVersion: "",
				Platform:  "",
			},
			contains: []string{"development build"},
		},
		{
			name: "unknown values",
			info: BuildInfo{
				Version:   "dev",
				Commit:    "unknown",
				BuildTime: "unknown",
				GoVersion: "go1.21.5",
				Platform:  "windows/amd64",
			},
			contains: []string{"development build", "with go1.21.5", "for windows/amd64"},
			excludes: []string{"commit unknown", "built unknown"},
		},
		{
			name: "short commit hash",
			info: BuildInfo{
				Version:   "2.0.0",
				Commit:    "abc123",
				BuildTime: "2024-01-15T10:30:00Z",
				GoVersion: "go1.21.5",
				Platform:  "linux/amd64",
			},
			contains: []string{"version 2.0.0", "commit abc123", "built 2024-01-15T10:30:00Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.info.String()

			for _, contain := range tt.contains {
				if !strings.Contains(result, contain) {
					t.Errorf("String() = %q, should contain %q", result, contain)
				}
			}

			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("String() = %q, should not contain %q", result, exclude)
				}
			}
		})
	}
}

func TestBuildInfo_Short(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info BuildInfo
		want string
	}{
		{
			name: "release version",
			info: BuildInfo{Version: "1.0.0", Commit: "abc123"},
			want: "1.0.0",
		},
		{
			name: "dev version with commit",
			info: BuildInfo{Version: "dev", Commit: "abcdef123456"},
			want: "dev-abcdef1",
		},
		{
			name: "dev version with short commit",
			info: BuildInfo{Version: "dev", Commit: "abc123"},
			want: "dev-abc123",
		},
		{
			name: "empty version with commit",
			info: BuildInfo{Version: "", Commit: "xyz789abc"},
			want: "dev-xyz789a",
		},
		{
			name: "no version, unknown commit",
			info: BuildInfo{Version: "", Commit: "unknown"},
			want: "dev",
		},
		{
			name: "no version, empty commit",
			info: BuildInfo{Version: "", Commit: ""},
			want: "dev",
		},
		{
			name: "pre-release version",
			info: BuildInfo{Version: "1.0.0-rc1", Commit: "abc123"},
			want: "1.0.0-rc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.info.Short(); got != tt.want {
				t.Errorf("Short() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildInfo_Detailed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     BuildInfo
		contains []string
		excludes []string
	}{
		{
			name: "full release info",
			info: BuildInfo{
				Version:   "1.0.0",
				Commit:    "abcdef123456",
				BuildTime: "2024-01-15T10:30:00Z",
				GoVersion: "go1.21.5",
				Platform:  "linux/amd64",
			},
			contains: []string{
				"GOLLM 1.0.0",
				"Version:    1.0.0",
				"Commit:     abcdef123456",
				"Build Time: 2024-01-15T10:30:00Z",
				"Go Version: go1.21.5",
				"Platform:   linux/amd64",
			},
		},
		{
			name: "development build",
			info: BuildInfo{
				Version:   "dev",
				Commit:    "xyz789abc",
				BuildTime: "unknown",
				GoVersion: "go1.21.5",
				Platform:  "darwin/arm64",
			},
			contains: []string{
				"GOLLM dev-xyz789a",
				"Version:    dev",
				"Commit:     xyz789abc",
				"Go Version: go1.21.5",
				"Platform:   darwin/arm64",
			},
			excludes: []string{"Build Time: unknown"},
		},
		{
			name: "minimal info",
			info: BuildInfo{
				Version:   "",
				Commit:    "",
				BuildTime: "",
				GoVersion: "go1.21.5",
				Platform:  "windows/amd64",
			},
			contains: []string{
				"GOLLM dev",
				"Go Version: go1.21.5",
				"Platform:   windows/amd64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.info.Detailed()

			for _, contain := range tt.contains {
				if !strings.Contains(result, contain) {
					t.Errorf("Detailed() should contain %q, got:\n%s", contain, result)
				}
			}

			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("Detailed() should not contain %q, got:\n%s", exclude, result)
				}
			}

			// Check that it's multi-line
			lines := strings.Split(result, "\n")
			if len(lines) < 3 {
				t.Errorf("Detailed() should be multi-line, got %d lines", len(lines))
			}
		})
	}
}

func TestBuildInfo_IsRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"release version", "1.0.0", true},
		{"pre-release", "1.0.0-rc1", true},
		{"patch version", "1.0.1", true},
		{"major version", "2.0.0", true},
		{"dev version", "dev", false},
		{"empty version", "", false},
		{"dev in version", "1.0.0-dev", false},
		{"development", "development", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := &BuildInfo{Version: tt.version}
			if got := info.IsRelease(); got != tt.want {
				t.Errorf("IsRelease() = %v, want %v for version %q", got, tt.want, tt.version)
			}
		})
	}
}

func TestBuildInfo_IsDevelopment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"dev version", "dev", true},
		{"empty version", "", true},
		{"dev in version", "1.0.0-dev", true},
		{"development", "development", true},
		{"release version", "1.0.0", false},
		{"pre-release", "1.0.0-rc1", false},
		{"patch version", "1.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := &BuildInfo{Version: tt.version}
			if got := info.IsDevelopment(); got != tt.want {
				t.Errorf("IsDevelopment() = %v, want %v for version %q", got, tt.want, tt.version)
			}
		})
	}
}

func TestBuildInfo_UserAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info BuildInfo
		want string
	}{
		{
			name: "release version",
			info: BuildInfo{
				Version:  "1.0.0",
				Platform: "linux/amd64",
			},
			want: "gollm/1.0.0 (linux/amd64)",
		},
		{
			name: "dev version with commit",
			info: BuildInfo{
				Version:  "dev",
				Commit:   "abc123def",
				Platform: "darwin/arm64",
			},
			want: "gollm/dev-abc123d (darwin/arm64)",
		},
		{
			name: "empty version",
			info: BuildInfo{
				Version:  "",
				Commit:   "xyz789",
				Platform: "windows/amd64",
			},
			want: "gollm/dev-xyz789 (windows/amd64)",
		},
		{
			name: "no commit",
			info: BuildInfo{
				Version:  "",
				Commit:   "",
				Platform: "linux/arm64",
			},
			want: "gollm/dev (linux/arm64)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.info.UserAgent(); got != tt.want {
				t.Errorf("UserAgent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildInfo_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("zero value build info", func(t *testing.T) {
		t.Parallel()
		info := &BuildInfo{}

		// These should not panic with zero-value receiver
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Method panicked with zero-value receiver: %v", r)
			}
		}()

		_ = info.String()
		_ = info.Short()
		_ = info.Detailed()
		_ = info.IsRelease()
		_ = info.IsDevelopment()
		_ = info.UserAgent()
	})

	t.Run("very long commit hash", func(t *testing.T) {
		t.Parallel()
		info := BuildInfo{
			Commit: "abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnopqrstuvwxyz",
		}

		short := info.Short()
		if !strings.HasPrefix(short, "dev-abcdefg") {
			t.Errorf("Short() should truncate long commit, got %q", short)
		}

		str := info.String()
		if !strings.Contains(str, "commit abcdefg") {
			t.Errorf("String() should truncate long commit, got %q", str)
		}
	})

	t.Run("special characters in version", func(t *testing.T) {
		t.Parallel()
		info := BuildInfo{
			Version:  "1.0.0+build.123",
			Platform: "linux/amd64",
		}

		userAgent := info.UserAgent()
		expected := "gollm/1.0.0+build.123 (linux/amd64)"
		if userAgent != expected {
			t.Errorf("UserAgent() = %q, want %q", userAgent, expected)
		}
	})
}

func TestBuildInfo_StringFormat(t *testing.T) {
	t.Parallel()

	info := BuildInfo{
		Version:   "1.0.0",
		Commit:    "abc123",
		BuildTime: "2024-01-15T10:30:00Z",
		GoVersion: "go1.21.5",
		Platform:  "linux/amd64",
	}

	str := info.String()

	// Check that components are separated by ", "
	parts := strings.Split(str, ", ")
	if len(parts) < 2 {
		t.Errorf("String() should contain comma-separated parts, got %q", str)
	}

	// Check order - version should come first
	if !strings.HasPrefix(parts[0], "version ") {
		t.Errorf("String() should start with version, got %q", str)
	}
}

// Benchmark tests for performance validation
func BenchmarkBuildInfo_String(b *testing.B) {
	info := BuildInfo{
		Version:   "1.0.0",
		Commit:    "abcdef123456",
		BuildTime: "2024-01-15T10:30:00Z",
		GoVersion: "go1.21.5",
		Platform:  "linux/amd64",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.String()
	}
}

func BenchmarkBuildInfo_Short(b *testing.B) {
	info := BuildInfo{
		Version: "dev",
		Commit:  "abcdef123456",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.Short()
	}
}

func BenchmarkBuildInfo_UserAgent(b *testing.B) {
	info := BuildInfo{
		Version:  "1.0.0",
		Platform: "linux/amd64",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.UserAgent()
	}
}

func BenchmarkGetBuildInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetBuildInfo()
	}
}
