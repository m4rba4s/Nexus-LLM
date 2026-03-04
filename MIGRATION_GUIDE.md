# GOLLM Migration Guide: From CLI to Minimal Terminal Menu

## Overview

GOLLM has been transformed from a traditional CLI tool with multiple commands to a **minimal terminal menu** with two focused modes:

1. **PC-Operator** - System operations (commands, files, services, packages)
2. **Coder** - Development tasks (code generation, refactoring, tests, docs)

## What Changed

### Before (CLI Commands)
```bash
gollm chat "What is Go?"
gollm interactive --model gpt-4
gollm complete "def fibonacci(n):"
gollm tui --theme cyberpunk
```

### After (Terminal Menu)
```bash
gollm  # Launches menu directly
```

## New Architecture

```
╔═══════════════════════════════════════════╗
║            GOLLM OPERATOR                 ║
╚═══════════════════════════════════════════╝
│ Provider: openai      Model: gpt-3.5-turbo
│ Coder Op: Codegen
├───────────────────────────────────────────
│ 1) PC-Operator  - System operations
│ 2) Coder        - Development tasks
│ 3) Settings     - Configure provider/model
│ 0) Exit
└───────────────────────────────────────────
Select >
```

## Key Features

### PC-Operator Mode
- **Run command** - Execute shell commands with safety checks
- **Edit file** - Create/modify files with backup
- **Manage service** - Start/stop system services
- **Install package** - Package management with dry-run
- **Request sudo** - Transparent elevation handling

### Coder Mode
- **Codegen** - Generate new code from requirements
- **Refactor** - Improve existing code structure
- **Test** - Generate comprehensive unit tests
- **Docs** - Create/update documentation

### Settings
- Dynamic provider selection
- Model selection with API listing
- Coder operator selection
- Persistent configuration

## Configuration

Create `~/.gollm/config.yaml`:

```yaml
default_provider: openai
model: gpt-3.5-turbo

providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    default_model: gpt-3.5-turbo
  
  anthropic:
    api_key: ${ANTHROPIC_API_KEY}
    default_model: claude-3-sonnet-20240229

extra:
  coder_operator: Codegen
```

## Migration Steps

1. **Remove old CLI commands**
   ```bash
   # Old commands no longer work
   gollm chat "Hello"  # ❌
   ```

2. **Launch menu instead**
   ```bash
   gollm  # ✅
   ```

3. **Configure providers**
   - Copy `config.yaml.example` to `~/.gollm/config.yaml`
   - Set your API keys

4. **Use menu navigation**
   - Select mode with number keys (1-3, 0)
   - Follow prompts for operations
   - Settings persist between sessions

## Benefits

1. **Simpler UX** - No command memorization
2. **Focused workflows** - Two clear roles
3. **Safety first** - Confirmation for dangerous operations
4. **Minimal dependencies** - Pure Go, no heavy UI frameworks
5. **Fast startup** - Direct to menu, no command parsing

## Code Structure

```
cmd/gollm/
  main.go              # Minimal entry point

internal/
  menu/
    menu.go            # Main menu loop
  modes/
    operator/          # PC-Operator implementation
    coder/             # Coder implementation
  core/
    provider.go        # Provider interfaces
    global_registry.go # Singleton registry
```

## Removed Features

The following features were removed in favor of simplicity:

- Chat command (use Coder mode)
- Interactive chat (use menu)
- TUI themes (single minimal theme)
- Completion command (use Coder mode)
- Complex CLI flags (use Settings)

## API Keys

Set environment variables or add directly to config:

```bash
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...
export DEEPSEEK_API_KEY=...
```

## Troubleshooting

1. **No providers available**
   - Check config.yaml exists
   - Verify API keys are set

2. **Provider not configured**
   - Add provider section to config
   - Set api_key field

3. **Permission denied**
   - Use option 5 in PC-Operator for sudo
   - Run with appropriate privileges

## Philosophy

> "Do one thing well" - Unix Philosophy

GOLLM now focuses on two primary use cases:
- **System operations** through PC-Operator
- **Development tasks** through Coder

No chat interfaces, no complex UIs - just a minimal menu that gets work done.

---

For issues or questions, check the IMPLEMENTATION_DETAILS.md file.
