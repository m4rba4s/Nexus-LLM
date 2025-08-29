#!/bin/bash

# GOLLM Development Status Check
# Quick script to assess current development state

set -e

echo "🚀 GOLLM Development Status Check"
echo "=================================="
echo

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    echo "❌ Error: Run this script from the project root directory"
    exit 1
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check Go version
echo -e "${BLUE}📋 Environment Check${NC}"
echo "-------------------"
GO_VERSION=$(go version)
echo "Go Version: $GO_VERSION"

# Check if dependencies are up to date
echo "Checking dependencies..."
go mod tidy
echo "✅ Dependencies OK"
echo

# Build check
echo -e "${BLUE}🔨 Build Status${NC}"
echo "---------------"
if go build ./cmd/gollm > /dev/null 2>&1; then
    echo -e "✅ ${GREEN}Build: PASSING${NC}"
else
    echo -e "❌ ${RED}Build: FAILING${NC}"
    echo "Run 'go build ./cmd/gollm' to see errors"
fi
echo

# Current focus: Config package tests
echo -e "${BLUE}🧪 Config Package Tests (Current Focus)${NC}"
echo "--------------------------------------"

# Test the specific failing areas from checkpoint
echo "Testing environment variables..."
ENV_TEST=$(go test ./internal/config -run "TestLoad_EnvironmentVariables" -v 2>&1 | tail -1 | grep -o "PASS\|FAIL" || echo "ERROR")
if [[ "$ENV_TEST" == "PASS" ]]; then
    echo -e "✅ ${GREEN}Environment Variables: PASSING${NC}"
else
    echo -e "❌ ${RED}Environment Variables: FAILING${NC}"
fi

echo "Testing validation..."
VAL_TEST=$(go test ./internal/config -run "TestValidate" -v 2>&1 | tail -1 | grep -o "PASS\|FAIL" || echo "ERROR")
if [[ "$VAL_TEST" == "PASS" ]]; then
    echo -e "✅ ${GREEN}Validation Tests: PASSING${NC}"
else
    echo -e "❌ ${RED}Validation Tests: FAILING${NC}"
fi

echo "Testing file loading..."
FILE_TEST=$(go test ./internal/config -run "TestLoad_FromConfigFile" -v 2>&1 | tail -1 | grep -o "PASS\|FAIL" || echo "ERROR")
if [[ "$FILE_TEST" == "PASS" ]]; then
    echo -e "✅ ${GREEN}File Loading Tests: PASSING${NC}"
else
    echo -e "❌ ${RED}File Loading Tests: FAILING${NC}"
fi

# Overall config test status
echo
echo "Overall config package status..."
CONFIG_STATUS=$(go test ./internal/config 2>&1 | tail -1 | grep -o "PASS\|FAIL" || echo "ERROR")
if [[ "$CONFIG_STATUS" == "PASS" ]]; then
    echo -e "🎉 ${GREEN}Config Package: ALL TESTS PASSING${NC}"
else
    echo -e "⚠️  ${YELLOW}Config Package: SOME TESTS FAILING${NC}"
fi

# Test coverage for config
echo
echo "Generating coverage report..."
if go test ./internal/config -cover -coverprofile=config_coverage.out > /dev/null 2>&1; then
    COVERAGE=$(go tool cover -func=config_coverage.out | tail -1 | awk '{print $3}')
    echo "📊 Config Package Coverage: $COVERAGE"
    if [[ "${COVERAGE%.*}" -ge 90 ]]; then
        echo -e "✅ ${GREEN}Coverage Target: MET (≥90%)${NC}"
    else
        echo -e "⚠️  ${YELLOW}Coverage Target: NOT MET (<90%)${NC}"
    fi
else
    echo -e "❌ ${RED}Could not generate coverage report${NC}"
fi

echo

# Quick project structure check
echo -e "${BLUE}📁 Key Files Status${NC}"
echo "-------------------"
declare -a key_files=(
    "internal/config/config.go:Configuration Logic"
    "internal/config/config_test.go:Config Tests"
    "internal/core/types.go:Core Types"
    "DEVELOPMENT_CHECKPOINT.md:Progress Tracker"
    "RULEBOOK.md:Project Standards"
    "TASKS.md:Development Roadmap"
)

for file_info in "${key_files[@]}"; do
    IFS=':' read -r file desc <<< "$file_info"
    if [[ -f "$file" ]]; then
        echo -e "✅ ${GREEN}$desc${NC} ($file)"
    else
        echo -e "❌ ${RED}$desc${NC} ($file) - MISSING"
    fi
done

echo

# Next steps from checkpoint
echo -e "${BLUE}🎯 Next Steps (From Checkpoint)${NC}"
echo "-------------------------------"
echo "1. Fix SecureString environment variable deserialization"
echo "2. Complete config package test suite (90%+ coverage)"
echo "3. Resolve all validation test failures"
echo
echo "🧪 Quick test commands:"
echo "  go test -v ./internal/config -run 'TestLoad_EnvironmentVariables'"
echo "  go test -v ./internal/config -run 'TestValidate'"
echo "  go test -v ./internal/config"
echo
echo "📊 Coverage report:"
echo "  go test ./internal/config -cover -coverprofile=config_coverage.out"
echo "  go tool cover -html=config_coverage.out -o config_coverage.html"

# Development phase indicator
echo
echo -e "${BLUE}📍 Current Phase${NC}"
echo "----------------"
echo "Phase 5: Testing & Quality Assurance (Week 13-14)"
echo "Focus: Comprehensive Unit Testing - Config Package"
echo "Target: 90%+ test coverage, all edge cases handled"

echo
echo "🔄 Ready to continue development!"
echo "=================================="

# Cleanup
rm -f config_coverage.out 2>/dev/null || true
