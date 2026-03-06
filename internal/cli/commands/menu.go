package commands

import (
    "context"
    "os"

    "github.com/spf13/cobra"
    "github.com/m4rba4s/Nexus-LLM/internal/menu"
)

type MenuFlags struct {
    Provider string
    Model    string
    Operator string
}

// NewMenuCommand creates the minimal menu command that exposes
// PC-Operator and Coder modes via a simple terminal UI.
func NewMenuCommand() *cobra.Command {
    flags := &MenuFlags{}
    cmd := &cobra.Command{
        Use:   "menu",
        Short: "Launch minimal menu (Operator + Coder)",
        Long: `Launches a minimal terminal menu with two focused modes:

1) PC-Operator — system operations (commands, files, services, packages)
2) Coder       — development tasks (codegen, refactor, tests, docs)

Use --provider/--model/--operator to preselect initial state.
Press 0/q/quit to exit.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Use command context if available, otherwise background
            ctx := cmd.Context()
            if ctx == nil || ctx == context.TODO() {
                ctx = context.Background()
            }
            return menu.RunWithOptions(ctx, os.Stdin, os.Stdout, menu.Options{
                Provider: flags.Provider,
                Model:    flags.Model,
                CoderOp:  flags.Operator,
            })
        },
    }

    addMenuFlags(cmd, flags)
    return cmd
}

func addMenuFlags(cmd *cobra.Command, flags *MenuFlags) {
    f := cmd.Flags()
    // Avoid shorthand conflicts with global flags (-p/-m) and output (-o)
    f.StringVar(&flags.Provider, "provider", "", "preselect provider (openai, anthropic, ...)")
    f.StringVar(&flags.Model, "model", "", "preselect model")
    f.StringVar(&flags.Operator, "operator", "", "preselect coder operator: codegen|refactor|test|docs")
}
