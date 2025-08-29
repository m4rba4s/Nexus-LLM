# GOLLM Makefile
# High-performance, cross-platform LLM CLI tool

# Project configuration
BINARY_NAME := gollm
PACKAGE := github.com/yourusername/gollm
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build configuration
BUILD_DIR := bin
DIST_DIR := dist
COVERAGE_DIR := coverage

# Go configuration
GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w \
	-X '$(PACKAGE)/internal/version.Version=$(VERSION)' \
	-X '$(PACKAGE)/internal/version.Commit=$(COMMIT)' \
	-X '$(PACKAGE)/internal/version.BuildTime=$(BUILD_TIME)'

# Cross-compilation targets
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

# Tools
GOLANGCI_LINT_VERSION := v1.55.2
GOSEC_VERSION := v2.18.2
NANCY_VERSION := v1.0.42

# Colors for output
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
RESET := \033[0m

.PHONY: help
help: ## Show this help message
	@echo "$(CYAN)GOLLM Build System$(RESET)"
	@echo
	@echo "$(GREEN)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(BLUE)%-20s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo
	@echo "$(GREEN)Environment variables:$(RESET)"
	@echo "  $(BLUE)VERSION$(RESET)     Version to build (default: git describe)"
	@echo "  $(BLUE)CGO_ENABLED$(RESET) Enable/disable CGO (default: 0)"
	@echo "  $(BLUE)RACE$(RESET)        Enable race detection (default: disabled)"

##@ Development

.PHONY: setup
setup: ## Set up development environment
	@echo "$(YELLOW)Setting up development environment...$(RESET)"
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@$(GO) install github.com/securecodewarrior/nancy@$(NANCY_VERSION)
	@$(GO) install github.com/securecodewarrior/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	@$(GO) install golang.org/x/tools/cmd/goimports@latest
	@$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	@$(GO) mod download
	@echo "$(GREEN)Development environment ready!$(RESET)"

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "$(YELLOW)Downloading dependencies...$(RESET)"
	@$(GO) mod download
	@$(GO) mod tidy
	@$(GO) mod verify

.PHONY: generate
generate: ## Generate code using go generate
	@echo "$(YELLOW)Generating code...$(RESET)"
	@$(GO) generate ./...

##@ Building

.PHONY: build
build: ## Build binary for current platform
	@echo "$(YELLOW)Building $(BINARY_NAME) for current platform...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gollm
	@echo "$(GREEN)Binary built: $(BUILD_DIR)/$(BINARY_NAME)$(RESET)"

.PHONY: build-debug
build-debug: ## Build binary with debug information
	@echo "$(YELLOW)Building debug binary...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -gcflags="all=-N -l" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-debug ./cmd/gollm
	@echo "$(GREEN)Debug binary built: $(BUILD_DIR)/$(BINARY_NAME)-debug$(RESET)"

.PHONY: build-race
build-race: ## Build binary with race detection
	@echo "$(YELLOW)Building binary with race detection...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=1 $(GO) build $(GOFLAGS) -race -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-race ./cmd/gollm
	@echo "$(GREEN)Race detection binary built: $(BUILD_DIR)/$(BINARY_NAME)-race$(RESET)"

.PHONY: build-all
build-all: ## Build binaries for all supported platforms
	@echo "$(YELLOW)Building for all platforms...$(RESET)"
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(MAKE) build-platform PLATFORM=$$platform; \
	done
	@echo "$(GREEN)All platform binaries built in $(DIST_DIR)/$(RESET)"

.PHONY: build-platform
build-platform: ## Build for specific platform (internal target)
	@echo "$(YELLOW)Building for $(PLATFORM)...$(RESET)"
	@GOOS=$(word 1,$(subst /, ,$(PLATFORM))) \
	 GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
	 CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
	 -o $(DIST_DIR)/$(BINARY_NAME)-$(subst /,-,$(PLATFORM))$(if $(findstring windows,$(PLATFORM)),.exe,) \
	 ./cmd/gollm

##@ Testing

.PHONY: test
test: ## Run unit tests
	@echo "$(YELLOW)Running unit tests...$(RESET)"
	@$(GO) test -v -race -timeout=5m ./...
	@echo "$(GREEN)Unit tests completed!$(RESET)"

.PHONY: test-short
test-short: ## Run unit tests (short mode)
	@echo "$(YELLOW)Running unit tests (short mode)...$(RESET)"
	@$(GO) test -short -race -timeout=2m ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(RESET)"
	@$(GO) test -tags=integration -v -timeout=10m ./test/integration/...
	@echo "$(GREEN)Integration tests completed!$(RESET)"

.PHONY: test-e2e
test-e2e: build ## Run end-to-end tests
	@echo "$(YELLOW)Running E2E tests...$(RESET)"
	@$(GO) test -tags=e2e -v -timeout=15m ./test/e2e/...
	@echo "$(GREEN)E2E tests completed!$(RESET)"

.PHONY: test-all
test-all: test test-integration test-e2e ## Run all tests

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(RESET)"
	@$(GO) test -bench=. -benchmem -run=^$$ ./...

.PHONY: benchmark-compare
benchmark-compare: ## Compare benchmarks (requires benchstat)
	@echo "$(YELLOW)Comparing benchmarks...$(RESET)"
	@$(GO) test -bench=. -benchmem -run=^$$ ./... -count=5 | tee benchmark.txt
	@echo "$(BLUE)Run 'benchstat benchmark.txt' to analyze results$(RESET)"

##@ Quality

.PHONY: fmt
fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(RESET)"
	@$(GO) fmt ./...
	@goimports -w .

.PHONY: lint
lint: ## Run linter
	@echo "$(YELLOW)Running linter...$(RESET)"
	@golangci-lint run --timeout=5m ./...
	@echo "$(GREEN)Linting completed!$(RESET)"

.PHONY: lint-fix
lint-fix: ## Run linter and fix issues
	@echo "$(YELLOW)Running linter with auto-fix...$(RESET)"
	@golangci-lint run --fix --timeout=5m ./...

.PHONY: vet
vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(RESET)"
	@$(GO) vet ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@echo "$(YELLOW)Running staticcheck...$(RESET)"
	@$(GO) run honnef.co/go/tools/cmd/staticcheck@latest ./...

##@ Security

.PHONY: security
security: security-scan security-vulns ## Run all security checks

.PHONY: security-scan
security-scan: ## Run security scanner
	@echo "$(YELLOW)Running security scan...$(RESET)"
	@gosec -fmt=json -out=security-report.json ./...
	@echo "$(GREEN)Security scan completed! Check security-report.json$(RESET)"

.PHONY: security-vulns
security-vulns: ## Check for known vulnerabilities
	@echo "$(YELLOW)Checking for vulnerabilities...$(RESET)"
	@govulncheck ./...
	@echo "$(GREEN)Vulnerability check completed!$(RESET)"

.PHONY: security-deps
security-deps: ## Audit dependencies for vulnerabilities
	@echo "$(YELLOW)Auditing dependencies...$(RESET)"
	@$(GO) list -json -deps ./... | nancy sleuth
	@echo "$(GREEN)Dependency audit completed!$(RESET)"

##@ Coverage

.PHONY: coverage
coverage: ## Generate test coverage report
	@echo "$(YELLOW)Generating coverage report...$(RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GO) test -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_DIR)/coverage.html$(RESET)"

.PHONY: coverage-check
coverage-check: coverage ## Check coverage threshold (85%)
	@echo "$(YELLOW)Checking coverage threshold...$(RESET)"
	@COVERAGE=$$($(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$${COVERAGE%.*}" -lt 85 ]; then \
		echo "$(RED)Coverage $${COVERAGE}% is below threshold (85%)$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)Coverage $${COVERAGE}% meets threshold$(RESET)"; \
	fi

##@ Documentation

.PHONY: docs
docs: ## Generate documentation
	@echo "$(YELLOW)Generating documentation...$(RESET)"
	@$(GO) doc -all ./...

.PHONY: godoc
godoc: ## Start godoc server
	@echo "$(YELLOW)Starting godoc server at http://localhost:6060$(RESET)"
	@$(GO) run golang.org/x/tools/cmd/godoc@latest -http=:6060

##@ Installation

.PHONY: install
install: build ## Install binary to GOPATH/bin
	@echo "$(YELLOW)Installing $(BINARY_NAME)...$(RESET)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(shell $(GO) env GOPATH)/bin/
	@echo "$(GREEN)$(BINARY_NAME) installed to $(shell $(GO) env GOPATH)/bin/$(RESET)"

.PHONY: install-local
install-local: build ## Install binary to /usr/local/bin
	@echo "$(YELLOW)Installing $(BINARY_NAME) to /usr/local/bin...$(RESET)"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)$(BINARY_NAME) installed to /usr/local/bin/$(RESET)"

.PHONY: uninstall
uninstall: ## Uninstall binary
	@echo "$(YELLOW)Uninstalling $(BINARY_NAME)...$(RESET)"
	@rm -f $(shell $(GO) env GOPATH)/bin/$(BINARY_NAME)
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)$(BINARY_NAME) uninstalled$(RESET)"

