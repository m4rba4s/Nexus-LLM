package commands

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"

    "github.com/yourusername/gollm/internal/modes/operator"
    branding "github.com/yourusername/gollm/internal/branding"
)

type OperatorFlags struct {
    Action string // command|file|service|package
}

// NewOperatorCommand exposes the PC-Operator flows with a simple action flag.
func NewOperatorCommand() *cobra.Command {
    flags := &OperatorFlags{}
    cmd := &cobra.Command{
        Use:   "operator",
        Short: "PC-Operator flows (commands, files, services, packages)",
        Long: `Run PC-Operator tasks with safety checks and confirmations.

Examples:
  gollm operator --action command
  gollm operator --action file
  gollm operator --action service
  gollm operator --action package`,
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            if ctx == nil {
                ctx = context.Background()
            }
            // Minimal logo for visual identity
            branding.StartupLogo()
            reader := bufio.NewReader(os.Stdin)
            out := os.Stdout

            switch strings.ToLower(flags.Action) {
            case "command", "cmd":
                return operator.RunCommandFlow(ctx, reader, out)
            case "file", "edit":
                return operator.EditFileFlow(ctx, reader, out)
            case "service", "svc":
                return operator.ServiceFlow(ctx, reader, out)
            case "package", "pkg":
                return operator.PackageFlow(ctx, reader, out)
            default:
                return fmt.Errorf("unknown --action: %s (valid: command|file|service|package)", flags.Action)
            }
        },
    }

    addOperatorFlags(cmd, flags)
    return cmd
}

func addOperatorFlags(cmd *cobra.Command, flags *OperatorFlags) {
    f := cmd.Flags()
    f.StringVar(&flags.Action, "action", "command", "operator action: command|file|service|package")
}
