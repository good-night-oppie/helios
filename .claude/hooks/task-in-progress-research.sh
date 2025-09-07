#!/bin/bash
# SPDX-FileCopyrightText: 2025 Yongbing Tang and contributors
# SPDX-License-Identifier: MIT

# Claude Hook: Automatic Research Phase for In-Progress Tasks
# Triggers when task-master set-status --status=in-progress is executed

# Check if this is a task status change to in-progress
if [[ "$1" == *"task-master"* ]] && [[ "$1" == *"set-status"* ]] && [[ "$1" == *"in-progress"* ]]; then
    echo "🔬 RESEARCH PHASE INITIATED - Gathering best practices and documentation..."
    echo ""
    echo "📚 Research Sources Being Consulted:"
    echo "  • Context7: Official documentation and API references"
    echo "  • DeepWiki: Technical concepts and implementation patterns"
    echo "  • Exa Deep Research: Industry best practices and case studies"
    echo ""
    echo "⏳ This research will inform the implementation strategy..."
    echo ""
    echo "💡 Research Topics for Current Task:"
    echo "  - Design patterns and architectural decisions"
    echo "  - Performance optimization techniques"
    echo "  - Security considerations and vulnerabilities"
    echo "  - Testing strategies and edge cases"
    echo "  - Common pitfalls and solutions"
    echo ""
    echo "📝 Research findings will be logged to task notes before implementation begins."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
fi