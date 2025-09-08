#!/bin/bash
# SPDX-License-Identifier: MIT
# Copyright 2025 Oppie Thunder Contributors
#
# Helios installer script for oppie.xyz

set -euo pipefail

# Configuration
REPO="good-night-oppie/helios"
BINARY_NAME="helios"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Detect platform
detect_platform() {
    local os
    local arch
    
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*)    os="windows" ;;
        MINGW*)     os="windows" ;;
        MSYS*)      os="windows" ;;
        *)          log_error "Unsupported operating system: $(uname -s)" && exit 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64)     arch="amd64" ;;
        amd64)      arch="amd64" ;;
        arm64)      arch="arm64" ;;
        aarch64)    arch="arm64" ;;
        *)          log_error "Unsupported architecture: $(uname -m)" && exit 1 ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')
    if [ -z "$version" ]; then
        log_error "Failed to get latest version"
        exit 1
    fi
    echo "$version"
}

# Download and install
install_helios() {
    local platform
    local version
    local download_url
    local archive_name
    local tmp_dir
    
    platform=$(detect_platform)
    version=$(get_latest_version)
    
    log_info "Installing Helios v${version} for ${platform}..."
    
    # Determine archive format
    if [[ "$platform" == "windows-"* ]]; then
        archive_name="${BINARY_NAME}-${version}-${platform}.zip"
    else
        archive_name="${BINARY_NAME}-${version}-${platform}.tar.gz"
    fi
    
    download_url="https://github.com/${REPO}/releases/download/v${version}/${archive_name}"
    
    # Create temporary directory
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    log_info "Downloading from: $download_url"
    
    # Download archive
    if ! curl -sSL "$download_url" -o "$tmp_dir/$archive_name"; then
        log_error "Failed to download Helios"
        exit 1
    fi
    
    # Extract archive
    cd "$tmp_dir"
    if [[ "$platform" == "windows-"* ]]; then
        if command -v unzip >/dev/null; then
            unzip -q "$archive_name"
        else
            log_error "unzip is required to install Helios on Windows"
            exit 1
        fi
        binary_name="${BINARY_NAME}-${platform}.exe"
    else
        tar -xzf "$archive_name"
        binary_name="${BINARY_NAME}-${platform}"
    fi
    
    # Make sure the binary exists
    if [ ! -f "$binary_name" ]; then
        log_error "Binary $binary_name not found in archive"
        exit 1
    fi
    
    # Make binary executable
    chmod +x "$binary_name"
    
    # Install binary
    log_info "Installing to $INSTALL_DIR/$BINARY_NAME"
    
    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        if command -v sudo >/dev/null; then
            sudo mv "$binary_name" "$INSTALL_DIR/$BINARY_NAME"
        else
            log_error "Cannot write to $INSTALL_DIR and sudo is not available"
            log_info "Try running: INSTALL_DIR=\$HOME/.local/bin $0"
            exit 1
        fi
    else
        mv "$binary_name" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    log_success "Helios v${version} installed successfully!"
    
    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null; then
        log_success "Helios is ready to use!"
        echo ""
        echo "🚀 Quick start:"
        echo "  helios init my-project"
        echo "  cd my-project" 
        echo "  echo 'print(\"hello\")' > test.py"
        echo "  helios commit --work ."
    else
        log_warning "Helios installed but not found in PATH"
        log_info "You may need to add $INSTALL_DIR to your PATH"
        log_info "Or run: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null; then
        missing_deps+=("curl")
    fi
    
    if ! command -v tar >/dev/null; then
        missing_deps+=("tar")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Please install them and try again"
        exit 1
    fi
}

# Main
main() {
    echo "🌟 Helios Installer"
    echo "Fast version control for AI agents"
    echo ""
    
    check_dependencies
    install_helios
    
    echo ""
    log_info "For technical details: https://github.com/${REPO}"
    log_info "For support: https://github.com/${REPO}/issues"
}

main "$@"