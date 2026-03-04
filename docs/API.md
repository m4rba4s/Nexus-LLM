# GOLLM API Documentation

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Authentication](#authentication)
- [Commands](#commands)
  - [Chat](#chat)
  - [Complete](#complete)
  - [Interactive](#interactive)
  - [Models](#models)
  - [Config](#config)
  - [Version](#version)
- [Configuration](#configuration)
- [Providers](#providers)
- [Global Flags](#global-flags)
- [Output Formats](#output-formats)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Advanced Usage](#advanced-usage)

## Overview

GOLLM provides a unified command-line interface for interacting with Large Language Models from various providers. The API is designed to be simple, consistent, and powerful.

### Key Features

- **Multi-Provider Support**: OpenAI, Anthropic, Ollama, and OpenAI-compatible APIs
- **Streaming Responses**: Real-time response streaming with progress indicators
- **Interactive Mode**: Chat-like interface for extended conversations
- **Flexible Configuration**: Multiple configuration methods with hierarchical precedence
- **Output Formats**: Support for text, JSON, YAML, and Markdown output
- **Security**: Enterprise-grade security with secure credential management

## Installation

### Quick Install

```bash
# macOS/Linux (Homebrew)
brew install yourusername/tap/gollm

# Linux/macOS (curl)
curl -fsSL https://raw.githubusercontent.com/yourusername/gollm/main/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/yourusername/gollm/main/install.ps1 | iex
```

### Build from Source

```bash
git clone https://github.com/yourusername/gollm.git
cd gollm
make build
./bin/gollm version
```

## Authentication

GOLLM supports multiple authentication methods:

### Environment Variables (Recommended)

```bash
export OPENAI_API_KEY="sk-your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export GOLLM_PROVIDERS_OLLAMA_BASE_URL="http://localhost:11434"
```

### Configuration File

```bash
gollm config set providers.openai.api_key "sk-your-key"
gollm config set providers.anthropic.api_key "your-key"
```

### Command Line (Not Recommended for Security)

```bash
gollm chat --provider openai --api-key "sk-your-key" "Hello"
```

## Commands

### Chat

Send a chat message to an LLM and receive a response.

#### Syntax

```bash
gollm chat [flags] <message>
```

#### Flags

| Flag | Short | Type | Description | Default |
|------|-------|------|-------------|---------|
| `--system` | `-s` | string | System prompt to set context | "" |
| `--stream` | | bool | Stream response in real-time | false |
| `--no-stream` | | bool | Disable streaming (override config) | false |
| `--max-tokens` | | int | Maximum tokens in response | provider default |
| `--temperature` | `-t` | float | Sampling temperature (0.0-2.0) | provider default |
| `--top-p` | | float | Nucleus sampling parameter | provider default |
| `--frequency-penalty` | | float | Frequency penalty (-2.0 to 2.0) | provider default |
| `--presence-penalty` | | float | Presence penalty (-2.0 to 2.0) | provider default |

#### Examples

```bash
# Basic chat
gollm chat "What is Go programming language?"

# With system prompt
gollm chat --system "You are a helpful coding assistant" "Write a Go function"

# Streaming response
gollm chat --stream "Tell me a story"

# With parameters
gollm chat --temperature 0.9 --max-tokens 1000 "Be creative"

# Pipe input
echo "Translate to French: Hello" | gollm chat

# From file
gollm chat "$(cat document.txt)"
```

#### Response Format

**Text Output (Default):**
```
The response from the LLM appears here as plain text.
```

**JSON Output:**
```json
{
  "response": "The response from the LLM",
  "model": "gpt-3.5-turbo",
  "provider": "openai",
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 25,
    "total_tokens": 35
  },
  "finish_reason": "stop"
}
```

### Complete

Generate code completions and implementations.

#### Syntax

```bash
gollm complete [flags] <prompt>
```

#### Flags

Core flags (in addition to chat):

- `--language` string: Programming language for context (auto-detected from `--input-file` extension if omitted)
- `--style` string: Completion style (`function`, `class`, `comment`, `fix`, `test`, `refactor`, `explain`; default `function`)
- `--multiple` int: Generate multiple options (1–5)
- `--input-file` string: Read code from file instead of argument
- `--output-file` string: Save completion to file
- `--stream` / `--no-stream`: Enable or disable streaming (default is non-streaming for code completion)
- `--show-context`: Include original code context in output
- `--show-explanation`: Include short explanation of the completion
- `--raw`: Output raw response without formatting/highlighting

#### Examples

```bash
# Code completion
gollm complete "def fibonacci(n):"

# From file
gollm complete --input-file main.go --style fix

# Specific language and style
gollm complete --language python --style class "class Calculator:"

# Multiple options
gollm complete --multiple 3 "def sort_array(arr):"

# Save to file
gollm complete --input-file handler.go --style test --output-file handler_test.go
```

### Interactive

Start an interactive chat session.

#### Syntax

```bash
gollm interactive [flags]
```

#### Features

- Multi-turn conversations with context
- Command shortcuts (`/help`, `/clear`, `/save`, `/quit`)
- History navigation with arrow keys
- Tab completion for commands
- Session management

#### Interactive Commands

| Command | Description |
|---------|-------------|
| `/help` | Show help message |
| `/clear` | Clear conversation history |
| `/save <file>` | Save conversation to file |
| `/load <file>` | Load conversation from file |
| `/model <name>` | Switch model |
| `/provider <name>` | Switch provider |
| `/system <prompt>` | Set system prompt |
| `/tokens` | Show token usage |
| `/quit` | Exit interactive mode |

#### Examples

```bash
# Start interactive session
gollm interactive

# With specific model
gollm interactive --model gpt-4

# With system prompt
gollm interactive --system "You are a coding assistant"
```

### Models

List and manage available models.

#### Subcommands

##### `models list`

List available models from providers.

```bash
gollm models list [flags]
```

**Flags:**
- `--provider, -p`: Filter by provider
- `--available`: Only show available models
- `--json`: Output in JSON format

**Examples:**
```bash
# List all models
gollm models list

# List OpenAI models only
gollm models list --provider openai

# JSON output
gollm models list --json
```

##### `models info`

Get detailed information about a specific model.

```bash
gollm models info [flags] <model-name>
```

**Examples:**
```bash
# Model information
gollm models info gpt-4

# With JSON output
gollm models info --json claude-3-sonnet
```

### Config

Manage GOLLM configuration.

#### Subcommands

##### `config init`

Initialize configuration with interactive setup.

```bash
gollm config init [flags]
```

##### `config get`

Get configuration values.

```bash
gollm config get [key]
```

**Examples:**
```bash
# Get all configuration
gollm config get

# Get specific value
gollm config get providers.openai.api_key
gollm config get default_provider
```

##### `config set`

Set configuration values.

```bash
gollm config set <key> <value> [flags]
```

**Flags:**
- `--secure`: Store value securely (for API keys)

**Examples:**
```bash
# Set provider
gollm config set default_provider openai

# Set API key securely
gollm config set --secure providers.openai.api_key "sk-your-key"

# Set global settings
gollm config set settings.max_tokens 2048
```

##### `config list`

List all configuration files and their locations.

```bash
gollm config list
```

##### `config validate`

Validate configuration files.

```bash
gollm config validate
```

### Version

Show version information.

#### Syntax

```bash
gollm version [flags]
```

#### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--short` | `-s` | Show short version only |
| `--detailed` | `-d` | Show detailed version information |

#### Examples

```bash
# Default version info
gollm version

# Short version
gollm version --short

# Detailed information
gollm version --detailed
```

## Configuration

GOLLM uses hierarchical configuration with the following precedence:

1. Command-line flags (highest priority)
2. Environment variables
3. Configuration file
4. Default values (lowest priority)

### Configuration File Locations

GOLLM searches for configuration files in the following order:

1. `./config.yaml` (current directory)
2. `~/.gollm/config.yaml` (user home)
3. `/etc/gollm/config.yaml` (system-wide)

### Configuration Schema

```yaml
# Default provider to use
default_provider: openai

# Provider configurations
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    organization: "your-org-id"  # optional
    
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    
  ollama:
    base_url: "http://localhost:11434"
    
  custom:
    type: "openai"  # OpenAI-compatible API
    api_key: "${CUSTOM_API_KEY}"
    base_url: "https://your-api.example.com/v1"

# Global settings
settings:
  max_tokens: 2048
  temperature: 0.7
  timeout: 300s
  stream: true
  
# Output preferences
output:
  format: "text"  # text, json, yaml, markdown
  color: true
  
# Feature flags
features:
  streaming: true
  caching: true
  plugins: true
```

### Environment Variables

All configuration values can be set via environment variables using the prefix `GOLLM_`:

```bash
# Provider settings
export GOLLM_DEFAULT_PROVIDER=openai
export GOLLM_PROVIDERS_OPENAI_API_KEY=sk-your-key
export GOLLM_PROVIDERS_ANTHROPIC_API_KEY=your-key

# Global settings
export GOLLM_SETTINGS_MAX_TOKENS=4096
export GOLLM_SETTINGS_TEMPERATURE=0.8
export GOLLM_SETTINGS_TIMEOUT=600s

# Output settings
export GOLLM_OUTPUT_FORMAT=json
export GOLLM_OUTPUT_COLOR=false
```

## Providers

### Supported Providers

#### OpenAI

```yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"  # optional
    organization: "your-org-id"  # optional
```

**Models:** `gpt-4`, `gpt-4-turbo`, `gpt-3.5-turbo`, `gpt-3.5-turbo-16k`

#### Anthropic

```yaml
providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"  # optional
```

**Models:** `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku`, `claude-2.1`, `claude-instant-1.2`

#### Ollama

```yaml
providers:
  ollama:
    base_url: "http://localhost:11434"
```

**Models:** Any model installed in Ollama (e.g., `llama2`, `codellama`, `mistral`)

#### OpenAI-Compatible APIs

```yaml
providers:
  custom:
    type: "openai"
    api_key: "${CUSTOM_API_KEY}"
    base_url: "https://your-api.example.com/v1"
```

### Provider-Specific Features

| Feature | OpenAI | Anthropic | Ollama |
|---------|--------|-----------|--------|
| Streaming | ✅ | ✅ | ✅ |
| System Messages | ✅ | ✅ | ✅ |
| Function Calling | ✅ | ✅ | ❌ |
| Vision | ✅ | ✅ | ✅ |
| JSON Mode | ✅ | ❌ | ✅ |

## Global Flags

These flags are available for all commands:

| Flag | Short | Type | Description | Default |
|------|-------|------|-------------|---------|
| `--config` | `-c` | string | Config file path | auto-search |
| `--provider` | `-p` | string | LLM provider to use | from config |
| `--model` | `-m` | string | Model to use | provider default |
| `--output` | `-o` | string | Output format (text, json, yaml, markdown) | text |
| `--log-level` | `-l` | string | Log level (debug, info, warn, error, fatal) | info |
| `--verbose` | `-v` | bool | Enable verbose output | false |
| `--quiet` | `-q` | bool | Suppress non-error output | false |
| `--no-color` | | bool | Disable colored output | false |
| `--timeout` | | duration | Request timeout | provider default |
| `--max-tokens` | | int | Maximum tokens in response | provider default |
| `--temperature` | `-t` | float | Sampling temperature (0.0-2.0) | provider default |

## Output Formats

GOLLM supports multiple output formats for different use cases:

### Text (Default)

Human-readable plain text output.

```bash
gollm chat "Hello" --output text
```

### JSON

Structured JSON output for programmatic use.

```bash
gollm chat "Hello" --output json
```

```json
{
  "response": "Hello! How can I help you today?",
  "model": "gpt-3.5-turbo",
  "provider": "openai",
  "usage": {
    "prompt_tokens": 2,
    "completion_tokens": 9,
    "total_tokens": 11
  },
  "finish_reason": "stop",
  "created": "2024-01-01T12:00:00Z"
}
```

### YAML

YAML format for configuration-friendly output.

```bash
gollm chat "Hello" --output yaml
```

```yaml
response: "Hello! How can I help you today?"
model: gpt-3.5-turbo
provider: openai
usage:
  prompt_tokens: 2
  completion_tokens: 9
  total_tokens: 11
finish_reason: stop
created: "2024-01-01T12:00:00Z"
```

### Markdown

Formatted markdown output for documentation.

```bash
gollm chat "Explain Go" --output markdown
```

## Error Handling

GOLLM provides detailed error messages and appropriate exit codes:

### Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Authentication error |
| 4 | Network error |
| 5 | Provider error |
| 6 | Invalid input |

### Error Format

Errors are output to stderr in a consistent format:

```
Error: <error-message>

Details:
  - <detail-1>
  - <detail-2>

Suggestions:
  - <suggestion-1>
  - <suggestion-2>

For more help: gollm <command> --help
```

### Common Errors

#### Authentication Errors

```bash
Error: authentication failed for provider 'openai'

Details:
  - Invalid API key or insufficient permissions
  - API key: sk-...abc123 (partially redacted)

Suggestions:
  - Check your API key: gollm config get providers.openai.api_key
  - Set a valid API key: gollm config set --secure providers.openai.api_key "sk-your-key"
  - Verify your OpenAI account has sufficient credits
```

#### Model Not Found

```bash
Error: model 'gpt-5' not found for provider 'openai'

Details:
  - Available models: gpt-4, gpt-4-turbo, gpt-3.5-turbo

Suggestions:
  - List available models: gollm models list --provider openai
  - Use a valid model: --model gpt-4
```

## Examples

### Basic Usage

```bash
# Simple chat
gollm chat "What is Kubernetes?"

# Code completion
gollm complete "func main() {"

# Interactive session
gollm interactive --model gpt-4
```

### Advanced Usage

```bash
# Multi-step workflow
gollm chat --system "You are a code reviewer" "$(cat code.go)" \
  --temperature 0.1 \
  --max-tokens 1000 \
  --output json > review.json

# Batch processing
for file in *.go; do
  echo "Reviewing $file..."
  gollm chat --system "Review this Go code" "$(cat $file)" \
    --provider openai \
    --model gpt-4 \
    --output markdown > "${file%.go}_review.md"
done

# Configuration management
gollm config init
gollm config set default_provider anthropic
gollm config set providers.anthropic.api_key "your-key"
gollm config validate
```

### Integration Examples

#### Shell Script Integration

```bash
#!/bin/bash

# Generate commit message
DIFF=$(git diff --cached)
if [ -n "$DIFF" ]; then
  COMMIT_MSG=$(echo "$DIFF" | gollm chat \
    --system "Generate a concise git commit message for this diff" \
    --max-tokens 100 \
    --temperature 0.1)
  echo "Suggested commit message: $COMMIT_MSG"
fi
```

#### CI/CD Integration

```yaml
# GitHub Actions
- name: Code Review with GOLLM
  run: |
    gollm chat \
      --system "You are a senior Go developer reviewing code for security and performance" \
      --provider openai \
      --model gpt-4 \
      --output json \
      "$(git diff HEAD~1)" > code_review.json
```

#### JSON Processing

```bash
# Extract specific fields
gollm chat "Explain Docker" --output json | jq '.response'

# Calculate costs
gollm chat "Hello world" --output json | jq '.usage.total_tokens'

# Chain with other tools
gollm chat "Generate Python code for sorting" --output json | \
  jq -r '.response' | \
  python3 -c "import sys; exec(sys.stdin.read())"
```

## Advanced Usage

### Environment-Specific Configurations

```bash
# Development environment
export GOLLM_DEFAULT_PROVIDER=ollama
export GOLLM_PROVIDERS_OLLAMA_BASE_URL=http://localhost:11434

# Production environment
export GOLLM_DEFAULT_PROVIDER=openai
export GOLLM_PROVIDERS_OPENAI_API_KEY=${PROD_OPENAI_KEY}
export GOLLM_SETTINGS_TIMEOUT=60s
```

### Custom Provider Configuration

```yaml
providers:
  azure_openai:
    type: "openai"
    base_url: "https://your-resource.openai.azure.com/openai/deployments/gpt-4/chat/completions"
    api_key: "${AZURE_OPENAI_KEY}"
    headers:
      api-version: "2023-12-01-preview"
      
  local_llm:
    type: "openai"
    base_url: "http://localhost:8080/v1"
    api_key: "not-needed"
```

### Performance Optimization

```bash
# Use faster models for quick tasks
gollm chat --model gpt-3.5-turbo "Quick question"

# Optimize token usage
gollm chat --max-tokens 100 --temperature 0.1 "Concise answer please"

# Batch similar requests
gollm chat --system "Translate to French" \
  "Hello\nGoodbye\nThank you" \
  --max-tokens 50
```

### Security Best Practices

```bash
# Use environment variables for API keys
export OPENAI_API_KEY="sk-your-key"

# Or use secure config storage
gollm config set --secure providers.openai.api_key

# Validate configuration
gollm config validate

# Check for exposed credentials
gollm config get --mask-secrets
```

### Monitoring and Logging

```bash
# Enable verbose logging
gollm chat "Hello" --verbose --log-level debug

# Monitor token usage
gollm chat "Explain AI" --output json | jq '.usage'

# Log to file
gollm chat "Hello" --verbose 2>debug.log
```

---

For more information and latest updates, visit:
- **Documentation**: https://docs.gollm.dev
- **GitHub**: https://github.com/yourusername/gollm
- **Issues**: https://github.com/yourusername/gollm/issues
