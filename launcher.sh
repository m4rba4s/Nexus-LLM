#!/bin/bash

# GOLLM Launcher Script - Complete Edition
# Comprehensive launcher for all GOLLM modes and features

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# ASCII Art Logo
show_logo() {
    clear
    echo -e "${PURPLE}"
    cat << 'EOF'
    ╔═══════════════════════════════════════════════════════════════╗
    ║                                                               ║
    ║   ██████╗  ██████╗ ██╗     ██╗     ███╗   ███╗              ║
    ║  ██╔════╝ ██╔═══██╗██║     ██║     ████╗ ████║              ║
    ║  ██║  ███╗██║   ██║██║     ██║     ██╔████╔██║              ║
    ║  ██║   ██║██║   ██║██║     ██║     ██║╚██╔╝██║              ║
    ║  ╚██████╔╝╚██████╔╝███████╗███████╗██║ ╚═╝ ██║              ║
    ║   ╚═════╝  ╚═════╝ ╚══════╝╚══════╝╚═╝     ╚═╝              ║
    ║                                                               ║
    ║         🚀 Ultimate AI Terminal Interface Suite 🚀           ║
    ║                                                               ║
    ╚═══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# Check if gollm binary exists
check_binary() {
    if [ ! -f "./bin/gollm" ]; then
        echo -e "${YELLOW}Binary not found. Building GOLLM...${NC}"
        make build
        if [ $? -ne 0 ]; then
            echo -e "${RED}Build failed! Please check the error messages above.${NC}"
            exit 1
        fi
        echo -e "${GREEN}Build successful!${NC}"
        sleep 1
    fi
}

# Main menu
show_main_menu() {
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${WHITE}                    🎯 MAIN MENU 🎯                           ${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo
    echo -e "${GREEN}TERMINAL UI MODES:${NC}"
    echo -e "  ${YELLOW}1)${NC} 🎨 Ultimate Mode    - ${WHITE}Full-featured UI with all capabilities${NC}"
    echo -e "  ${YELLOW}2)${NC} 💬 TUI Mode        - ${WHITE}Beautiful chat interface${NC}"
    echo -e "  ${YELLOW}3)${NC} 🚀 Advanced Mode   - ${WHITE}Enhanced interactive mode with MCP${NC}"
    echo -e "  ${YELLOW}4)${NC} 🎮 Interactive Mode - ${WHITE}Enhanced interactive terminal${NC}"
    echo -e "  ${YELLOW}5)${NC} 📝 Simple Interactive - ${WHITE}Basic interactive mode${NC}"
    echo
    echo -e "${GREEN}COMMAND LINE MODES:${NC}"
    echo -e "  ${YELLOW}6)${NC} 💬 Chat            - ${WHITE}Direct chat completion${NC}"
    echo -e "  ${YELLOW}7)${NC} 🔧 Complete        - ${WHITE}Code completion${NC}"
    echo -e "  ${YELLOW}8)${NC} 📊 Benchmark       - ${WHITE}Performance testing${NC}"
    echo
    echo -e "${GREEN}CONFIGURATION:${NC}"
    echo -e "  ${YELLOW}9)${NC} ⚙️  Config          - ${WHITE}Manage configuration${NC}"
    echo -e "  ${YELLOW}10)${NC} 👤 Profile        - ${WHITE}Manage profiles${NC}"
    echo -e "  ${YELLOW}11)${NC} 📋 Models         - ${WHITE}List available models${NC}"
    echo
    echo -e "${GREEN}UTILITIES:${NC}"
    echo -e "  ${YELLOW}12)${NC} 📚 Help           - ${WHITE}Show help for any command${NC}"
    echo -e "  ${YELLOW}13)${NC} ℹ️  Version        - ${WHITE}Show version information${NC}"
    echo -e "  ${YELLOW}14)${NC} 🛠️  Custom Command - ${WHITE}Run custom gollm command${NC}"
    echo
    echo -e "  ${YELLOW}0)${NC} 🚪 Exit"
    echo
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
}

# Theme selection for UI modes
select_theme() {
    echo -e "${CYAN}Select a theme:${NC}"
    echo -e "  1) 🌈 Neon (default)"
    echo -e "  2) 🌙 Dark"
    echo -e "  3) ☀️  Light"
    echo -e "  4) 💻 Matrix"
    echo -e "  5) 🎮 Cyberpunk"
    echo -e "  6) 📝 Minimal"
    echo -e "  7) 🔲 Simple"
    echo
    read -p "Choice (1-7): " theme_choice
    
    case $theme_choice in
        1) echo "neon" ;;
        2) echo "dark" ;;
        3) echo "light" ;;
        4) echo "matrix" ;;
        5) echo "cyberpunk" ;;
        6) echo "minimal" ;;
        7) echo "simple" ;;
        *) echo "neon" ;;
    esac
}

