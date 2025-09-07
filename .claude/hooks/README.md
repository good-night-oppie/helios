# Claude Code Hooks

## Research Phase Hook

### Purpose
Automatically triggers research phase when a task is set to `in-progress` status.

### Hook: `task-in-progress-research.sh`

**Trigger**: When `task-master set-status --status=in-progress` is executed

**Actions**:
1. Notifies that research phase has started
2. Lists research sources to be consulted:
   - Context7 for official documentation
   - DeepWiki for technical concepts
   - Exa Deep Research for industry practices
3. Identifies research topics relevant to implementation
4. Prepares for logging findings to task notes

### Integration with TDD Workflow

The enhanced TDD workflow is now:
```
Research → Red → Green → Refactor → Validate
```

This hook ensures the Research phase is never skipped when starting a new task.

### Manual Research Trigger

If needed, you can manually trigger research for any task:
```bash
# Use these tools in parallel for comprehensive research:
mcp__context7__search "topic"
mcp__deepwiki__search "concept"
mcp__exa__deep_researcher_start "best practices for X"
```

### Recording Research Findings

Always record research findings before implementation:
```bash
task-master update-subtask --id=<task-id> --prompt="Research findings: [summarize key discoveries]"
```