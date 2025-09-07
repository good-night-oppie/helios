#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: 2025 Good Night Oppie

# Interactive installation and onboarding script for Monitor Daemon
# Supports multi-project configuration and PM2 autostart

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
GLOBAL_CONFIG_DIR="$HOME/.monitor-daemon"
GLOBAL_CONFIG_FILE="$GLOBAL_CONFIG_DIR/config.json"
PROJECTS_DIR="$GLOBAL_CONFIG_DIR/projects"
DAEMON_BIN="/usr/local/bin/monitor-daemon"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Functions
print_header() {
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║        Monitor Daemon - Multi-Project Installation          ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo
}

log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*"
}

success() {
    echo -e "${GREEN}✓${NC} $*"
}

error() {
    echo -e "${RED}✗${NC} $*" >&2
}

warning() {
    echo -e "${YELLOW}⚠${NC} $*"
}

prompt() {
    local prompt_text=$1
    local var_name=$2
    local default_value=${3:-}
    
    if [ -n "$default_value" ]; then
        echo -ne "${MAGENTA}?${NC} $prompt_text ${YELLOW}[$default_value]${NC}: "
    else
        echo -ne "${MAGENTA}?${NC} $prompt_text: "
    fi
    
    read -r input
    if [ -z "$input" ] && [ -n "$default_value" ]; then
        eval "$var_name='$default_value'"
    else
        eval "$var_name='$input'"
    fi
}

prompt_yes_no() {
    local prompt_text=$1
    local default=${2:-y}
    
    if [ "$default" = "y" ]; then
        echo -ne "${MAGENTA}?${NC} $prompt_text ${YELLOW}[Y/n]${NC}: "
    else
        echo -ne "${MAGENTA}?${NC} $prompt_text ${YELLOW}[y/N]${NC}: "
    fi
    
    read -r response
    response=${response:-$default}
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}

check_dependencies() {
    log "Checking dependencies..."
    
    local missing_deps=()
    
    # Check for Go
    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi
    
    # Check for PM2
    if ! command -v pm2 &> /dev/null; then
        missing_deps+=("pm2")
    fi
    
    # Check for jq
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        error "Missing dependencies: ${missing_deps[*]}"
        echo
        echo "Installation instructions:"
        
        for dep in "${missing_deps[@]}"; do
            case $dep in
                go)
                    echo "  Go: https://golang.org/doc/install"
                    ;;
                pm2)
                    echo "  PM2: npm install -g pm2"
                    ;;
                jq)
                    echo "  jq: sudo apt-get install jq (Ubuntu) or brew install jq (macOS)"
                    ;;
            esac
        done
        
        return 1
    fi
    
    success "All dependencies satisfied"
    return 0
}

create_directories() {
    log "Creating directory structure..."
    
    mkdir -p "$GLOBAL_CONFIG_DIR"
    mkdir -p "$PROJECTS_DIR"
    mkdir -p "$GLOBAL_CONFIG_DIR/logs"
    mkdir -p "$GLOBAL_CONFIG_DIR/state"
    mkdir -p "$GLOBAL_CONFIG_DIR/cache"
    
    success "Directory structure created"
}

build_daemon() {
    log "Building monitor daemon..."
    
    cd "$SCRIPT_DIR"
    
    if [ ! -f "go.mod" ]; then
        error "go.mod not found in $SCRIPT_DIR"
        return 1
    fi
    
    go mod download
    go build -o monitor-daemon main.go
    
    if [ ! -f "monitor-daemon" ]; then
        error "Failed to build daemon"
        return 1
    fi
    
    success "Daemon built successfully"
}

install_binary() {
    log "Installing daemon binary..."
    
    if [ -f "$DAEMON_BIN" ]; then
        warning "Daemon already installed at $DAEMON_BIN"
        if prompt_yes_no "Overwrite existing installation?" y; then
            sudo cp "$SCRIPT_DIR/monitor-daemon" "$DAEMON_BIN"
            sudo chmod 755 "$DAEMON_BIN"
            success "Daemon binary updated"
        fi
    else
        sudo cp "$SCRIPT_DIR/monitor-daemon" "$DAEMON_BIN"
        sudo chmod 755 "$DAEMON_BIN"
        success "Daemon binary installed to $DAEMON_BIN"
    fi
}

