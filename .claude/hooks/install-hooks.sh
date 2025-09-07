#!/bin/bash
# SPDX-FileCopyrightText: 2025 Good Night Oppie
# SPDX-License-Identifier: MIT

# Installation script for optimized CI monitoring hooks
# Merges optimized hooks into Claude Code's actual configuration

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
CLAUDE_HOOKS_CONFIG="$HOME/.config/claude-code/hooks.json"
BACKUP_DIR="$HOME/.config/claude-code/backups"
SCRIPTS_DIR="$(dirname "$0")"

echo -e "${BLUE}=== Claude Code CI Monitoring Hooks Installation ===${NC}"
echo ""

# Check if scripts exist
if [ ! -f "$SCRIPTS_DIR/ci-monitor-optimized.sh" ]; then
    echo -e "${RED}❌ Error: ci-monitor-optimized.sh not found${NC}"
    echo "Please run this script from the .claude/hooks directory"
    exit 1
fi

# Make scripts executable
echo "Making scripts executable..."
chmod +x "$SCRIPTS_DIR"/*.sh

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup existing configuration
if [ -f "$CLAUDE_HOOKS_CONFIG" ]; then
    BACKUP_FILE="$BACKUP_DIR/hooks.json.$(date +%Y%m%d_%H%M%S)"
    echo -e "${YELLOW}📦 Backing up existing configuration to:${NC}"
    echo "   $BACKUP_FILE"
    cp "$CLAUDE_HOOKS_CONFIG" "$BACKUP_FILE"
else
    echo -e "${YELLOW}⚠️  No existing hooks configuration found${NC}"
    echo "Creating new configuration..."
    mkdir -p "$(dirname "$CLAUDE_HOOKS_CONFIG")"
    echo '{"hooks": {}, "settings": {}}' > "$CLAUDE_HOOKS_CONFIG"
fi

# Function to check if hook already exists
hook_exists() {
    local event=$1
    local script_name=$2
    
    if [ -f "$CLAUDE_HOOKS_CONFIG" ]; then
        grep -q "$script_name" "$CLAUDE_HOOKS_CONFIG" 2>/dev/null && return 0
    fi
    return 1
}

# Update hooks configuration using jq
echo ""
echo -e "${BLUE}Updating Claude Code hooks configuration...${NC}"

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${RED}❌ Error: jq is required but not installed${NC}"
    echo "Install with: sudo apt-get install jq (Ubuntu) or brew install jq (Mac)"
    exit 1
fi

# Read current configuration
CURRENT_CONFIG=$(cat "$CLAUDE_HOOKS_CONFIG")

# Update configuration with optimized hooks
UPDATED_CONFIG=$(echo "$CURRENT_CONFIG" | jq '
    # Ensure hooks object exists
    .hooks = (.hooks // {}) |
    
    # Update post-tool-use hooks for git push
    .hooks["post-tool-use"] = (
        .hooks["post-tool-use"] // [] |
        # Remove old monitor_ci_automated.sh entries
        map(select(
            if .action.script then
                (.action.script | contains("monitor_ci_automated.sh") | not)
            else
                true
            end
        )) +
        # Add optimized hook if not exists
        if any(.[]; 
            if .action.script then
                .action.script | contains("ci-monitor-optimized.sh")
            else
                false
            end
        ) then
            []
        else
            [{
                "name": "CI Monitor After Push (Optimized)",
                "description": "Ultra-fast CI monitoring after git push",
                "trigger": {
                    "tool": "Bash",
                    "pattern": "git push.*origin",
                    "excludePatterns": ["--dry-run", "--help", "-n"]
                },
                "action": {
                    "type": "script",
                    "script": "'"$SCRIPTS_DIR/ci-monitor-optimized.sh"'",
                    "args": ["background"],
                    "timeout": 1,
                    "background": true
                },
                "enabled": true
            }]
        end
    ) |
    
    # Update on-error hooks for CI failures  
    .hooks["on-error"] = (
        .hooks["on-error"] // [] |
        # Remove old entries
        map(select(
            if .action.script then
                (.action.script | contains("monitor_ci_automated.sh") | not)
            else
                true
            end
        )) +
        # Add optimized hook
        if any(.[]; 
            if .action.script then
                .action.script | contains("ci-monitor-optimized.sh")
            else
                false
            end
        ) then
            []
        else
            [{
                "name": "CI Failure Auto-Fix (Optimized)",
                "description": "Attempt to auto-fix CI failures with caching",
                "trigger": {
                    "tool": "Bash",
                    "pattern": "gh run|gh pr checks",
                    "errorPattern": "failure|failed"
                },
                "action": {
                    "type": "script",
                    "script": "'"$SCRIPTS_DIR/ci-monitor-optimized.sh"'",
                    "args": ["autofix"],
                    "timeout": 30
                },
                "enabled": true
            }]
        end
    ) |
    
    # Update settings for performance
    .settings = (.settings // {}) |
    .settings.enableHooks = true |
    .settings.logHookExecutions = true |
    .settings.logPath = "/tmp/claude-code-hooks.log" |
    .settings.maxConcurrentHooks = 3 |
    .settings.enableCaching = true |
    .settings.cacheDirectory = "/tmp/ci-monitor-cache" |
    .settings.asyncByDefault = true
')

# Write updated configuration
echo "$UPDATED_CONFIG" > "$CLAUDE_HOOKS_CONFIG"

echo -e "${GREEN}✅ Hooks configuration updated successfully!${NC}"
echo ""

# Verify installation
echo -e "${BLUE}Verifying installation...${NC}"

if grep -q "ci-monitor-optimized.sh" "$CLAUDE_HOOKS_CONFIG"; then
    echo -e "${GREEN}✅ Optimized CI monitoring hook installed${NC}"
else
    echo -e "${RED}❌ Installation verification failed${NC}"
    exit 1
fi

# Test the optimized script
echo ""
echo -e "${BLUE}Testing optimized CI monitoring script...${NC}"
if "$SCRIPTS_DIR/ci-monitor-optimized.sh" quick 2>/dev/null; then
    echo -e "${GREEN}✅ Script execution successful${NC}"
else
    echo -e "${YELLOW}⚠️  Script test failed (this may be normal if no CI is running)${NC}"
fi

# Display summary
echo ""
echo -e "${GREEN}=== Installation Complete ===${NC}"
echo ""
echo "📋 Summary:"
echo "  - Optimized hooks installed to: $CLAUDE_HOOKS_CONFIG"
echo "  - Previous config backed up to: ${BACKUP_FILE:-N/A}"
echo "  - Scripts location: $SCRIPTS_DIR"
echo ""
echo "🚀 Features enabled:"
echo "  - Sub-second CI status checks (<500ms)"
echo "  - 30-second result caching"
echo "  - Async execution (non-blocking)"
echo "  - Auto-fix for common CI failures"
echo ""
echo "📝 Next steps:"
echo "  1. Restart Claude Code or reload configuration"
echo "  2. Test with: git push origin <branch>"
echo "  3. Monitor logs: tail -f /tmp/claude-code-hooks.log"
echo ""
echo "🔧 To uninstall:"
echo "  Restore backup: cp $BACKUP_FILE $CLAUDE_HOOKS_CONFIG"
echo ""
echo -e "${BLUE}Happy coding with optimized CI monitoring! 🎉${NC}"