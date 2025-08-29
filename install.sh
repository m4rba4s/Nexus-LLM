#!/bin/bash

# GOLLM Installation Script for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/yourusername/gollm/main/install.sh | sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
REPO="yourusername/gollm"
BINARY_NAME="gollm"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TEMP_DIR=""

# Create temporary directory
create_temp_dir() {
    TEMP_DIR=$(mktemp -d)
    trap cleanup EXIT
}

# Cleanup function
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Print functions
print_info() {
    printf "${BLUE}ℹ ${NC}%s\n" "$1"
}

print_success() {
    printf "${GREEN}✓ ${NC}%s\n" "$1"
}

print_warning() {
    printf "${YELLOW}⚠ ${NC}%s\n" "$1"
}

print_error() {
    printf "${RED}✗ ${NC}%s\n" "$1" >&2
}

print_header() {
    printf "\n${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    printf "${PURPLE}                        GOLLM Installation Script                         ${NC}\n"
    printf "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n\n"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)  os="linux" ;;
        Darwin*) os="darwin" ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            print_info "Supported: Linux, macOS"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l) arch="arm7" ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            print_info "Supported: amd64, arm64, arm7"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Get latest version from GitHub
get_latest_version() {
    if command_exists curl; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
    elif command_exists wget; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
    else
        print_error "Neither curl nor wget is available"
        print_info "Please install curl or wget to continue"
        exit 1
    fi
}

# Download file
download_file() {
    local url="$1"
    local output="$2"

    print_info "Downloading from: $url"

    if command_exists curl; then
        if ! curl -fsSL --progress-bar "$url" -o "$output"; then
            print_error "Failed to download with curl"
            return 1
        fi
    elif command_exists wget; then
        if ! wget -q --show-progress "$url" -O "$output"; then
            print_error "Failed to download with wget"
            return 1
        fi
    else
        print_error "Neither curl nor wget is available"
        return 1
    fi
}

# Verify checksum
verify_checksum() {
    local archive="$1"
    local checksum_file="$2"
    local checksum_url="$3"

    print_info "Verifying checksum..."

    # Download checksum file
    if ! download_file "$checksum_url" "$checksum_file"; then
        print_warning "Could not download checksum file, skipping verification"
        return 0
    fi

    # Verify checksum
    if command_exists sha256sum; then
        if (cd "$(dirname "$archive")" && sha256sum -c "$(basename "$checksum_file")" --quiet); then
            print_success "Checksum verification passed"
            return 0
        else
            print_error "Checksum verification failed"
            return 1
        fi
    elif command_exists shasum; then
        if (cd "$(dirname "$archive")" && shasum -a 256 -c "$(basename "$checksum_file")" --quiet); then
            print_success "Checksum verification passed"
            return 0
        else
            print_error "Checksum verification failed"
            return 1
        fi
    else
        print_warning "No checksum utility available (sha256sum/shasum), skipping verification"
        return 0
    fi
}

# Extract archive
extract_archive() {
    local archive="$1"
    local extract_dir="$2"

    print_info "Extracting archive..."

    if command_exists tar; then
        if tar -xzf "$archive" -C "$extract_dir"; then
            print_success "Archive extracted successfully"
            return 0
        else
            print_error "Failed to extract archive"
            return 1
        fi
    else
        print_error "tar command not found"
        return 1
    fi
}

# Install binary
install_binary() {
    local binary_path="$1"
    local install_path="$2"

    print_info "Installing binary to $install_path"

    # Check if install directory exists
    if [ ! -d "$(dirname "$install_path")" ]; then
        print_error "Install directory does not exist: $(dirname "$install_path")"
        print_info "Please create the directory or run with sudo"
        return 1
    fi

    # Check write permissions
    if [ ! -w "$(dirname "$install_path")" ]; then
        print_warning "No write permission to $(dirname "$install_path")"
        print_info "Attempting to install with sudo..."

        if ! sudo cp "$binary_path" "$install_path"; then
            print_error "Failed to install binary"
            return 1
        fi

        if ! sudo chmod 755 "$install_path"; then
            print_error "Failed to set permissions"
            return 1
        fi
    else
        if ! cp "$binary_path" "$install_path"; then
            print_error "Failed to install binary"
            return 1
        fi

        if ! chmod 755 "$install_path"; then
            print_error "Failed to set permissions"
            return 1
        fi
    fi

    print_success "Binary installed successfully"
    return 0
}