configure_project() {
    echo
    echo -e "${CYAN}=== Project Configuration ===${NC}"
    echo
    
    # Get project details
    prompt "Project name" project_name
    prompt "GitHub owner" github_owner "good-night-oppie"
    prompt "GitHub repository" github_repo
    prompt "GitHub token (will be stored securely)" github_token
    prompt "Check interval (e.g., 2m, 5m, 10m)" check_interval "2m"
    
    # Feature flags
    echo
    echo -e "${CYAN}Features:${NC}"
    prompt_yes_no "Enable PR monitoring?" y && enable_pr="true" || enable_pr="false"
    prompt_yes_no "Enable CI monitoring?" y && enable_ci="true" || enable_ci="false"
    prompt_yes_no "Enable debate management?" y && enable_debate="true" || enable_debate="false"
    prompt_yes_no "Enable debug mode?" n && debug_mode="true" || debug_mode="false"
    
    # Create project config
    local project_config_file="$PROJECTS_DIR/${project_name}.json"
    
    cat > "$project_config_file" << EOF
{
  "project_name": "$project_name",
  "github_token": "$github_token",
  "owner": "$github_owner",
  "repo": "$github_repo",
  "check_interval": "$check_interval",
  "cache_dir": "$GLOBAL_CONFIG_DIR/cache/$project_name",
  "state_dir": "$GLOBAL_CONFIG_DIR/state/$project_name",
  "log_file": "$GLOBAL_CONFIG_DIR/logs/${project_name}.log",
  "max_retries": 3,
  "enable_pr_monitor": $enable_pr,
  "enable_ci_monitor": $enable_ci,
  "enable_debate": $enable_debate,
  "debug_mode": $debug_mode,
  "features": {
    "auto_fix_ci": true,
    "auto_respond_reviews": true,
    "track_complexity": true,
    "generate_evidence": true
  },
  "thresholds": {
    "debate_rounds_max": 3,
    "complexity_threshold": 7,
    "approval_keywords": ["APPROVED", "LGTM", "READY FOR MERGE"],
    "critical_keywords": ["CRITICAL", "SECURITY", "BUG", "ERROR"]
  }
}
EOF
    
    # Create project directories
    mkdir -p "$GLOBAL_CONFIG_DIR/cache/$project_name"
    mkdir -p "$GLOBAL_CONFIG_DIR/state/$project_name"
    
    success "Project '$project_name' configured"
    
    # Add to global config
    add_project_to_global "$project_name" "$project_config_file"
}

add_project_to_global() {
    local project_name=$1
    local config_file=$2
    
    # Create or update global config
    if [ ! -f "$GLOBAL_CONFIG_FILE" ]; then
        cat > "$GLOBAL_CONFIG_FILE" << EOF
{
  "version": "1.0.0",
  "projects": {
    "$project_name": "$config_file"
  },
  "default_project": "$project_name",
  "pm2_process_name": "monitor-daemon",
  "global_settings": {
    "max_concurrent_monitors": 10,
    "log_retention_days": 30
  }
}
EOF
    else
        # Update existing config
        jq ".projects[\"$project_name\"] = \"$config_file\"" "$GLOBAL_CONFIG_FILE" > "$GLOBAL_CONFIG_FILE.tmp"
        mv "$GLOBAL_CONFIG_FILE.tmp" "$GLOBAL_CONFIG_FILE"
    fi
}

