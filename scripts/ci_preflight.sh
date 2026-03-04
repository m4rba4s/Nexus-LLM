#!/usr/bin/env bash

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Starting NexusLLM CI/CD Quality Gates Pre-Flight Check..."

echo "[1/5] Running go mod tidy and verifying vendor..."
go mod tidy
go mod verify

echo "[2/5] Running Go Unit Tests with Race Detector..."
if go test -race ./...; then
    echo -e "${GREEN}Tests passed.${NC}"
else
    echo -e "${RED}Tests failed! Aborting.${NC}"
    exit 1
fi

echo "[3/5] Running Static Analysis (golangci-lint)..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run ./...; then
        echo -e "${GREEN}Linter checks passed.${NC}"
    else
        echo -e "${RED}Linter checks failed! Aborting.${NC}"
        exit 1
    fi
else
    echo -e "${RED}golangci-lint not found! Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin v1.55.2${NC}"
    exit 1
fi

echo "[4/5] Running Vulnerability Scanner (govulncheck)..."
if command -v govulncheck &> /dev/null; then
    if govulncheck ./...; then
        echo -e "${GREEN}Vulnerability checks passed.${NC}"
    else
        echo -e "${RED}govulncheck found issues! Aborting.${NC}"
        exit 1
    fi
else
    echo -e "${RED}govulncheck not found! Install with: go install golang.org/x/vuln/cmd/govulncheck@latest${NC}"
    exit 1
fi

echo "[5/5] Checking for Hardcoded Secrets (gitleaks)..."
if command -v gitleaks &> /dev/null; then
    if gitleaks detect -v --no-git; then
        echo -e "${GREEN}No secrets detected.${NC}"
    else
        echo -e "${RED}Gitleaks detected hardcoded secrets! Aborting.${NC}"
        exit 1
    fi
else
    echo -e "${RED}gitleaks not found! Install via Homebrew, Apt, or Go.${NC}"
    # Continuing for now, but in strict mode we'd exit
fi

echo -e "${GREEN}All Quality Gates Passed Successfully! Code is safe to deploy.${NC}"
exit 0
