#!/bin/bash

# GOLLM CLI Showcase Script
# Demonstrates the ASCII logos and key features

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Check if colors should be used
if [[ "${NO_COLOR}" != "" ]]; then
    RED=""
    GREEN=""
    YELLOW=""
    BLUE=""
    PURPLE=""
    CYAN=""
    WHITE=""
    NC=""
fi

# Function to print colored header
print_header() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}                    🔥 GOLLM SHOWCASE 🔥                      ${CYAN}║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo
}

# Function to print section header
print_section() {
    local title="$1"
    echo
    echo -e "${YELLOW}▓▓▓ $title ▓▓▓${NC}"
    echo -e "${YELLOW}$(printf '═%.0s' $(seq 1 $((${#title} + 8))))${NC}"
    echo
}

# Function to wait for user input
wait_for_enter() {
    echo -e "${PURPLE}Press Enter to continue...${NC}"
    read -r
    echo
}

# Function to run command with description
run_demo() {
    local description="$1"
    local command="$2"

    echo -e "${GREEN}➤ ${description}${NC}"
    echo -e "${BLUE}Command: ${WHITE}$command${NC}"
    echo

    if [[ "$3" == "no-wait" ]]; then
        eval "$command"
    else
        eval "$command"
        echo
        wait_for_enter
    fi
}

# Check if GOLLM is built
check_build() {
    if [[ ! -f "./gollm" ]]; then
        echo -e "${RED}❌ GOLLM binary not found!${NC}"
        echo -e "${YELLOW}Building GOLLM...${NC}"
        go build -o gollm cmd/gollm/main.go
        if [[ $? -eq 0 ]]; then
            echo -e "${GREEN}✅ GOLLM built successfully!${NC}"
        else
            echo -e "${RED}❌ Failed to build GOLLM${NC}"
            exit 1
        fi
        echo
    else
        echo -e "${GREEN}✅ GOLLM binary found${NC}"
        echo
    fi
}

# Main showcase function
main() {
    clear
    print_header

    echo -e "${WHITE}Welcome to the GOLLM CLI Showcase!${NC}"
    echo -e "${CYAN}This demo will show you the beautiful ASCII logos and key features.${NC}"
    echo
    wait_for_enter

    check_build

    # Demo 1: Logo Test
    print_section "🎨 ASCII LOGO DEMO"
    run_demo "Running logo test with all variations" "go run test_logo.go"

    # Demo 2: Version Command
    print_section "📋 VERSION COMMAND"
    run_demo "Show version with full ASCII logo" "./gollm version"

    # Demo 3: Help Command
    print_section "❓ HELP COMMAND"
    run_demo "Show help with logo and available commands" "./gollm --help"

    # Demo 4: Advanced Logo Demo
    print_section "🌈 ADVANCED LOGO SHOWCASE"
    echo -e "${CYAN}Running interactive logo demo with color schemes...${NC}"
    echo -e "${YELLOW}Note: Press Enter at each pause to continue${NC}"
    echo
    wait_for_enter
    FORCE_COLOR=1 go run demo/logo_demo.go

    # Demo 5: Configuration
    print_section "⚙️ CONFIGURATION FEATURES"
    run_demo "Show available configuration options" "./gollm config --help"

    # Demo 6: Profiles
    print_section "📁 PROFILE SYSTEM"
    run_demo "Show profile management commands" "./gollm profile --help"

    # Demo 7: Interactive Commands
    print_section "💬 INTERACTIVE MODES"
    echo -e "${GREEN}Available interactive modes:${NC}"
    echo -e "${WHITE}1. Basic Interactive:${NC} ./gollm interactive"
    echo -e "${WHITE}2. Enhanced Interactive:${NC} ./gollm interactive-enhanced"
    echo
    echo -e "${CYAN}These modes feature the compact logo on startup!${NC}"
    echo
    wait_for_enter

    # Demo 8: Benchmark
    print_section "📊 BENCHMARKING"
    run_demo "Show benchmark options for performance testing" "./gollm benchmark --help"

    # Demo 9: Build Info
    print_section "🏗️ BUILD INFORMATION"
    run_demo "Show detailed version information" "./gollm version --detailed"

    # Demo 10: All Commands
    print_section "🗂️ AVAILABLE COMMANDS"
    echo -e "${GREEN}Here are all available GOLLM commands:${NC}"
    echo
    ./gollm --help | grep -A 20 "Available Commands:"
    echo
    wait_for_enter

    # Final demo
    print_section "🎯 FINALE"
    echo -e "${WHITE}🎉 GOLLM CLI Showcase Complete! 🎉${NC}"
    echo
    echo -e "${CYAN}✨ Key Features Demonstrated:${NC}"
    echo -e "${WHITE}  🎨 Beautiful ASCII logos in multiple formats${NC}"
    echo -e "${WHITE}  ⚡ Lightning-fast performance (sub-100ms startup)${NC}"
    echo -e "${WHITE}  🔗 Multi-provider support${NC}"
    echo -e "${WHITE}  📋 Configuration profiles${NC}"
    echo -e "${WHITE}  💬 Enhanced interactive modes${NC}"
    echo -e "${WHITE}  📊 Performance benchmarking${NC}"
    echo -e "${WHITE}  🎯 Enterprise-ready features${NC}"
    echo
    echo -e "${GREEN}🚀 Ready to revolutionize your LLM workflow!${NC}"
    echo

    # Show final logo
    echo -e "${YELLOW}Final ASCII Logo Display:${NC}"
    echo -e "${CYAN}░▒▓███████▓▒░░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓███████▓▒░${NC}"
    echo -e "${BLUE}░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░${NC}"
    echo -e "${PURPLE}░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░${NC}"
    echo -e "${RED}░▒▓█▓▒░░▒▓█▓▒░▒▓██████▓▒░  ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░${NC}"
    echo -e "${PURPLE}░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░${NC}"
    echo -e "${BLUE}░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░      ░▒▓█▓▒░${NC}"
    echo -e "${CYAN}░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░░▒▓██████▓▒░░▒▓███████▓▒░${NC}"
    echo
    echo -e "${WHITE}            🚀 Lightning Fast • 🔗 Multi-Provider • 🎯 Enterprise Ready${NC}"
    echo
    echo -e "${GREEN}Thank you for exploring GOLLM!${NC}"
}

# Handle Ctrl+C
trap 'echo -e "\n${YELLOW}Showcase interrupted. Thanks for watching!${NC}"; exit 0' INT

# Check if we should run in quick mode
if [[ "$1" == "--quick" ]]; then
    echo -e "${YELLOW}Running in quick mode (no pauses)${NC}"
    wait_for_enter() { echo; }
fi

# Run the main showcase
main

echo
echo -e "${CYAN}🎯 Want to try GOLLM now?${NC}"
echo -e "${WHITE}  ./gollm version${NC}          # Show version with logo"
echo -e "${WHITE}  ./gollm --help${NC}          # Show help with logo  "
echo -e "${WHITE}  ./gollm interactive${NC}     # Start interactive mode"
echo -e "${WHITE}  ./gollm profile list${NC}    # Manage profiles"
echo