setup_pm2() {
    echo
    echo -e "${CYAN}=== PM2 Configuration ===${NC}"
    echo
    
    log "Setting up PM2 process..."
    
    # Create PM2 ecosystem file
    cat > "$GLOBAL_CONFIG_DIR/ecosystem.config.js" << 'EOF'
module.exports = {
  apps: [{
    name: 'monitor-daemon',
    script: '/usr/local/bin/monitor-daemon',
    args: '-config ' + process.env.HOME + '/.monitor-daemon/config.json',
    instances: 1,
    exec_mode: 'fork',
    autorestart: true,
    watch: false,
    max_memory_restart: '100M',
    env: {
      NODE_ENV: 'production',
      DAEMON_MODE: 'multi-project'
    },
    error_file: process.env.HOME + '/.monitor-daemon/logs/pm2-error.log',
    out_file: process.env.HOME + '/.monitor-daemon/logs/pm2-out.log',
    log_file: process.env.HOME + '/.monitor-daemon/logs/pm2-combined.log',
    time: true,
    merge_logs: true,
    min_uptime: '10s',
    max_restarts: 10,
    restart_delay: 4000,
    kill_timeout: 3000
  }]
};
EOF
    
    # Start with PM2
    cd "$GLOBAL_CONFIG_DIR"
    pm2 start ecosystem.config.js
    
    success "PM2 process started"
    
    # Setup autostart
    if prompt_yes_no "Enable PM2 autostart on system boot?" y; then
        pm2 startup systemd -u "$USER" --hp "$HOME" | tail -n 1 | bash
        pm2 save
        success "PM2 autostart configured"
    fi
}

show_summary() {
    echo
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                   Installation Complete!                     ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo
    echo "Monitor Daemon has been installed and configured."
    echo
    echo -e "${CYAN}Locations:${NC}"
    echo "  Binary: $DAEMON_BIN"
    echo "  Config: $GLOBAL_CONFIG_DIR"
    echo "  Logs: $GLOBAL_CONFIG_DIR/logs"
    echo
    echo -e "${CYAN}PM2 Commands:${NC}"
    echo "  View status:  pm2 status monitor-daemon"
    echo "  View logs:    pm2 logs monitor-daemon"
    echo "  Restart:      pm2 restart monitor-daemon"
    echo "  Stop:         pm2 stop monitor-daemon"
    echo
    echo -e "${CYAN}Add More Projects:${NC}"
    echo "  Run: $0 --add-project"
    echo
    echo -e "${CYAN}Uninstall:${NC}"
    echo "  Run: $0 --uninstall"
}

add_project_mode() {
    print_header
    log "Adding new project to monitor daemon..."
    
    if [ ! -f "$GLOBAL_CONFIG_FILE" ]; then
        error "Monitor daemon not installed. Run installation first."
        exit 1
    fi
    
    configure_project
    
    # Restart daemon to pick up new project
    pm2 restart monitor-daemon
    
    success "Project added successfully"
}

uninstall() {
    print_header
    warning "This will remove Monitor Daemon and all configurations"
    
    if ! prompt_yes_no "Are you sure you want to uninstall?" n; then
        log "Uninstall cancelled"
        exit 0
    fi
    
    log "Uninstalling Monitor Daemon..."
    
    # Stop PM2 process
    pm2 stop monitor-daemon 2>/dev/null || true
    pm2 delete monitor-daemon 2>/dev/null || true
    
    # Remove binary
    sudo rm -f "$DAEMON_BIN"
    
    # Remove config (with confirmation)
    if prompt_yes_no "Remove all configuration and logs?" n; then
        rm -rf "$GLOBAL_CONFIG_DIR"
    fi
    
    success "Monitor Daemon uninstalled"
}

# Main installation flow
main() {
    # Parse arguments
    case "${1:-}" in
        --add-project)
            add_project_mode
            exit 0
            ;;
        --uninstall)
            uninstall
            exit 0
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --add-project    Add a new project to monitor"
            echo "  --uninstall      Remove Monitor Daemon"
            echo "  --help           Show this help message"
            exit 0
            ;;
    esac
    
    print_header
    
    echo "This script will install and configure Monitor Daemon for multiple projects."
    echo "The daemon will be managed by PM2 and start automatically on system boot."
    echo
    
    if ! prompt_yes_no "Continue with installation?" y; then
        log "Installation cancelled"
        exit 0
    fi
    
    # Check dependencies
    if ! check_dependencies; then
        exit 1
    fi
    
    # Create directories
    create_directories
    
    # Build daemon
    if ! build_daemon; then
        exit 1
    fi
    
    # Install binary
    install_binary
    
    # Configure first project
    if prompt_yes_no "Configure a project now?" y; then
        configure_project
    fi
    
    # Setup PM2
    setup_pm2
    
    # Show summary
    show_summary
}

# Run main function
main "$@"