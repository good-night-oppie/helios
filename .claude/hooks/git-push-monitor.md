# Git Push CI Monitoring Hook

## Hook Configuration

When executing `git push` commands, Claude Code should automatically:

1. **After Push Success**:
   ```bash
   # Start CI monitoring in background
   nohup /home/dev/workspace/oppie-thunder/scripts/monitor_ci_automated.sh fix > /tmp/ci_monitor_latest.log 2>&1 &
   MONITOR_PID=$!
   echo "🚀 CI Monitor started (PID: $MONITOR_PID)"
   echo "📋 Check status: tail -f /tmp/ci_monitor_latest.log"
   ```

2. **For PR Branches** (feat/*, fix/*, etc.):
   ```bash
   # Get PR number if exists
   PR_NUMBER=$(gh pr view --json number -q .number 2>/dev/null || echo "")
   if [ -n "$PR_NUMBER" ]; then
     # Monitor PR specifically
     /home/dev/workspace/oppie-thunder/scripts/monitor_ci_automated.sh pr "$PR_NUMBER" &
   fi
   ```

3. **On CI Failure Detection**:
   - Automatically attempt fixes for common issues
   - Post analysis to PR if applicable
   - Generate fix commits if needed

## Implementation in Claude Code

When Claude Code detects a `git push` command in the Bash tool:

1. Execute the push normally
2. If successful, immediately follow with:
   ```bash
   # Check if CI monitoring is needed
   BRANCH=$(git rev-parse --abbrev-ref HEAD)
   if [[ "$BRANCH" =~ ^(feat|fix|chore)/ ]]; then
     echo "🔍 Starting CI monitoring for feature branch..."
     /home/dev/workspace/oppie-thunder/scripts/monitor_ci_automated.sh fix &
   fi
   ```

3. Report CI status to user:
   ```bash
   # Wait briefly for CI to start
   sleep 5
   
   # Get latest run status
   gh run list --limit 1 --json status,conclusion,url | jq -r '.[0] | "CI Status: \(.status) - \(.conclusion // "pending")\nURL: \(.url)"'
   ```

## Auto-Fix Capabilities

The monitoring script can automatically fix:
- Cache permission issues
- GitHub token permission problems
- Linting errors (with golangci-lint --fix)
- Simple test failures

## Usage Example

```claude
User: Push my changes
Claude: I'll push your changes and monitor the CI for you.

[Executes: git push origin feat/my-feature]
✅ Push successful

[Automatically starts CI monitoring]
🚀 CI monitoring started
📊 Monitoring PR #29...

[If CI fails]
❌ CI failed - attempting auto-fix...
🔧 Fixed: Cache permission issue
🔄 Re-running CI...
✅ CI passed after auto-fix!
```

## Configuration

Set these environment variables to customize behavior:
- `CI_MONITOR_TIMEOUT=300` - Monitor timeout in seconds (default: 300)
- `CI_MONITOR_AUTO_FIX=true` - Enable auto-fix attempts (default: true)
- `CI_MONITOR_PR_COMMENT=true` - Post PR comments (default: true)