# Setup shell completion
setup_completion() {
    local binary_path="$1"

    print_info "Setting up shell completion..."

    # Try to detect shell
    local shell_name
    if [ -n "$BASH_VERSION" ]; then
        shell_name="bash"
    elif [ -n "$ZSH_VERSION" ]; then
        shell_name="zsh"
    elif [ -n "$FISH_VERSION" ]; then
        shell_name="fish"
    else
        shell_name=$(basename "$SHELL" 2>/dev/null || echo "unknown")
    fi

    case "$shell_name" in
        bash)
            local completion_dir
            # Try common completion directories
            for dir in /usr/local/share/bash-completion/completions /usr/share/bash-completion/completions /etc/bash_completion.d; do
                if [ -d "$dir" ]; then
                    completion_dir="$dir"
                    break
                fi
            done

            if [ -n "$completion_dir" ]; then
                if [ -w "$completion_dir" ] || sudo test -w "$completion_dir" 2>/dev/null; then
                    if "$binary_path" completion bash | sudo tee "$completion_dir/gollm" >/dev/null 2>&1 ||
                       "$binary_path" completion bash > "$completion_dir/gollm" 2>/dev/null; then
                        print_success "Bash completion installed"
                    else
                        print_warning "Failed to install bash completion"
                    fi
                fi
            fi
            ;;
        zsh)
            # Try to add to user's completion directory
            local zsh_comp_dir="${HOME}/.zsh/completions"
            if [ ! -d "$zsh_comp_dir" ]; then
                mkdir -p "$zsh_comp_dir" 2>/dev/null
            fi

            if [ -d "$zsh_comp_dir" ]; then
                if "$binary_path" completion zsh > "$zsh_comp_dir/_gollm" 2>/dev/null; then
                    print_success "Zsh completion installed"
                    print_info "Add 'fpath=(~/.zsh/completions \$fpath)' to your .zshrc if not already present"
                else
                    print_warning "Failed to install zsh completion"
                fi
            fi
            ;;
        fish)
            local fish_comp_dir="${HOME}/.config/fish/completions"
            if [ ! -d "$fish_comp_dir" ]; then
                mkdir -p "$fish_comp_dir" 2>/dev/null
            fi

            if [ -d "$fish_comp_dir" ]; then
                if "$binary_path" completion fish > "$fish_comp_dir/gollm.fish" 2>/dev/null; then
                    print_success "Fish completion installed"
                else
                    print_warning "Failed to install fish completion"
                fi
            fi
            ;;
        *)
            print_warning "Unknown shell: $shell_name, skipping completion setup"
            print_info "You can manually set up completion later with: gollm completion [bash|zsh|fish|powershell]"
            ;;
    esac
}

# Test installation
test_installation() {
    local binary_path="$1"

    print_info "Testing installation..."

    if ! command_exists "$BINARY_NAME"; then
        print_error "Binary not found in PATH"
        print_info "You may need to restart your shell or run: export PATH=\"$INSTALL_DIR:\$PATH\""
        return 1
    fi

    # Test version command
    local version
    if version=$("$binary_path" version --short 2>/dev/null); then
        print_success "Installation test passed"
        print_info "GOLLM version: $version"
        return 0
    else
        print_error "Installation test failed"
        return 1
    fi
}

