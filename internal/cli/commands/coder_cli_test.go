package commands

import "testing"

func TestNewCoderCommand_Basics(t *testing.T) {
    cmd := NewCoderCommand()
    if cmd.Use != "coder" {
        t.Fatalf("unexpected Use: %s", cmd.Use)
    }
    // Ensure flags exist
    for _, f := range []string{"provider", "model", "operator", "prompt", "files", "output-file", "no-stream", "non-interactive", "diff", "apply", "apply-to", "backup-ext", "yes", "patch"} {
        if cmd.Flags().Lookup(f) == nil {
            t.Fatalf("expected flag %q", f)
        }
    }
}
