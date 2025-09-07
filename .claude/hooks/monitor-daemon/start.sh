#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: 2025 Good Night Oppie

# Start script for the monitor daemon

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DAEMON_BIN="$SCRIPT_DIR/monitor-daemon"
CONFIG_FILE="$SCRIPT_DIR/config.json"
PID_FILE="/tmp/monitor-daemon.pid"
LOG_FILE="/tmp/monitor-daemon.log"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Functions
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

build_daemon() {
    log "Building monitor daemon..."
    cd "$SCRIPT_DIR"
    
    if ! command -v go &> /dev/null; then
        error "Go is not installed"
        exit 1
    fi
    
    go mod download
    go build -o monitor-daemon main.go
    
    if [ -f "$DAEMON_BIN" ]; then
        success "Daemon built successfully"
    else
        error "Failed to build daemon"
        exit 1
    fi
}

check_running() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

start_daemon() {
    if check_running; then
        warning "Daemon is already running (PID: $(cat $PID_FILE))"
        exit 0
    fi
    
    # Check for GitHub token
    if [ -z "${GITHUB_TOKEN:-}" ]; then
        error "GITHUB_TOKEN environment variable is not set"
        echo "Please set it with: export GITHUB_TOKEN='your-token'"
        exit 1
    fi
    
    # Build if binary doesn't exist
    if [ ! -f "$DAEMON_BIN" ]; then
        build_daemon
    fi
    
    log "Starting monitor daemon..."
    
    # Start daemon in background
    nohup "$DAEMON_BIN" -config "$CONFIG_FILE" >> "$LOG_FILE" 2>&1 &
    PID=$!
    echo $PID > "$PID_FILE"
    
    # Wait a moment to check if it started successfully
    sleep 2
    
    if check_running; then
        success "Monitor daemon started (PID: $PID)"
        log "Monitoring logs at: $LOG_FILE"
    else
        error "Failed to start daemon"
        exit 1
    fi
}

stop_daemon() {
    if ! check_running; then
        warning "Daemon is not running"
        exit 0
    fi
    
    PID=$(cat "$PID_FILE")
    log "Stopping monitor daemon (PID: $PID)..."
    
    kill "$PID"
    
    # Wait for graceful shutdown
    for i in {1..10}; do
        if ! check_running; then
            rm -f "$PID_FILE"
            success "Daemon stopped"
            exit 0
        fi
        sleep 1
    done
    
    # Force kill if still running
    kill -9 "$PID" 2>/dev/null || true
    rm -f "$PID_FILE"
    success "Daemon force stopped"
}

status_daemon() {
    if check_running; then
        PID=$(cat "$PID_FILE")
        success "Monitor daemon is running (PID: $PID)"
        
        # Show recent logs
        if [ -f "$LOG_FILE" ]; then
            echo ""
            log "Recent activity:"
            tail -n 10 "$LOG_FILE"
        fi
    else
        warning "Monitor daemon is not running"
    fi
}

restart_daemon() {
    log "Restarting monitor daemon..."
    stop_daemon
    sleep 2
    start_daemon
}

show_logs() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        error "Log file not found: $LOG_FILE"
        exit 1
    fi
}

# Main script
case "${1:-}" in
    start)
        start_daemon
        ;;
    stop)
        stop_daemon
        ;;
    restart)
        restart_daemon
        ;;
    status)
        status_daemon
        ;;
    build)
        build_daemon
        ;;
    logs)
        show_logs
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|build|logs}"
        echo ""
        echo "Commands:"
        echo "  start    - Start the monitor daemon"
        echo "  stop     - Stop the monitor daemon"
        echo "  restart  - Restart the monitor daemon"
        echo "  status   - Check daemon status"
        echo "  build    - Build the daemon binary"
        echo "  logs     - Follow daemon logs"
        echo ""
        echo "Environment Variables:"
        echo "  GITHUB_TOKEN - Required for GitHub API access"
        echo "  DEBUG        - Set to 'true' for verbose output"
        exit 1
        ;;
esac