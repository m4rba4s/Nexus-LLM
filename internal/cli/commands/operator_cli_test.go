package commands

import "testing"

func TestNewOperatorCommand_Basics(t *testing.T) {
    cmd := NewOperatorCommand()
    if cmd.Use != "operator" {
        t.Fatalf("unexpected Use: %s", cmd.Use)
    }
    if cmd.Flags().Lookup("action") == nil {
        t.Fatalf("expected flag 'action'")
    }
}

