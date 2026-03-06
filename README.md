# GOLLM — High‑Performance LLM CLI

[![CI](https://github.com/m4rba4s/Nexus-LLM/actions/workflows/ci.yml/badge.svg)](https://github.com/m4rba4s/Nexus-LLM/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/yourusername/gollm/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/gollm)

Nexus-LLM is a fast, reliable command‑line interface for Large Language Models with multiple providers, streaming, rich TUI, and strong defaults for security and portability.

- Docs: `docs/` (API, SECURITY, PERFORMANCE, ADRs)
- Source: `cmd/gollm` (entrypoint), `internal/**` (packages)
- Examples: `examples/`, Binaries: `bin/`

## Quick Start

Build locally:

```
make build
./bin/gollm version --short
```

Run a chat:

```
# Provide keys via env or config (see docs)
./bin/gollm chat "Hello"
```

Interactive mode:

```
./bin/gollm interactive --model gpt-4
```

## Development

- Tools: `make setup`
- Format/Lint/Test: `make fmt lint test`
- Quick loop: `make quick`
- Coverage report + threshold (85%): `make coverage-check`

If your environment blocks opening local sockets (e.g., restricted CI), run tests with:

```
CI_SANDBOX=1 make test-short
```

## Configuration

- Search order: `./config.yaml`, `~/.gollm/config.yaml`, `/etc/gollm/config.yaml`
- Override with env, e.g. `GOLLM_PROVIDERS_OPENAI_API_KEY=sk-...`

See `docs/API.md` and `config.yaml.example` for details.

## License

Apache-2.0 — see [LICENSE](LICENSE).

## Acknowledgments

- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Viper](https://github.com/spf13/viper) — configuration management
- [Go](https://golang.org/) team

---

**Made with ❤️ by the GOLLM / Nexus-LLM team**

