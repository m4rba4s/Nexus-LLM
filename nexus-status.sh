#!/bin/bash

# ╔══════════════════════════════════════════════════════════════╗
# ║               🐉 NEXUS RED HYDRA STATUS SCANNER 🐉             ║
# ║                 Automated War Room Intelligence               ║
# ╚══════════════════════════════════════════════════════════════╝

set -u

# Color codes for cyber warfare aesthetics
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

# Icons for status indicators
SKULL="💀"
FIRE="🔥"
DRAGON="🐉"
TARGET="🎯"
SHIELD="🛡️"
LOCK="🔒"
BOLT="⚡"
ROCKET="🚀"
GEAR="🔧"
TEST="🧪"
CHECK="✅"
CROSS="❌"
WARN="⚠️"
INFO="ℹ️"

# Global status variables
OVERALL_STATUS=0
CRITICAL_COUNT=0
WARNING_COUNT=0
SUCCESS_COUNT=0

# ASCII Art Banner
print_banner() {
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
    echo -e "${WHITE}    NEXUS STATUS SCANNER v2.1.337${NC}"
    echo -e "${GRAY}    [CLASSIFIED] Automated Intelligence System${NC}"
    echo -e "${GRAY}    Timestamp: $(date '+%Y-%m-%d %H:%M:%S UTC')${NC}"
    echo
}

# Progress bar function
show_progress() {
    local current=$1
    local total=$2
    local message=$3
    local width=50
    local percentage=$((current * 100 / total))
    local filled=$((current * width / total))

    printf "\r${CYAN}[${NC}"
    for ((i=0; i<filled; i++)); do printf "▓"; done
    for ((i=filled; i<width; i++)); do printf "░"; done
    printf "${CYAN}] ${percentage}%% ${message}${NC}"
    if [ $current -eq $total ]; then
        echo
    fi
}

# Status reporting functions
report_success() {
    echo -e "${GREEN}${CHECK} $1${NC}"
    ((SUCCESS_COUNT++))
}

report_warning() {
    echo -e "${YELLOW}${WARN} $1${NC}"
    ((WARNING_COUNT++))
    OVERALL_STATUS=1
}

report_critical() {
    echo -e "${RED}${CROSS} $1${NC}"
    ((CRITICAL_COUNT++))
    OVERALL_STATUS=2
}

report_info() {
    echo -e "${BLUE}${INFO} $1${NC}"
}

# Check if we're in the right directory
check_project_directory() {
    echo -e "${WHITE}${GEAR} Validating NEXUS operational environment...${NC}"

    if [ ! -f "go.mod" ]; then
        report_critical "Not in Go project directory (go.mod missing)"
        return 1
    fi

    if [ ! -d "internal" ]; then
        report_warning "Internal directory structure missing"
    fi

    if [ ! -f "cmd/gollm/main.go" ]; then
        report_warning "Main entry point missing (cmd/gollm/main.go)"
    fi

    report_success "Project directory structure validated"
    return 0
}

# Check Git status
check_git_status() {
    echo -e "\n${WHITE}${DRAGON} Git Repository Intelligence${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if ! command -v git &> /dev/null; then
        report_warning "Git not available"
        return
    fi

    # Check if we're in a git repo
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        report_warning "Not a git repository"
        return
    fi

    # Current branch
    local branch=$(git branch --show-current 2>/dev/null || echo "detached")
    report_info "Current branch: ${CYAN}${branch}${NC}"

    # Check for uncommitted changes
    if git_status_output=$(git status --porcelain 2>/dev/null) && [ -n "$git_status_output" ]; then
        report_warning "Uncommitted changes detected"
        echo "$git_status_output" | head -10 | sed 's/^/    /'
    else
        report_success "Working directory clean"
    fi

    # Check for unpushed commits
    local unpushed=$(git log @{u}.. --oneline 2>/dev/null | wc -l || echo "0")
    if [ "$unpushed" -gt 0 ]; then
        report_info "Unpushed commits: ${unpushed}"
    fi

    # Latest commit
    local last_commit=$(git log -1 --pretty=format:"%h %s" 2>/dev/null || echo "N/A")
    report_info "Latest commit: ${last_commit}"
}

