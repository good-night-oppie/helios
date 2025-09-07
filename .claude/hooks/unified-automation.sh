#!/bin/bash
# SPDX-FileCopyrightText: 2025 Good Night Oppie
# SPDX-License-Identifier: MIT

# Unified CI and PR Review Automation
# Comprehensive solution for automated development workflow

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
readonly CI_MONITOR="$SCRIPT_DIR/ci-monitor-optimized.sh"
readonly PR_MONITOR="$SCRIPT_DIR/pr-review-monitor.sh"
readonly CONFIG_FILE="$HOME/.config/claude-code/hooks.json"
readonly LOG_DIR="/tmp/unified-automation"

# Colors
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly MAGENTA='\033[0;35m'
readonly NC='\033[0m'

# Create log directory
mkdir -p "$LOG_DIR"

# Logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_DIR/unified.log"
}

# Detect workflow type from command
detect_workflow() {
    local command=$1
    
    # Git push - needs both CI and PR monitoring
    if echo "$command" | grep -q "git push"; then
        echo "push"
        return
    fi
    
    # PR creation - needs review setup
    if echo "$command" | grep -q "gh pr create"; then
        echo "pr-create"
        return
    fi
    
    # PR comment - might be review request
    if echo "$command" | grep -q "gh pr comment.*@claude"; then
        echo "review-request"
        return
    fi
    
    # CI check commands
    if echo "$command" | grep -q "gh run\|gh pr checks"; then
        echo "ci-check"
        return
    fi
    
    echo "unknown"
}

# Extract context from command
extract_context() {
    local command=$1
    local context="{}"
    
    # Extract branch
    local branch=$(git branch --show-current 2>/dev/null || echo "")
    if [ -n "$branch" ]; then
        context=$(echo "$context" | jq --arg b "$branch" '.branch = $b')
    fi
    
    # Extract PR number if exists
    if [ -n "$branch" ]; then
        local pr_number=$(gh pr list --head "$branch" --json number -q '.[0].number' 2>/dev/null || echo "")
        if [ -n "$pr_number" ]; then
            context=$(echo "$context" | jq --arg p "$pr_number" '.pr_number = $p')
        fi
    fi
    
    # Extract task ID from branch name
    local task_id=$(echo "$branch" | grep -oP 'task-\K[\d.]+' || echo "")
    if [ -n "$task_id" ]; then
        context=$(echo "$context" | jq --arg t "$task_id" '.task_id = $t')
    fi
    
    echo "$context"
}

# Handle git push workflow
handle_push_workflow() {
    local context=$1
    local pr_number=$(echo "$context" | jq -r '.pr_number // ""')
    
    log "🚀 Handling git push workflow"
    
    # Start CI monitoring in background
    log "Starting CI monitoring..."
    "$CI_MONITOR" background &
    local ci_pid=$!
    
    # If PR exists, start PR monitoring
    if [ -n "$pr_number" ]; then
        log "PR #$pr_number detected, starting review monitoring..."
        "$PR_MONITOR" monitor "$pr_number" &
        local pr_pid=$!
        
        echo -e "${GREEN}✅ Automation started:${NC}"
        echo -e "  • CI Monitor: PID $ci_pid"
        echo -e "  • PR Monitor: PID $pr_pid"
    else
        echo -e "${GREEN}✅ CI monitoring started: PID $ci_pid${NC}"
    fi
}

# Handle PR creation workflow
handle_pr_create() {
    local context=$1
    local task_id=$(echo "$context" | jq -r '.task_id // ""')
    
    log "📝 Handling PR creation"
    
    # Wait for PR to be created
    sleep 2
    
    # Get new PR number
    local branch=$(echo "$context" | jq -r '.branch // ""')
    if [ -n "$branch" ]; then
        local pr_number=$(gh pr list --head "$branch" --json number -q '.[0].number' 2>/dev/null || echo "")
        
        if [ -n "$pr_number" ]; then
            log "PR #$pr_number created"
            
            # Check if high complexity task
            local complexity=5
            if [ -n "$task_id" ]; then
                complexity=$(task-master get-task --id="$task_id" 2>/dev/null | jq -r '.complexity // 5' || echo "5")
            fi
            
            # Auto-request review for complex tasks
            if [ "$complexity" -ge 7 ]; then
                log "High complexity task ($complexity/10), requesting Claude review..."
                request_claude_review "$pr_number" "$task_id" "$complexity"
            fi
            
            # Start monitoring
            "$PR_MONITOR" monitor "$pr_number" &
            echo -e "${GREEN}✅ PR #$pr_number monitoring started${NC}"
        fi
    fi
}