# Provider selection
select_provider() {
    echo -e "${CYAN}Select AI Provider:${NC}"
    echo -e "  1) OpenAI"
    echo -e "  2) Anthropic (Claude)"
    echo -e "  3) Google (Gemini)"
    echo -e "  4) DeepSeek"
    echo -e "  5) OpenRouter"
    echo -e "  6) Ollama (local)"
    echo
    read -p "Choice (1-6): " provider_choice
    
    case $provider_choice in
        1) echo "openai" ;;
        2) echo "anthropic" ;;
        3) echo "gemini" ;;
        4) echo "deepseek" ;;
        5) echo "openrouter" ;;
        6) echo "ollama" ;;
        *) echo "" ;;
    esac
}

# Model selection based on provider
select_model() {
    local provider=$1
    
    echo -e "${CYAN}Select Model:${NC}"
    
    case $provider in
        openai)
            echo -e "  1) gpt-4-turbo-preview"
            echo -e "  2) gpt-4"
            echo -e "  3) gpt-3.5-turbo"
            echo -e "  4) gpt-3.5-turbo-16k"
            read -p "Choice (1-4): " model_choice
            case $model_choice in
                1) echo "gpt-4-turbo-preview" ;;
                2) echo "gpt-4" ;;
                3) echo "gpt-3.5-turbo" ;;
                4) echo "gpt-3.5-turbo-16k" ;;
                *) echo "gpt-3.5-turbo" ;;
            esac
            ;;
        anthropic)
            echo -e "  1) claude-3-opus-20240229"
            echo -e "  2) claude-3-sonnet-20240229"
            echo -e "  3) claude-3-haiku-20240307"
            echo -e "  4) claude-2.1"
            read -p "Choice (1-4): " model_choice
            case $model_choice in
                1) echo "claude-3-opus-20240229" ;;
                2) echo "claude-3-sonnet-20240229" ;;
                3) echo "claude-3-haiku-20240307" ;;
                4) echo "claude-2.1" ;;
                *) echo "claude-3-haiku-20240307" ;;
            esac
            ;;
        gemini)
            echo -e "  1) gemini-pro"
            echo -e "  2) gemini-pro-vision"
            read -p "Choice (1-2): " model_choice
            case $model_choice in
                1) echo "gemini-pro" ;;
                2) echo "gemini-pro-vision" ;;
                *) echo "gemini-pro" ;;
            esac
            ;;
        deepseek)
            echo -e "  1) deepseek-chat"
            echo -e "  2) deepseek-coder"
            read -p "Choice (1-2): " model_choice
            case $model_choice in
                1) echo "deepseek-chat" ;;
                2) echo "deepseek-coder" ;;
                *) echo "deepseek-chat" ;;
            esac
            ;;
        ollama)
            echo -e "  1) llama2"
            echo -e "  2) codellama"
            echo -e "  3) mistral"
            echo -e "  4) mixtral"
            echo -e "  5) Custom (enter name)"
            read -p "Choice (1-5): " model_choice
            case $model_choice in
                1) echo "llama2" ;;
                2) echo "codellama" ;;
                3) echo "mistral" ;;
                4) echo "mixtral" ;;
                5) 
                    read -p "Enter custom model name: " custom_model
                    echo "$custom_model"
                    ;;
                *) echo "llama2" ;;
            esac
            ;;
        *)
            echo ""
            ;;
    esac
}