# Print usage information
print_usage_info() {
    printf "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    printf "${CYAN}                              Getting Started                               ${NC}\n"
    printf "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n\n"

    printf "${GREEN}1. Initialize configuration:${NC}\n"
    printf "   gollm config init\n\n"

    printf "${GREEN}2. Set up your API keys:${NC}\n"
    printf "   gollm config set providers.openai.api_key \"sk-your-api-key\"\n"
    printf "   gollm config set providers.anthropic.api_key \"your-anthropic-key\"\n\n"

    printf "${GREEN}3. Test your first completion:${NC}\n"
    printf "   gollm chat \"Hello, world!\"\n\n"

    printf "${GREEN}4. Start interactive mode:${NC}\n"
    printf "   gollm interactive\n\n"

    printf "${GREEN}5. Get help:${NC}\n"
    printf "   gollm --help\n"
    printf "   gollm [command] --help\n\n"

    printf "${BLUE}Documentation:${NC} https://docs.gollm.dev\n"
    printf "${BLUE}GitHub:${NC}        https://github.com/${REPO}\n"
    printf "${BLUE}Issues:${NC}        https://github.com/${REPO}/issues\n\n"
}

# Main installation function
main() {
    print_header

    # Check prerequisites
    if ! command_exists curl && ! command_exists wget; then
        print_error "Neither curl nor wget is available"
        print_info "Please install curl or wget and try again"
        exit 1
    fi

    if ! command_exists tar; then
        print_error "tar command not found"
        print_info "Please install tar and try again"
        exit 1
    fi

    # Create temporary directory
    create_temp_dir

    # Detect platform
    print_info "Detecting platform..."
    local platform
    platform=$(detect_platform)
    print_success "Platform detected: $platform"

    # Get latest version
    print_info "Getting latest version..."
    local version
    version=$(get_latest_version)
    if [ -z "$version" ]; then
        print_error "Could not determine latest version"
        print_info "Please check your internet connection and try again"
        exit 1
    fi
    print_success "Latest version: $version"

    # Construct download URLs
    local version_no_v="${version#v}"
    local archive_name="gollm-${version_no_v}-${platform}.tar.gz"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
    local checksum_url="${download_url}.sha256"
    local archive_path="${TEMP_DIR}/${archive_name}"
    local checksum_path="${TEMP_DIR}/${archive_name}.sha256"

    # Download archive
    if ! download_file "$download_url" "$archive_path"; then
        print_error "Failed to download GOLLM"
        exit 1
    fi
    print_success "Download completed"

    # Verify checksum
    verify_checksum "$archive_path" "$checksum_path" "$checksum_url"

    # Extract archive
    if ! extract_archive "$archive_path" "$TEMP_DIR"; then
        exit 1
    fi

    # Find binary
    local binary_temp_path="${TEMP_DIR}/${BINARY_NAME}"
    if [ ! -f "$binary_temp_path" ]; then
        print_error "Binary not found in archive"
        exit 1
    fi

    # Make binary executable
    chmod +x "$binary_temp_path"

    # Install binary
    local install_path="${INSTALL_DIR}/${BINARY_NAME}"
    if ! install_binary "$binary_temp_path" "$install_path"; then
        exit 1
    fi

    # Setup shell completion
    setup_completion "$install_path"

    # Test installation
    if ! test_installation "$install_path"; then
        print_warning "Installation completed but tests failed"
        print_info "You may need to restart your shell or update your PATH"
    fi

    print_success "GOLLM installed successfully!"
    print_usage_info
}

# Parse command line arguments
while [ $# -gt 0 ]; do
    case $1 in
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            printf "GOLLM Installation Script\n\n"
            printf "Usage: %s [options]\n\n" "$0"
            printf "Options:\n"
            printf "  --install-dir DIR   Installation directory (default: /usr/local/bin)\n"
            printf "  --version VERSION   Specific version to install (default: latest)\n"
            printf "  -h, --help          Show this help message\n\n"
            printf "Environment Variables:\n"
            printf "  INSTALL_DIR         Installation directory\n"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            print_info "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main function
main "$@"
