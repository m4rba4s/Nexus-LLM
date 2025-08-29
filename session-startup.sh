#!/bin/bash

# ╔══════════════════════════════════════════════════════════════╗
# ║               🚀 NEXUS SESSION STARTUP SCANNER 🚀             ║
# ║                Quick Development Environment Check            ║
# ╚══════════════════════════════════════════════════════════════╝

# Color codes for professional output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

# Status indicators
CHECK="✅"
CROSS="❌"
WARN="⚠️"
ROCKET="🚀"
GEAR="🔧"
FIRE="🔥"
DRAGON="🐉"
TARGET="🎯"

clear

# Banner
echo -e "${CYAN}"
cat << 'EOF'
    ███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗
    ████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝
    ██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗
    ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║
    ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║
    ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝
EOF
echo -e "${NC}"
echo -e "${WHITE}    SESSION STARTUP SCANNER v1.0${NC}"
echo -e "${GRAY}    Quick Development Environment Check${NC}"
echo -e "${GRAY}    Timestamp: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo

# Quick status checks
echo -e "${WHITE}${ROCKET} RAPID STATUS CHECK${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 1. Git Status
echo -e "${GRAY}[1/5] Checking Git status...${NC}"
if git status &>/dev/null; then
    if [ -z "$(git status --porcelain)" ]; then
        echo -e "  ${CHECK} Git: Working directory clean"
    else
        echo -e "  ${WARN} Git: Uncommitted changes detected"
        git status --porcelain | head -3 | sed 's/^/    /'
    fi
else
    echo -e "  ${CROSS} Git: Not a git repository"
fi

# 2. Build Status
echo -e "${GRAY}[2/5] Verifying build...${NC}"
if go build -o /tmp/gollm-startup-test cmd/gollm/main.go 2>/dev/null; then
    echo -e "  ${CHECK} Build: Successful"
    rm -f /tmp/gollm-startup-test
else
    echo -e "  ${CROSS} Build: Failed"
fi

# 3. Core Tests
echo -e "${GRAY}[3/5] Running core tests...${NC}"
test_packages=(
    "./internal/config"
    "./internal/providers/mock"
    "./internal/transport"
)

test_results=""
all_passed=true
for pkg in "${test_packages[@]}"; do
    if go test -short "$pkg" &>/dev/null; then
        test_results="${test_results}${CHECK} "
    else
        test_results="${test_results}${CROSS} "
        all_passed=false
    fi
done

if $all_passed; then
    echo -e "  ${CHECK} Core Tests: All passing ${test_results}"
else
    echo -e "  ${WARN} Core Tests: Some issues ${test_results}"
fi

# 4. Dependencies
echo -e "${GRAY}[4/5] Checking dependencies...${NC}"
if go mod verify &>/dev/null; then
    echo -e "  ${CHECK} Dependencies: Verified"
else
    echo -e "  ${WARN} Dependencies: Issues detected"
fi

# 5. Performance Check
echo -e "${GRAY}[5/5] Performance baseline...${NC}"
if [ -f "./gollm" ] || go build -o /tmp/gollm-perf cmd/gollm/main.go 2>/dev/null; then
    binary_path="./gollm"
    [ ! -f "$binary_path" ] && binary_path="/tmp/gollm-perf"

    startup_time=$(time (timeout 5s "$binary_path" --help > /dev/null 2>&1) 2>&1 | grep real | awk '{print $2}' || echo "N/A")
    if [ "$startup_time" != "N/A" ]; then
        echo -e "  ${CHECK} Performance: Startup ${startup_time}"
    else
        echo -e "  ${TARGET} Performance: Ready for testing"
    fi
    rm -f /tmp/gollm-perf
else
    echo -e "  ${WARN} Performance: Build required"
fi

echo
echo -e "${WHITE}📊 QUICK SUMMARY${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Environment info
echo -e "${BLUE}Environment:${NC}"
echo -e "  Go Version: $(go version | awk '{print $3}')"
echo -e "  PWD: $(pwd | sed "s|$HOME|~|")"
echo -e "  Git Branch: $(git branch --show-current 2>/dev/null || echo 'N/A')"

echo
echo -e "${WHITE}⚡ DEVELOPMENT COMMANDS${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${YELLOW}Core Development:${NC}"
echo -e "  ${GEAR} Build:           ${GRAY}go build -o gollm cmd/gollm/main.go${NC}"
echo -e "  ${GEAR} Test Core:       ${GRAY}go test ./internal/config ./internal/providers/mock ./internal/transport -v${NC}"
echo -e "  ${GEAR} Test All:        ${GRAY}go test ./... -short${NC}"
echo -e "  ${GEAR} Run Binary:      ${GRAY}./gollm version${NC}"

echo
echo -e "${YELLOW}NEXUS Operations:${NC}"
echo -e "  ${DRAGON} Full Status:     ${GRAY}./nexus-status.sh${NC}"
echo -e "  ${DRAGON} Verbose Mode:    ${GRAY}./nexus-status.sh --verbose${NC}"
echo -e "  ${TARGET} Quick Build:     ${GRAY}make build${NC} (if Makefile exists)"

echo
echo -e "${YELLOW}Git Operations:${NC}"
echo -e "  ${GEAR} Status:          ${GRAY}git status${NC}"
echo -e "  ${GEAR} Commit:          ${GRAY}git add -A && git commit -m 'message'${NC}"
echo -e "  ${GEAR} History:         ${GRAY}git log --oneline -10${NC}"

echo
echo -e "${WHITE}🎯 RECOMMENDED NEXT STEPS${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Provide intelligent recommendations based on status
if ! $all_passed; then
    echo -e "1. ${CROSS} ${RED}Fix failing tests first:${NC} go test ./... -v"
fi

if [ ! -f "./gollm" ]; then
    echo -e "2. ${ROCKET} ${YELLOW}Build the binary:${NC} go build -o gollm cmd/gollm/main.go"
fi

if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    echo -e "3. ${WARN} ${YELLOW}Commit changes:${NC} Review and commit uncommitted files"
fi

echo -e "4. ${DRAGON} ${GREEN}Run full status:${NC} ./nexus-status.sh"
echo -e "5. ${TARGET} ${BLUE}Start development:${NC} Focus on test coverage improvements"

echo
echo -e "${GRAY}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}${FIRE} NEXUS SESSION READY - HAPPY CODING! ${FIRE}${NC}"
echo -e "${GRAY}For detailed analysis, run: ./nexus-status.sh${NC}"
echo

# Exit with appropriate code
if $all_passed && go build -o /tmp/test-build cmd/gollm/main.go 2>/dev/null; then
    rm -f /tmp/test-build
    exit 0
else
    rm -f /tmp/test-build 2>/dev/null
    exit 1
fi