# Check build status
check_build_status() {
    echo -e "\n${WHITE}${GEAR} Build System Analysis${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Check Go version
    if command -v go &> /dev/null; then
        local go_version=$(go version | awk '{print $3}')
        report_info "Go version: ${go_version}"
    else
        report_critical "Go compiler not found"
        return
    fi

    # Try to build
    echo -e "${GRAY}    Building GOLLM...${NC}"
    if go build -o /tmp/gollm-test cmd/gollm/main.go 2>/dev/null; then
        report_success "Build successful"
        rm -f /tmp/gollm-test
    else
        report_critical "Build failed"
        go build -o /tmp/gollm-test cmd/gollm/main.go 2>&1 | head -10 | sed 's/^/    /' || true
    fi

    # Check dependencies
    if go mod tidy -v 2>/dev/null >/dev/null; then
        report_success "Dependencies up to date"
    else
        report_warning "Dependency issues detected"
    fi
}

# Check test status
check_test_status() {
    echo -e "\n${WHITE}${TEST} Combat Readiness Assessment${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Run tests with coverage
    echo -e "${GRAY}    Executing test suite...${NC}"
    if go test -short -cover ./... > /tmp/test_results.txt 2>&1; then
        local coverage=$(grep -E "coverage:" /tmp/test_results.txt | awk '{sum+=$NF} END {print sum/NR "%"}' | sed 's/%.*/%/' || echo "")
        if [ -n "$coverage" ]; then
            local coverage_num=$(echo "$coverage" | sed 's/%//')
            if command -v bc >/dev/null 2>&1 && [ -n "$coverage_num" ] && (( $(echo "$coverage_num >= 75" | bc -l 2>/dev/null || echo "0") )); then
                report_success "Test coverage: ${coverage} ${TARGET}"
            elif command -v bc >/dev/null 2>&1 && [ -n "$coverage_num" ] && (( $(echo "$coverage_num >= 50" | bc -l 2>/dev/null || echo "0") )); then
                report_warning "Test coverage: ${coverage} (target: 75%+)"
            else
                report_info "Test coverage: ${coverage}"
            fi
        else
            report_success "All tests pass"
        fi

        # Count test results
        local test_count=$(grep -E "^(PASS|FAIL|SKIP)" /tmp/test_results.txt | wc -l || echo "0")
        local pass_count=$(grep -c "^PASS" /tmp/test_results.txt || echo "0")
        report_info "Tests executed: ${pass_count}/${test_count} passed"

    else
        report_critical "Test suite failed"
        tail -10 /tmp/test_results.txt | sed 's/^/    /' || true
    fi

    rm -f /tmp/test_results.txt
}

# Check provider status
check_provider_status() {
    echo -e "\n${WHITE}${SHIELD} Provider Health Assessment${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Check if config file exists
    if [ -f "config.yaml" ]; then
        report_success "Configuration file found"

        # Count configured providers
        local provider_count=$(grep -E "^  [a-z]+:" config.yaml | wc -l)
        report_info "Configured providers: ${provider_count}"

        # Check for API keys (but don't expose them)
        local keys_count=$(grep -c "api_key:" config.yaml || echo "0")
        if [ "$keys_count" -gt 0 ]; then
            report_info "API keys configured: ${keys_count}"
        else
            report_warning "No API keys configured"
        fi

    else
        report_warning "No config.yaml found"
    fi

    # Quick provider test if available
    if [ -f "test_operators_quick.go" ]; then
        echo -e "${GRAY}    Running quick provider test...${NC}"
        if timeout 30s go run test_operators_quick.go > /tmp/provider_test.txt 2>&1; then
            local success_rate=$(grep -E "Success Rate:" /tmp/provider_test.txt | head -1 | awk '{print $3}' || echo "N/A")
            if [ "$success_rate" != "N/A" ]; then
                report_info "Provider success rate: ${success_rate}"
            fi
        else
            report_warning "Provider quick test failed or timed out"
        fi
        rm -f /tmp/provider_test.txt
    fi
}