# Launch Ultimate mode
launch_ultimate() {
	echo -e "${GREEN}🔥 Launching REAL Ultimate Mode with API! 🔥${NC}"
	echo -e "${YELLOW}Features:${NC}"
	echo -e "  • ${CYAN}25+ Latest 2024-2025 models${NC}"
	echo -e "  • ${CYAN}REAL API calls to OpenRouter${NC}"
	echo -e "  • ${CYAN}GPT-4o, Claude 3.5, Gemini 1.5, Qwen 2.5${NC}"
	echo -e "  • ${CYAN}Working chat with streaming${NC}"
	echo -e "  • ${CYAN}Quick setup with test key${NC}"
	echo
	echo -e "${GREEN}💎 Starting REAL Ultimate Mode...${NC}"
	sleep 1
	./bin/gollm ultimate-real
}

# Launch TUI mode
launch_tui() {
    echo -e "${GREEN}Launching TUI Mode...${NC}"
    local theme=$(select_theme)
    echo -e "${YELLOW}Starting with theme: $theme${NC}"
    sleep 1
    ./bin/gollm tui --theme "$theme"
}

# Launch Advanced mode
launch_advanced() {
    echo -e "${GREEN}Configuring Advanced Mode...${NC}"
    local theme=$(select_theme)
    
    echo
    read -p "Enable MCP server? (y/n): " enable_mcp
    local mcp_flags=""
    if [[ "$enable_mcp" == "y" ]]; then
        read -p "MCP server port (default 8080): " mcp_port
        mcp_port=${mcp_port:-8080}
        mcp_flags="--mcp-port $mcp_port"
        
        read -p "Enable auto-execute? (y/n): " auto_exec
        if [[ "$auto_exec" == "y" ]]; then
            mcp_flags="$mcp_flags --auto-execute"
        fi
    fi
    
    echo -e "${YELLOW}Launching Advanced Mode...${NC}"
    sleep 1
    ./bin/gollm advanced --theme "$theme" $mcp_flags
}

# Launch Interactive Enhanced mode
launch_interactive_enhanced() {
    echo -e "${GREEN}Launching Interactive Enhanced Mode...${NC}"
    local provider=$(select_provider)
    
    if [ -n "$provider" ]; then
        local model=$(select_model "$provider")
        echo -e "${YELLOW}Starting with provider: $provider, model: $model${NC}"
        sleep 1
        ./bin/gollm interactive --enhanced --provider "$provider" --model "$model"
    else
        ./bin/gollm interactive --enhanced
    fi
}

# Launch Simple Interactive mode
launch_interactive() {
    echo -e "${GREEN}Launching Simple Interactive Mode...${NC}"
    ./bin/gollm interactive
}

# Launch Chat mode
launch_chat() {
    echo -e "${GREEN}Configuring Chat Mode...${NC}"
    local provider=$(select_provider)
    
    if [ -n "$provider" ]; then
        local model=$(select_model "$provider")
        echo
        echo -e "${CYAN}Enter your message (or press Ctrl+D to cancel):${NC}"
        read -r -d '' message || true
        
        if [ -n "$message" ]; then
            echo -e "${YELLOW}Sending to $model...${NC}"
            echo "$message" | ./bin/gollm chat --provider "$provider" --model "$model"
        fi
    else
        echo -e "${CYAN}Enter your message:${NC}"
        read -r message
        echo "$message" | ./bin/gollm chat
    fi
}

# Launch Complete mode
launch_complete() {
    echo -e "${GREEN}Code Completion Mode${NC}"
    echo -e "${CYAN}Enter code to complete (press Ctrl+D when done):${NC}"
    read -r -d '' code || true
    
    if [ -n "$code" ]; then
        echo "$code" | ./bin/gollm complete
    fi
}

# Launch Benchmark
launch_benchmark() {
    echo -e "${GREEN}Launching Benchmark Mode...${NC}"
    local provider=$(select_provider)
    
    if [ -n "$provider" ]; then
        ./bin/gollm benchmark --provider "$provider"
    else
        ./bin/gollm benchmark
    fi
}

