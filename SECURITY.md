# Security Policy

## Reporting a Vulnerability
- Please do not open public issues for security reports.
- Email security reports to security@gollm.dev with details, reproduction steps, affected versions, and impact.
- You may alternatively use GitHub’s “Security” tab (Security Advisories) to open a private report.
- We aim to acknowledge within 48 hours and provide updates until resolution.

## Supported Versions
- Main branch (latest) and the most recent tagged release receive security fixes.
- Older releases may receive fixes at our discretion if impact is high and backporting is low risk.

## Disclosure Policy
- We follow responsible disclosure. We will coordinate a fix and publish an advisory with CVE (when applicable), credits, and remediation steps.

## Hardening & Guidance
- See docs/SECURITY.md for defense-in-depth, configuration, and operational best practices.
- Run `make security` (gosec, govulncheck) and keep dependencies up to date (`make deps`).

## Scope
- This policy covers the GOLLM CLI and its official packages under `internal/**`.
- Third-party providers, plugins, or forks are out of scope unless explicitly stated.
