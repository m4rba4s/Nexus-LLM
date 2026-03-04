#!/bin/bash

# GOLLM Advanced Features Demo Script
# This script demonstrates the new advanced features of GOLLM CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_section() {
    echo -e "\n${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════${NC}\n"
}

print_command() {
    echo -e "${YELLOW}▶ Running:${NC} ${BLUE}$1${NC}"
}

print_info() {
    echo -e "${PURPLE}ℹ️  $1${NC}"
}

# ASCII Art Logo
clear
echo -e "${CYAN}"
cat << "EOF"
╔══════════════════════════════════════════════════════════════════════╗
║   ▄████████    ▄██████▄   ▄█        ▄█        ▄▄▄▄███▄▄▄▄           ║
║  ███    ███   ███    ███ ███       ███      ▄██▀▀▀███▀▀▀██▄         ║
║  ███    █▀    ███    ███ ███       ███      ███   ███   ███         ║
║  ███          ███    ███ ███       ███      ███   ███   ███         ║
║  ███    ▄███▄ ███    ███ ███       ███      ███   ███   ███         ║
║  ███    ███   ███    ███ ███       ███      ███   ███   ███         ║
║  ███    ███   ███    ███ ███▌    ▄ ███▌    ▄███   ███   ███         ║
║  ████████▀     ▀██████▀  █████▄▄██ █████▄▄██ ▀█   ███   █▀          ║
║                          ▀         ▀                                 ║
║              🚀 Advanced AI Terminal Interface Demo 🤖               ║
╚══════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${GREEN}Welcome to GOLLM Advanced Features Demo!${NC}"
echo -e "${PURPLE}This demo will showcase the new capabilities of GOLLM CLI${NC}\n"

# Check if gollm is built
if [ ! -f "./bin/gollm" ]; then
    print_info "Building GOLLM..."
    make build
fi

# Demo sections
sleep 2

print_section "1. BASIC COMMANDS"
print_command "./bin/gollm version"
./bin/gollm version
sleep 2

print_section "2. CONFIGURATION PROFILES"
print_command "./bin/gollm profile list"
./bin/gollm profile list || true
sleep 2

print_section "3. MODEL LISTING"
print_info "List available models (requires provider configuration)"
print_command "./bin/gollm models list --provider ollama"
# This might fail if not configured, so we use || true
./bin/gollm models list --provider ollama 2>/dev/null || echo "Provider not configured - this is expected in demo mode"
sleep 2

print_section "4. ADVANCED TUI FEATURES"
print_info "The advanced TUI includes:"
echo "  • 🎨 Beautiful animated UI with gradient effects"
echo "  • 🎯 Interactive menu system with hotkeys"
echo "  • 🔧 MCP server integration for extended capabilities"
echo "  • ⚡ Auto-execution mode for commands and code"
echo "  • 🌈 Multiple themes (Neon, Dark, Light, Matrix)"
echo "  • 📊 Real-time statistics and monitoring"
echo ""
print_info "To launch the advanced TUI, run:"
echo -e "${BLUE}    ./bin/gollm advanced${NC}"
echo -e "${BLUE}    ./bin/gollm advanced --theme matrix${NC}"
echo -e "${BLUE}    ./bin/gollm advanced --mcp-port 8080 --auto-execute${NC}"
sleep 3

print_section "5. MCP SERVER CAPABILITIES"
print_info "Model Context Protocol (MCP) server provides:"
echo "  • 📁 File Operations (read/write with security)"
echo "  • 🌐 HTTP Request capabilities"
echo "  • 💾 Database access"
echo "  • 🔍 Code analysis and search"
echo "  • 📊 Data processing (JSON/CSV)"
echo "  • ⚡ Command execution (with safeguards)"
sleep 3

print_section "6. SECURITY FEATURES"
print_info "Advanced security features include:"
echo "  • 🔒 Command whitelisting and blacklisting"
echo "  • ✅ User confirmation for dangerous operations"
echo "  • 🛡️ Path traversal protection"
echo "  • ⏱️ Rate limiting"
echo "  • 🔐 Authentication tokens for MCP"
sleep 3

print_section "7. INTERACTIVE MODES"
print_info "Multiple interactive modes available:"
echo ""
echo "Standard Interactive:"
echo -e "${BLUE}    ./bin/gollm interactive${NC}"
echo ""
echo "Enhanced Interactive:"
echo -e "${BLUE}    ./bin/gollm interactive-enhanced${NC}"
echo ""
echo "Cyberpunk TUI:"
echo -e "${BLUE}    ./bin/gollm tui${NC}"
echo ""
echo "Advanced TUI with MCP:"
echo -e "${BLUE}    ./bin/gollm advanced${NC}"
sleep 3

print_section "8. QUICK DEMO COMMANDS"
echo "Try these commands to explore GOLLM:"
echo ""
echo -e "${BLUE}# Show help${NC}"
echo "    ./bin/gollm --help"
echo ""
echo -e "${BLUE}# Launch advanced interface${NC}"
echo "    ./bin/gollm advanced"
echo ""
echo -e "${BLUE}# Launch with Matrix theme${NC}"
echo "    ./bin/gollm advanced --theme matrix"
echo ""
echo -e "${BLUE}# Start MCP server and auto-execution${NC}"
echo "    ./bin/gollm advanced --mcp-port 8080 --auto-execute"
echo ""

print_section "DEMO COMPLETE!"
echo -e "${GREEN}✨ GOLLM is ready to use!${NC}"
echo -e "${PURPLE}Enjoy the advanced AI terminal experience!${NC}"
echo ""
echo -e "${CYAN}Press Ctrl+C to exit any interactive mode${NC}"
echo -e "${CYAN}Press Ctrl+M to open the menu in advanced mode${NC}"
echo ""