# Request Claude review with appropriate persona
request_claude_review() {
    local pr_number=$1
    local task_id=$2
    local complexity=$3
    
    log "Requesting specialized review for PR #$pr_number"
    
    # Select reviewer persona based on complexity
    local reviewer_persona=""
    local expected_rounds=""
    
    if [ "$complexity" -ge 9 ]; then
        reviewer_persona="Chief Scientist"
        expected_rounds="3-4"
    elif [ "$complexity" -ge 7 ]; then
        reviewer_persona="Principal Engineer"
        expected_rounds="2-3"
    else
        reviewer_persona="Senior Developer"
        expected_rounds="1-2"
    fi
    
    # Post review request
    gh pr comment "$pr_number" --body "@claude Please review this implementation.

## Review Context
- Task: $task_id
- Complexity: $complexity/10
- Reviewer Role: $reviewer_persona
- Expected Rounds: $expected_rounds

Please provide a thorough architectural review focusing on:
1. Correctness and test coverage
2. Performance against targets
3. Clean room compliance (no blue_team references)
4. Architectural decisions and trade-offs

Use the complexity level to guide your review depth." 2>/dev/null || \
        log "Failed to post review request"
    
    log "✅ Review requested for PR #$pr_number"
}

# Handle review request workflow
handle_review_request() {
    local context=$1
    local pr_number=$(echo "$context" | jq -r '.pr_number // ""')
    
    if [ -n "$pr_number" ]; then
        log "Review requested for PR #$pr_number, starting monitor..."
        "$PR_MONITOR" monitor "$pr_number" &
        echo -e "${GREEN}✅ PR review monitoring started${NC}"
    fi
}

# Handle CI check workflow
handle_ci_check() {
    log "Running CI status check..."
    "$CI_MONITOR" quick
    
    # Check if failures need attention
    local status=$("$CI_MONITOR" quick 2>&1 | grep -oP 'Status: \K\w+' || echo "unknown")
    if [ "$status" = "failure" ]; then
        log "CI failures detected, running auto-fix..."
        "$CI_MONITOR" autofix &
        echo -e "${YELLOW}⚠️ CI failures detected - auto-fix running${NC}"
    fi
}

# Show automation status
show_status() {
    echo -e "${BLUE}=== Unified Automation Status ===${NC}"
    echo ""
    
    # Check CI monitor
    echo -e "${MAGENTA}CI Monitoring:${NC}"
    local ci_cache="/tmp/ci-monitor-cache/ci_status_*"
    if ls $ci_cache 1> /dev/null 2>&1; then
        local last_check=$(ls -t $ci_cache | head -1)
        local age=$(($(date +%s) - $(stat -c %Y "$last_check")))
        echo "  Last check: ${age}s ago"
        cat "$last_check" 2>/dev/null | jq -r '.[] | "  • \(.status) - \(.conclusion)"' | head -3
    else
        echo "  No recent checks"
    fi
    echo ""
    
    # Check PR monitors
    echo -e "${MAGENTA}PR Monitoring:${NC}"
    "$PR_MONITOR" status
    echo ""
    
    # Check running processes
    echo -e "${MAGENTA}Running Processes:${NC}"
    ps aux | grep -E "(ci-monitor|pr-review-monitor)" | grep -v grep || echo "  None"
}

# Main execution
main() {
    local command=${1:-}
    
    # Special commands
    case "$command" in
        "status")
            show_status
            exit 0
            ;;
        "clean")
            log "Cleaning cache and state..."
            "$CI_MONITOR" clean
            "$PR_MONITOR" clean
            rm -rf "$LOG_DIR"/*
            echo -e "${GREEN}✅ Cleanup complete${NC}"
            exit 0
            ;;
        "help"|"--help"|"-h")
            cat << EOF
Unified CI and PR Review Automation

Usage:
  $0 [command]           Auto-detect workflow and start automation
  $0 status             Show automation status
  $0 clean              Clean cache and state
  $0 help               Show this help

Automated Workflows:
  • Git Push: CI monitoring + PR review monitoring
  • PR Create: Auto-request review for complex tasks
  • Review Request: Start PR monitoring for responses
  • CI Check: Quick status with auto-fix if needed

Configuration:
  Edit ~/.config/claude-code/hooks.json to customize triggers

Logs:
  $LOG_DIR/unified.log - Main log
  /tmp/ci-monitor-cache/ - CI cache
  /tmp/pr-monitor-cache/ - PR cache
EOF
            exit 0
            ;;
    esac
    
    # Auto-detect workflow
    if [ -n "$command" ]; then
        local workflow=$(detect_workflow "$command")
        local context=$(extract_context "$command")
        
        log "Detected workflow: $workflow"
        log "Context: $context"
        
        case "$workflow" in
            "push")
                handle_push_workflow "$context"
                ;;
            "pr-create")
                handle_pr_create "$context"
                ;;
            "review-request")
                handle_review_request "$context"
                ;;
            "ci-check")
                handle_ci_check
                ;;
            *)
                log "No automation needed for this command"
                ;;
        esac
    else
        echo "No command provided, showing status..."
        show_status
    fi
}

# Run main
main "$@"