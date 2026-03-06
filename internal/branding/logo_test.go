package branding

import (
	"strings"
	"testing"
)

func TestGetPlainLogo(t *testing.T) {
	t.Parallel()
	logo := GetPlainLogo()
	if logo == "" {
		t.Fatal("GetPlainLogo returned empty string")
	}
	// Logo uses block characters, just verify it's non-trivial
	if len(logo) < 20 {
		t.Errorf("logo too short: %d chars", len(logo))
	}
}

func TestGetCompactLogo(t *testing.T) {
	t.Parallel()
	logo := GetCompactLogo()
	if logo == "" {
		t.Fatal("GetCompactLogo returned empty string")
	}
}

func TestGetMiniLogo(t *testing.T) {
	t.Parallel()
	logo := GetMiniLogo()
	if !strings.Contains(logo, "GOLLM") {
		t.Errorf("mini logo should contain GOLLM, got: %q", logo)
	}
}

func TestGetLogo_DefaultOptions(t *testing.T) {
	t.Parallel()
	opts := DefaultLogoOptions()
	logo := GetLogo(opts)
	if logo == "" {
		t.Fatal("GetLogo with defaults returned empty")
	}
}

func TestGetLogo_Compact(t *testing.T) {
	t.Parallel()
	opts := LogoOptions{Compact: true}
	logo := GetLogo(opts)
	if logo == "" {
		t.Fatal("GetLogo compact returned empty")
	}
}

func TestGetLogo_Mini(t *testing.T) {
	t.Parallel()
	opts := LogoOptions{Mini: true}
	logo := GetLogo(opts)
	if !strings.Contains(logo, "GOLLM") {
		t.Errorf("mini logo should contain GOLLM, got: %q", logo)
	}
}

func TestCenterAlign(t *testing.T) {
	t.Parallel()
	result := centerAlign("hello", 20)
	if len(result) == 0 {
		t.Fatal("centerAlign returned empty")
	}
	// Should have leading spaces
	if result[0] != ' ' {
		t.Errorf("expected leading space for centered text, got: %q", result)
	}
}

func TestStripAnsiCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"plain text", "plain text"},
		{"\033[31mred\033[0m", "red"},
		{"no ansi", "no ansi"},
	}

	for _, tt := range tests {
		got := stripAnsiCodes(tt.input)
		if got != tt.want {
			t.Errorf("stripAnsiCodes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultLogoOptions(t *testing.T) {
	t.Parallel()
	opts := DefaultLogoOptions()
	if !opts.ShowTagline {
		t.Error("default should show tagline")
	}
	if !opts.Colored {
		t.Error("default should have coloring enabled")
	}
}
