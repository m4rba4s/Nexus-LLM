# Repository Guidelines

## Project Structure & Module Organization
- Source: `cmd/gollm` (entrypoint), `internal/**` (packages: `cli`, `core`, `providers`, `config`, `transport`, `tui`, `display`, `security`).
- Tests: co-located as `*_test.go` next to sources (see `internal/**`).
- Docs: `docs/` (API, SECURITY, PERFORMANCE, ADRs). Examples: `examples/`. Binaries: `bin/`.

## Build, Test, and Development Commands
- `make help` – list all targets.
- `make setup` – install dev tools (golangci-lint, goimports, gosec, etc.).
- `make build` – build `./cmd/gollm` to `bin/gollm`. Run with `./bin/gollm` or `go run ./cmd/gollm`.
- `make test` / `make quick` – full tests with race vs. fast loop.
- `make coverage` / `make coverage-check` – HTML report and enforce 85% total.
- `make fmt` `make lint` `make vet` – format and static analysis.

## Local Run
- `go run ./cmd/gollm chat "Hello"`
- `make build && ./bin/gollm interactive --model gpt-4`
- Override config with env: `GOLLM_PROVIDERS_OPENAI_API_KEY=sk-... ./bin/gollm models list`
- Config search: `./config.yaml`, `~/.gollm/config.yaml`, `/etc/gollm/config.yaml`.

## Coding Style & Naming Conventions
- Go style: gofmt/goimports (tabs, standard formatting). Run `make fmt`.
- Lint: `golangci-lint` (`make lint`). Fixes with `make lint-fix`.
- Packages are lower-case; filenames `snake_case.go`. Exported identifiers use PascalCase and have doc comments beginning with the name.
- CLI commands live in `internal/cli/commands` and follow `NewXCommand()` naming.
- Providers live in `internal/providers/<name>` and implement `core.Provider`.

## Testing Guidelines
- Framework: standard `go test` with race. Use table-driven tests.
- Names: files `*_test.go`; functions `TestXxx`, `BenchmarkXxx`, `ExampleXxx`.
- Run unit tests: `make test`; quick cycle: `make quick`.
- Coverage goal: ≥ 85% (`make coverage-check`). Add focused tests beside code.

## Commit & Pull Request Guidelines
- Commits: short, imperative subject; emoji prefixes acceptable (see `git log`).
- PRs: include clear description, linked issues, what/why, and test evidence (output or screenshots for CLI). Ensure `make fmt lint test coverage-check` passes and update docs/examples if CLI or config changes.

## PR Checklist
- Run: `make fmt lint test coverage-check` and `make security` locally.
- Add/adjust tests for new logic; keep coverage ≥ 85%.
- Update `docs/` and `examples/` if flags, output, or config change.
- No secrets in code or logs; verify config via `config.yaml.example`.

## Security & Configuration Tips
- Do not hardcode secrets. Configure via `config.yaml` (repo root, `~/.gollm/`, or `/etc/gollm/`) with env/flag overrides (Viper). See `config.yaml.example` and `docs/SECURITY.md`.
- Use `gosec` and `govulncheck` (`make security`). Avoid logging API keys; prefer `SecureString` in config.

## Low‑Level Architecture & Linux Portability
- Separation of concerns: keep `internal/core` pure Go (no OS calls), isolate IO/networking in `internal/transport`, and UI in `internal/tui`/`internal/display`.
- Interfaces at boundaries: expose small interfaces in `core` and depend on them from higher layers; implement providers in `internal/providers/<name>`.
- No CGO by default: target static builds with `CGO_ENABLED=0` (see Makefile). Use pure Go libs; add CGO only behind build tags if unavoidable.
- OS‑specific code via build tags: prefer `foo_linux.go`, `foo_darwin.go`, etc., instead of `runtime.GOOS` branches. Keep default implementation OS‑agnostic.
- Filesystem/paths: always use `filepath` (not string concatenation); avoid shelling out. Respect XDG paths on Linux; write under `~/.gollm` by default.
- Process and signals: use `context` for cancellation/timeouts; handle `os.Interrupt`/`SIGTERM` gracefully; never `os.Exit` from libraries.
- Syscall hygiene: avoid direct syscalls in hot paths; prefer `net/http`, `os/exec` only when necessary (and behind interfaces for testing).
- Error handling: wrap with context (`fmt.Errorf("…: %w", err)`); distinguish temporary vs fatal errors; do not panic in library code.
- Portability checks: keep `GOOS/GOARCH` matrix green (`make build-all`); verify `CGO_ENABLED=0` builds; run short tests on Linux (CI target).
- Performance: avoid allocations in tight loops, pre-size buffers, stream large outputs; keep provider/network timeouts sane and configurable.

## Linters & Static Analysis
- Use `golangci-lint` with repo config (`.golangci.yml`): run `make lint` (or `make lint-fix`) locally and in CI.
- Core tools: `go vet`, `staticcheck`, `errcheck`, `ineffassign`, `revive`, `gofumpt`. See Makefile targets: `make vet`, `make staticcheck`.
- Security scanners: `gosec`, `govulncheck`, dependency audit (`nancy`). Run `make security` (and `make security-deps`).
- Enforce coverage: `make coverage-check` (≥ 85%). Keep new code covered; add table‑driven tests next to code.

## Reliability & Failure Points
- Identify single points of failure (SPoF) per package; add retries with backoff where appropriate; use `internal/transport/circuit_breaker.go` for remote calls.
- Always pass `context.Context`; time‑bound network IO; surface precise errors (wrap lower‑level ones).
- Validate inputs at boundaries; keep types strict; prefer compile‑time checks over runtime reflection.
- Provide safe fallbacks (mock providers, safe‑mode UI) to keep UX responsive under provider failures.

## Architecture Overview
- CLI: Cobra-based root in `internal/cli`, subcommands in `internal/cli/commands` via `NewXCommand()`.
- Core: `internal/core` provides provider registry, types, validation, metrics.
- Providers: `internal/providers/<name>` implement `core.Provider`; registry creates/validates instances.
- Config: `internal/config` uses Viper + validation; sources: flags > env > file.
- UI/IO: `internal/tui` (themes/modes) and `internal/display` (formatting); HTTP in `internal/transport` with circuit breakers.

## Agent-Specific Instructions
- Keep diffs minimal and scoped; avoid unrelated refactors.
- Prefer Makefile tasks (`make fmt lint test coverage-check`) before changes land.
- When adding commands/providers, mirror existing patterns and add tests next to code.
- Update `docs/` and `examples/` when user-facing behavior or config changes.

## New Provider Steps
- Create `internal/providers/<name>` with a `NewFromConfig` and a type implementing `core.Provider` (+ optional streaming/model lister interfaces).
- Register in `init()` using `core.RegisterProviderFactory("<name>", NewFromConfig)`; import package anonymously where needed (see `internal/menu/menu.go`, TUI).
- Map config using `internal/config`’s `ProviderConfig` fields; support timeouts, retries, headers.
- Add unit tests for create/validate, completion, and error paths.