# Check performance metrics
check_performance() {
    echo -e "\n${WHITE}${BOLT} Performance Analysis${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Binary size check
    if go build -o /tmp/gollm-size-test cmd/gollm/main.go 2>/dev/null; then
        local binary_size=$(du -h /tmp/gollm-size-test | awk '{print $1}')
        report_info "Binary size: ${binary_size}"
        rm -f /tmp/gollm-size-test
    fi

    # Startup time test
    if go build -o /tmp/gollm-perf-test cmd/gollm/main.go 2>/dev/null; then
        echo -e "${GRAY}    Measuring startup performance...${NC}"
        local startup_time=$(timeout 10s bash -c 'time (/tmp/gollm-perf-test --help > /dev/null 2>/dev/null)' 2>&1 | grep real | awk '{print $2}' || echo "")
        if [ -n "$startup_time" ]; then
            report_info "Startup time: ${startup_time}"
        fi
        rm -f /tmp/gollm-perf-test
    fi

    # Memory usage estimation
    if command -v go &> /dev/null; then
        echo -e "${GRAY}    Analyzing memory allocation patterns...${NC}"
        if go test -short -benchmem ./internal/config > /tmp/bench_results.txt 2>/dev/null; then
            local mem_allocs=$(grep -E "allocs/op" /tmp/bench_results.txt | head -1 | awk '{print $(NF-1)}' 2>/dev/null || echo "N/A")
            if [ "$mem_allocs" != "N/A" ]; then
                report_info "Memory allocations/op: ${mem_allocs}"
            fi
        fi
        rm -f /tmp/bench_results.txt
    fi
}

# Check security status
check_security() {
    echo -e "\n${WHITE}${LOCK} Security Posture Evaluation${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Check for exposed secrets in config
    if [ -f "config.yaml" ]; then
        if grep -qE "(sk-[a-zA-Z0-9]{48}|xoxb-[a-zA-Z0-9-]+)" config.yaml; then
            report_warning "Potential API keys exposed in config"
        else
            report_success "No exposed API keys in config"
        fi
    fi

    # Check for hardcoded secrets in code
    local secret_count=$(find . -name "*.go" -type f -exec grep -l "sk-\|api[_-]key\|secret\|password" {} \; 2>/dev/null | wc -l || echo "0")
    if [ "$secret_count" -gt 5 ]; then
        report_warning "Multiple files contain potential secrets (${secret_count} files)"
    else
        report_success "Minimal credential exposure risk"
    fi

    # Check file permissions
    local sensitive_files=("config.yaml" "*.key" "*.pem" ".env")
    for pattern in "${sensitive_files[@]}"; do
        while IFS= read -r -d '' file; do
            local perms=$(stat -c "%a" "$file" 2>/dev/null || echo "")
            if [ -n "$perms" ] && [ "$perms" -gt 600 ]; then
                report_warning "File ${file} has overly permissive permissions (${perms})"
            fi
        done < <(find . -name "$pattern" -print0 2>/dev/null)
    done

    # Check for goroutine leaks in tests
    if grep -r "runtime.NumGoroutine" . >/dev/null 2>&1; then
        report_success "Goroutine leak detection present"
    else
        report_info "Consider adding goroutine leak detection"
    fi
}

