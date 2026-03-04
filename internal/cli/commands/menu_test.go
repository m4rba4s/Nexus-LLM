package commands

import (
    "testing"
)

func TestNewMenuCommand_Basics(t *testing.T) {
    cmd := NewMenuCommand()
    if cmd.Use != "menu" {
        t.Fatalf("unexpected Use: %s", cmd.Use)
    }
    if cmd.RunE == nil {
        t.Fatalf("RunE should not be nil")
    }
    if cmd.Short == "" {
        t.Fatalf("Short description should not be empty")
    }
    for _, f := range []string{"provider", "model", "operator"} {
        if cmd.Flags().Lookup(f) == nil {
            t.Fatalf("expected flag %q", f)
        }
    }
}
