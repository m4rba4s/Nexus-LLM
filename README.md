# GOLLM — High‑Performance LLM CLI

[![CI](https://github.com/yourusername/gollm/actions/workflows/ci.yml/badge.svg)](https://github.com/yourusername/gollm/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/yourusername/gollm/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/gollm)

GOLLM is a fast, reliable command‑line interface for Large Language Models with multiple providers, streaming, rich TUI, and strong defaults for security and portability.

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

Apache-2.0 (or as defined by the repository)

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) for CLI framework
- [Viper](https://github.com/spf13/viper) for configuration management
- [Go](https://golang.org/) team for the amazing language
- All contributors and users of this project

## 📞 Support

- **Documentation**: [docs.gollm.dev](https://docs.gollm.dev)
- **Issues**: [GitHub Issues](https://github.com/yourusername/gollm/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/gollm/discussions)
- **Discord**: [Join our Discord](https://discord.gg/gollm)

---

**Made with ❤️ and ⚡ by the GOLLM team**

*GOLLM - Where performance meets simplicity in LLM interactions.*
=======
>>>>>>> debb6072 (Phase 1-26: Nexus-LLM Core Engine)
=======
>>>>>>> debb6072 (Phase 1-26: Nexus-LLM Core Engine)
=======
>>>>>>> debb6072 (Phase 1-26: Nexus-LLM Core Engine)
