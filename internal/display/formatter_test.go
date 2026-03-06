package display

import (
	"strings"
	"testing"
	"time"

	"github.com/m4rba4s/Nexus-LLM/internal/core"
)

func TestNewResponseFormatter(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()
	if f == nil {
		t.Fatal("NewResponseFormatter returned nil")
	}
}

func TestDefaultFormatterConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultFormatterConfig()
	if cfg.MaxWidth == 0 {
		t.Error("MaxWidth should have a default value")
	}
	if cfg.Format == "" {
		t.Error("Format should have a default value")
	}
}

func TestResponseFormatter_SetFormat(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()
	f.SetFormat(FormatMarkdown)
	f.SetFormat(FormatPlain)
	f.SetFormat(FormatAuto)
	// No panic = pass
}

func TestResponseFormatter_SetMode(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()
	f.SetMode(ModeCompact)
	f.SetMode(ModeVerbose)
	f.SetMode(ModeQuiet)
}

func TestResponseFormatter_FormatResponse(t *testing.T) {
	t.Parallel()

	f := NewResponseFormatter()
	resp := &core.CompletionResponse{
		Choices: []core.Choice{
			{Message: core.Message{Role: core.RoleAssistant, Content: "Hello, world!"}},
		},
		Usage: core.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}

	for _, format := range []Format{FormatPlain, FormatMarkdown, FormatAuto} {
		t.Run(string(format), func(t *testing.T) {
			t.Parallel()
			f2 := NewResponseFormatterWithConfig(FormatterConfig{
				Format: format,
				Mode:   ModeCompact,
			})
			out, err := f2.FormatResponse(resp, 100*time.Millisecond)
			if err != nil {
				t.Fatalf("FormatResponse(%s): %v", format, err)
			}
			if !strings.Contains(out, "Hello, world!") {
				t.Errorf("FormatResponse(%s) missing content, got: %q", format, out)
			}
		})
	}

	_ = f // prevent unused
}

func TestResponseFormatter_FormatError(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()
	result := f.FormatError(core.ErrTimeout)
	if result == "" {
		t.Fatal("FormatError returned empty string")
	}
}

func TestResponseFormatter_FormatTable(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()

	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"key1", "val1"},
		{"key2", "val2"},
	}

	table := f.FormatTable(headers, rows)
	if !strings.Contains(table, "key1") {
		t.Errorf("FormatTable missing row data, got: %q", table)
	}
	if !strings.Contains(table, "Name") {
		t.Errorf("FormatTable missing header, got: %q", table)
	}
}

func TestResponseFormatter_FormatProgressBar(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()

	bar := f.FormatProgressBar(50, 100, "Loading")
	if bar == "" {
		t.Fatal("FormatProgressBar returned empty string")
	}
}

func TestResponseFormatter_FormatStreamChunk(t *testing.T) {
	t.Parallel()
	f := NewResponseFormatter()
	f.SetStreamingMode(true)

	chunk := core.StreamChunk{
		Choices: []core.Choice{
			{Message: core.Message{Content: "partial "}},
		},
		Done: false,
	}
	out, err := f.FormatStreamChunk(chunk)
	if err != nil {
		t.Fatalf("FormatStreamChunk: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty chunk output")
	}
}