##@ Release

.PHONY: release-prepare
release-prepare: clean build-all test-all security coverage-check ## Prepare for release
	@echo "$(YELLOW)Preparing release $(VERSION)...$(RESET)"
	@mkdir -p $(DIST_DIR)/archives
	@for file in $(DIST_DIR)/$(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			platform=$${file#$(DIST_DIR)/$(BINARY_NAME)-}; \
			if [[ "$$file" == *".exe" ]]; then \
				platform=$${platform%.exe}; \
				zip -j "$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-$$platform.zip" "$$file"; \
			else \
				tar -czf "$(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-$$platform.tar.gz" -C $(DIST_DIR) "$${file##*/}"; \
			fi; \
		fi; \
	done
	@echo "$(GREEN)Release $(VERSION) prepared in $(DIST_DIR)/archives/$(RESET)"

.PHONY: release-checksums
release-checksums: ## Generate checksums for release files
	@echo "$(YELLOW)Generating checksums...$(RESET)"
	@cd $(DIST_DIR)/archives && sha256sum * > checksums.txt
	@echo "$(GREEN)Checksums generated: $(DIST_DIR)/archives/checksums.txt$(RESET)"

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(RESET)"
	@docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .
	@echo "$(GREEN)Docker image built: $(BINARY_NAME):$(VERSION)$(RESET)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "$(YELLOW)Running Docker container...$(RESET)"
	@docker run --rm -it $(BINARY_NAME):latest

##@ Cleanup

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(RESET)"
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	@rm -f security-report.json benchmark.txt
	@$(GO) clean -cache -testcache -modcache
	@echo "$(GREEN)Clean completed!$(RESET)"

.PHONY: clean-deps
clean-deps: ## Clean dependency cache
	@echo "$(YELLOW)Cleaning dependency cache...$(RESET)"
	@$(GO) clean -modcache
	@echo "$(GREEN)Dependency cache cleaned!$(RESET)"

##@ CI/CD

.PHONY: ci-setup
ci-setup: setup ## Setup CI environment
	@echo "$(GREEN)CI environment ready!$(RESET)"

.PHONY: ci-test
ci-test: fmt vet lint test coverage-check security ## Run CI test pipeline
	@echo "$(GREEN)CI test pipeline completed!$(RESET)"

.PHONY: ci-build
ci-build: ci-test build-all ## Run CI build pipeline
	@echo "$(GREEN)CI build pipeline completed!$(RESET)"

.PHONY: ci-release
ci-release: ci-build release-prepare release-checksums ## Run CI release pipeline
	@echo "$(GREEN)CI release pipeline completed!$(RESET)"

# Development shortcuts
.PHONY: dev
dev: fmt vet lint test ## Quick development cycle

.PHONY: quick
quick: fmt test-short ## Quick test cycle

# Default target
.DEFAULT_GOAL := help