# Check documentation status
check_documentation() {
    echo -e "\n${WHITE}${INFO} Documentation Assessment${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    local docs=("README.md" "RULEBOOK.md" "SESSION_RULES.md" "TASKS.md")
    local doc_count=0

    for doc in "${docs[@]}"; do
        if [ -f "$doc" ]; then
            ((doc_count++))
            local size=$(wc -l < "$doc")
            report_success "${doc} (${size} lines)"
        else
            report_warning "${doc} missing"
        fi
    done

    report_info "Documentation files: ${doc_count}/${#docs[@]}"

    # Check for code comments
    local comment_ratio=$(find ./internal -name "*.go" -exec grep -c "//" {} \; 2>/dev/null | awk '{sum+=$1} END {print sum}' || echo "0")
    local code_lines=$(find ./internal -name "*.go" -exec wc -l {} \; 2>/dev/null | awk '{sum+=$1} END {print sum}' || echo "1")
    local ratio=0
    if [ "$code_lines" -gt 0 ]; then
        ratio=$((comment_ratio * 100 / code_lines))
    fi

    if [ "$ratio" -gt 15 ]; then
        report_success "Code comment ratio: ${ratio}%"
    else
        report_info "Code comment ratio: ${ratio}% (consider more documentation)"
    fi
}

# Generate final status report
generate_final_report() {
    echo
    echo -e "${WHITE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${WHITE}║                  ${DRAGON} NEXUS STATUS REPORT ${DRAGON}                  ║${NC}"
    echo -e "${WHITE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo

    # Status summary
    local total_checks=$((SUCCESS_COUNT + WARNING_COUNT + CRITICAL_COUNT))
    echo -e "${WHITE}📊 COMBAT READINESS SUMMARY${NC}"
    echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${GREEN}${CHECK} Success:  ${SUCCESS_COUNT}/${total_checks} operations${NC}"
    echo -e "${YELLOW}${WARN} Warnings: ${WARNING_COUNT}/${total_checks} operations${NC}"
    echo -e "${RED}${CROSS} Critical: ${CRITICAL_COUNT}/${total_checks} operations${NC}"
    echo

    # Overall status
    if [ $OVERALL_STATUS -eq 0 ]; then
        echo -e "${GREEN}${FIRE} SYSTEM STATUS: FULLY OPERATIONAL ${FIRE}${NC}"
        echo -e "${GREEN}${ROCKET} Ready for combat deployment${NC}"
    elif [ $OVERALL_STATUS -eq 1 ]; then
        echo -e "${YELLOW}${WARN} SYSTEM STATUS: CAUTION ADVISED${NC}"
        echo -e "${YELLOW}${TARGET} Some issues require attention${NC}"
    else
        echo -e "${RED}${SKULL} SYSTEM STATUS: CRITICAL ISSUES DETECTED${NC}"
        echo -e "${RED}${CROSS} Immediate intervention required${NC}"
    fi

    echo
    echo -e "${GRAY}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}${DRAGON} NEXUS RED HYDRA - STATUS SCAN COMPLETE${NC}"
    echo -e "${GRAY}Next scan recommended in 30 minutes${NC}"
    echo -e "${GRAY}For detailed analysis: ./nexus-status.sh --verbose${NC}"
    echo
}

# Main execution function
main() {
    print_banner

    # Check command line arguments
    local verbose=false
    if [[ "${1:-}" == "--verbose" ]]; then
        verbose=true
    fi

    # Initialize progress tracking
    local total_checks=7
    local current_check=0

    # Run all checks
    ((current_check++))
    show_progress $current_check $total_checks "Validating environment..."
    check_project_directory || true

    ((current_check++))
    show_progress $current_check $total_checks "Scanning Git repository..."
    check_git_status

    ((current_check++))
    show_progress $current_check $total_checks "Analyzing build system..."
    check_build_status

    ((current_check++))
    show_progress $current_check $total_checks "Testing combat readiness..."
    check_test_status

    ((current_check++))
    show_progress $current_check $total_checks "Assessing provider health..."
    check_provider_status

    ((current_check++))
    show_progress $current_check $total_checks "Measuring performance..."
    check_performance

    ((current_check++))
    show_progress $current_check $total_checks "Evaluating security posture..."
    check_security

    ((current_check++))
    show_progress $current_check $total_checks "Checking documentation..."
    check_documentation

    # Generate final report
    generate_final_report

    # Exit with appropriate code
    exit $OVERALL_STATUS
}

# Run the main function with all arguments
main "$@"
