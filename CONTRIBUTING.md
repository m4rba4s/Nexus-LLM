# Contributing to GOLLM

Thank you for helping improve this CLI. Please read this short guide before opening a PR.

## Quick Start
- Fork and create a branch: `feat/<slug>` or `fix/<slug>`.
- Setup tools: `make setup` (installs golangci-lint, gosec, etc.).
- Dev loop: `make fmt lint test` (or `make quick`), then implement changes.
- Run locally: `go run ./cmd/gollm ...` or `make build && ./bin/gollm ...`.

### Sandboxed environments
- If your environment blocks opening local sockets (e.g., restricted CI), set `CI_SANDBOX=1` for tests: `CI_SANDBOX=1 make test-short`.
- This skips only listener-based tests while keeping normal behavior in full environments.

## Guidelines
- Follow the Repository Guidelines in [AGENTS.md](AGENTS.md): structure, style, testing, security, and PR rules.
- Keep diffs minimal and focused. Update docs/examples if behavior or config changes.
- Commit style: short, imperative subject (emoji prefixes allowed, see `git log`).

## Pull Requests
- Use the PR template (auto-applied): see [.github/PULL_REQUEST_TEMPLATE.md](.github/PULL_REQUEST_TEMPLATE.md).
- Before pushing, ensure:
  - `make fmt lint test coverage-check` passes (≥ 85% coverage)
  - `make security` is clean (gosec, govulncheck)
  - No secrets in code/logs; config via env/flags/`config.yaml`

## Reporting Issues
- Include steps to reproduce, expected vs. actual behavior, and environment details.
- Add logs/CLI output snippets when relevant (redact secrets).

Thanks for contributing!