# Configuration management
manage_config() {
    echo -e "${GREEN}Configuration Management${NC}"
    echo -e "  1) Show current config"
    echo -e "  2) Set provider"
    echo -e "  3) Set model"
    echo -e "  4) Set API key"
    echo -e "  5) Edit config file"
    echo
    read -p "Choice (1-5): " config_choice
    
    case $config_choice in
        1)
            ./bin/gollm config show
            ;;
        2)
            local provider=$(select_provider)
            if [ -n "$provider" ]; then
                ./bin/gollm config set provider "$provider"
            fi
            ;;
        3)
            local provider=$(select_provider)
            if [ -n "$provider" ]; then
                local model=$(select_model "$provider")
                ./bin/gollm config set model "$model"
            fi
            ;;
        4)
            local provider=$(select_provider)
            if [ -n "$provider" ]; then
                echo -n "Enter API key for $provider: "
                read -s api_key
                echo
                ./bin/gollm config set "${provider}.api_key" "$api_key"
            fi
            ;;
        5)
            ${EDITOR:-nano} ~/.gollm/config.yaml
            ;;
    esac
}

# Profile management
manage_profiles() {
    echo -e "${GREEN}Profile Management${NC}"
    echo -e "  1) List profiles"
    echo -e "  2) Show current profile"
    echo -e "  3) Switch profile"
    echo -e "  4) Create profile"
    echo
    read -p "Choice (1-4): " profile_choice
    
    case $profile_choice in
        1)
            ./bin/gollm profile list
            ;;
        2)
            ./bin/gollm profile show
            ;;
        3)
            echo -n "Enter profile name: "
            read profile_name
            ./bin/gollm profile use "$profile_name"
            ;;
        4)
            echo -n "Enter new profile name: "
            read profile_name
            ./bin/gollm profile create "$profile_name"
            ;;
    esac
}

# List models
list_models() {
    echo -e "${GREEN}Listing Available Models...${NC}"
    local provider=$(select_provider)
    
    if [ -n "$provider" ]; then
        ./bin/gollm models --provider "$provider"
    else
        ./bin/gollm models
    fi
}

# Show help
show_help() {
    echo -e "${GREEN}GOLLM Help System${NC}"
    echo -e "  1) General help"
    echo -e "  2) Help for specific command"
    echo
    read -p "Choice (1-2): " help_choice
    
    case $help_choice in
        1)
            ./bin/gollm --help
            ;;
        2)
            echo -n "Enter command name (e.g., chat, tui, config): "
            read cmd_name
            ./bin/gollm "$cmd_name" --help
            ;;
    esac
}

# Custom command
run_custom() {
    echo -e "${GREEN}Custom Command${NC}"
    echo -e "${CYAN}Enter full gollm command (without './bin/gollm'):${NC}"
    read -r custom_cmd
    
    if [ -n "$custom_cmd" ]; then
        echo -e "${YELLOW}Running: ./bin/gollm $custom_cmd${NC}"
        ./bin/gollm $custom_cmd
    fi
}

# Main loop
main() {
    check_binary
    
    while true; do
        show_logo
        show_main_menu
        
        read -p "$(echo -e ${WHITE}Enter your choice: ${NC})" choice
        
        case $choice in
            1)
                launch_ultimate
                ;;
            2)
                launch_tui
                ;;
            3)
                launch_advanced
                ;;
            4)
                launch_interactive_enhanced
                ;;
            5)
                launch_interactive
                ;;
            6)
                launch_chat
                ;;
            7)
                launch_complete
                ;;
            8)
                launch_benchmark
                ;;
            9)
                manage_config
                ;;
            10)
                manage_profiles
                ;;
            11)
                list_models
                ;;
            12)
                show_help
                ;;
            13)
                ./bin/gollm version
                ;;
            14)
                run_custom
                ;;
            0)
                echo -e "${GREEN}Thank you for using GOLLM! Goodbye! 👋${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}Invalid choice. Please try again.${NC}"
                ;;
        esac
        
        echo
        echo -e "${CYAN}Press Enter to continue...${NC}"
        read
    done
}

# Run main function
main
