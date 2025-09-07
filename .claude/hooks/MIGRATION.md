# Migration Guide: Monitoring Scripts → Monitor Daemon

## Overview

All monitoring scripts have been consolidated into a single Go daemon located at `.claude/hooks/monitor-daemon/`. The new daemon supports **multiple projects**, **PM2 process management**, and **automatic system startup**.

## What Changed

### Removed Scripts
- `scripts/monitor_*.sh` - Individual PR monitoring scripts
- `tools/monitor_*.sh` - CI monitoring tools
- `.claude/hooks/*monitor*.sh` - Hook-based monitors
- `agents/claude/hooks/*monitor*.sh` - Agent monitors

### New Unified Daemon
- **Location**: `.claude/hooks/monitor-daemon/`
- **Language**: Go (compiled binary)
- **Features**: All previous functionality plus improvements

## Migration Steps

### 1. Stop Old Scripts

If you have any monitoring scripts running, stop them:

```bash
# Find running monitor processes
ps aux | grep monitor

# Kill old processes
pkill -f monitor_pr
pkill -f monitor_ci
```

### 2. Install New Daemon (Interactive)

```bash
cd .claude/hooks/monitor-daemon
./install.sh
```

The interactive installer will:
- Check dependencies (Go, PM2, jq)
- Build the daemon
- Install globally to `/usr/local/bin`
- Configure your projects interactively
- Set up PM2 with autostart

### 3. Add Additional Projects

```bash
# Add more projects to monitor
./install.sh --add-project
```

### 4. Verify Installation

```bash
# Check PM2 status
pm2 status monitor-daemon

# View logs
pm2 logs monitor-daemon
```

## Feature Mapping

| Old Script | New Daemon Feature | Config Setting |
|------------|-------------------|----------------|
| `monitor_pr_*.sh` | PR Monitoring | `enable_pr_monitor: true` |
| `monitor_ci_*.sh` | CI Monitoring | `enable_ci_monitor: true` |
| `monitor_pr_comments.sh` | Debate Management | `enable_debate: true` |
| `pr-review-monitor.sh` | All Features | Default config |

## Command Equivalents

### Old Way
```bash
# Monitor specific PR
./scripts/monitor_pr_38.sh

# Monitor CI
./scripts/monitor_ci_automated.sh pr 38

# Monitor with debate
./scripts/monitor_pr_comments.sh 38 10 8 architecture
```

### New Way
```bash
# All monitoring in one daemon
./start.sh start

# Check status
./start.sh status

# View logs
./start.sh logs
```

## Benefits of Migration

1. **Single Process**: One daemon instead of multiple scripts
2. **Multi-Project Support**: Monitor multiple repositories from one daemon
3. **PM2 Management**: Professional process management with autostart
4. **Better Performance**: Go's concurrency vs bash loops
5. **State Persistence**: Survives restarts
6. **Unified Logging**: Centralized logs at `~/.monitor-daemon/logs/`
7. **Resource Efficient**: ~10MB memory vs multiple bash processes
8. **Auto-Recovery**: Built-in retry logic
9. **Interactive Setup**: Easy onboarding for new projects
10. **Global Installation**: Available system-wide at `/usr/local/bin`

## Configuration Options

Edit `.claude/hooks/monitor-daemon/config.json`:

```json
{
  "check_interval": "2m",      // How often to check
  "enable_pr_monitor": true,    // Monitor PRs
  "enable_ci_monitor": true,    // Monitor CI
  "enable_debate": true,        // Handle debates
  "debug_mode": false,          // Verbose output
  "features": {
    "auto_fix_ci": true,        // Auto-suggest fixes
    "auto_respond_reviews": true, // Auto-respond to reviews
    "track_complexity": true,    // Track task complexity
    "generate_evidence": true   // Generate evidence for debates
  }
}
```

## Troubleshooting

### Issue: Scripts still running
```bash
# Force kill all old monitors
pkill -9 -f monitor
```

### Issue: Daemon won't start
```bash
# Check token
echo $GITHUB_TOKEN

# Check logs
cat /tmp/monitor-daemon.log

# Run in debug mode
DEBUG=true ./.claude/hooks/monitor-daemon/monitor-daemon
```

### Issue: Not detecting PRs
- Verify GitHub token has repo permissions
- Check API rate limits
- Increase `check_interval` if hitting limits

## Rollback (If Needed)

The old scripts are preserved in git history. To restore:

```bash
git checkout HEAD~1 -- scripts/monitor_*.sh
git checkout HEAD~1 -- tools/monitor_*.sh
```

## Support

For issues or questions:
1. Check daemon logs: `./start.sh logs`
2. Run in debug mode: `DEBUG=true ./start.sh start`
3. Review README: `.claude/hooks/monitor-daemon/README.md`