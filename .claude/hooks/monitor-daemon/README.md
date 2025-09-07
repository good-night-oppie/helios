# Monitor Daemon - Unified CI/PR Monitoring Service

A Go-based daemon that consolidates all CI/PR monitoring functionality into a single, efficient service with **multi-project support** and **PM2 process management**.

## Features

- **Multi-Project Support**: Monitor multiple GitHub repositories from a single daemon
- **PM2 Integration**: Automatic process management and system startup
- **PR Monitoring**: Tracks all open pull requests for Claude mentions and review requests
- **CI Monitoring**: Monitors GitHub Actions workflows and provides auto-fix suggestions
- **Debate Management**: Handles multi-round PR review debates with evidence-based responses
- **State Persistence**: Maintains monitoring state across restarts
- **Concurrent Monitoring**: Uses Go goroutines for efficient parallel monitoring
- **Auto-Recovery**: Automatic retry logic for transient failures
- **Interactive Onboarding**: Easy setup wizard for new projects

## Architecture Choice: Go

We chose Go over TypeScript and Python for several reasons:

1. **Performance**: Compiled binary with minimal resource usage
2. **Concurrency**: Native goroutines for parallel monitoring
3. **Project Consistency**: Primary backend language in oppie-thunder
4. **Single Binary**: Easy deployment without runtime dependencies
5. **System Service**: Ideal for long-running daemons

## Quick Start (Interactive Installation)

```bash
# Run the interactive installer
cd .claude/hooks/monitor-daemon
./install.sh

# This will:
# 1. Check dependencies (Go, PM2, jq)
# 2. Build the daemon
# 3. Install globally (/usr/local/bin)
# 4. Configure your first project
# 5. Set up PM2 with autostart
```

## Installation

### Prerequisites

- Go 1.18 or later (tested with 1.18)
- PM2 (`npm install -g pm2`)
- jq (`apt-get install jq` or `brew install jq`)
- GitHub Personal Access Token with repo permissions

### Interactive Installation (Recommended)

```bash
./install.sh
```

The installer will:
- Check all dependencies
- Build and install the daemon globally
- Create configuration directories at `~/.monitor-daemon/`
- Configure your first project interactively
- Set up PM2 for process management
- Enable autostart on system boot (optional)

### Manual Build

```bash
cd .claude/hooks/monitor-daemon
./start.sh build
```

### Configuration

1. Set your GitHub token:
```bash
export GITHUB_TOKEN="your-github-token"
```

2. Edit `config.json` to customize settings:
```json
{
  "check_interval": "2m",    // How often to check for updates
  "enable_pr_monitor": true,  // Monitor pull requests
  "enable_ci_monitor": true,  // Monitor CI runs
  "enable_debate": true,      // Handle PR debates
  "debug_mode": false         // Verbose logging
}
```

## Usage

### With PM2 (After Installation)

```bash
# View status
pm2 status monitor-daemon

# View logs
pm2 logs monitor-daemon

# Restart daemon
pm2 restart monitor-daemon

# Stop daemon
pm2 stop monitor-daemon

# Monitor resource usage
pm2 monit
```

### Multi-Project Management

```bash
# Add a new project to monitor
./install.sh --add-project

# View all projects
cat ~/.monitor-daemon/config.json | jq '.projects'

# Edit project configuration
nano ~/.monitor-daemon/projects/<project-name>.json

# Restart daemon to apply changes
pm2 restart monitor-daemon
```

### Manual Operation (Without PM2)

```bash
# Start the daemon
./start.sh start

# Stop the daemon
./start.sh stop

# Check status
./start.sh status

# View logs
./start.sh logs

# Restart daemon
./start.sh restart
```

## Systemd Service (Optional)

For production deployment as a system service:

1. Copy and configure the service file:
```bash
sudo cp monitor-daemon.service /etc/systemd/system/
sudo systemctl edit monitor-daemon.service
# Set your user and GitHub token
```

2. Enable and start:
```bash
sudo systemctl enable monitor-daemon
sudo systemctl start monitor-daemon
```

## Monitoring Capabilities

### PR Monitoring
- Detects Claude mentions (@claude)
- Tracks review requests
- Monitors comment threads
- Manages debate rounds

### CI Monitoring
- Tracks workflow runs
- Detects failures
- Generates fix suggestions
- Posts automated responses

### Debate Management
- Tracks debate rounds (max 3)
- Generates evidence-based responses
- Detects approval keywords
- Marks tasks complete when approved

## State Management

The daemon maintains state in `/tmp/monitor-state/daemon.state` containing:
- Active monitors
- Last check times
- Debate rounds
- Comment IDs

This ensures continuity across restarts.

## Logs

- Main log: `/tmp/monitor-daemon.log`
- Error log: `/tmp/monitor-daemon.error.log` (systemd only)
- Debug mode: Set `DEBUG=true` for console output

## Development

### Adding New Features

1. Add feature flag to `Config` struct
2. Implement monitoring goroutine
3. Add to `Start()` method
4. Update config.json

### Testing

```bash
# Run with debug mode
DEBUG=true ./monitor-daemon -config config.json

# Check specific PR
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/good-night-oppie/oppie-thunder/pulls/38
```

## Replaces These Scripts

This daemon consolidates and replaces:
- `scripts/monitor_*.sh`
- `tools/monitor_*.sh`
- `.claude/hooks/*monitor*.sh`
- Individual PR monitoring scripts

## Performance

- Memory: ~10-20MB
- CPU: <1% idle, ~5% active
- Network: Minimal (API calls only)
- Startup: <100ms
- Binary size: ~7MB

## Security

- Token never logged
- State files use restricted permissions
- No sensitive data in cache
- Graceful shutdown preserves state

## Troubleshooting

### Daemon won't start
- Check GitHub token is set
- Verify Go version (1.23+)
- Check port conflicts

### Not detecting PRs
- Verify token has repo permissions
- Check API rate limits
- Review debug logs

### High CPU usage
- Increase check_interval
- Disable unused features
- Check for API errors

## Future Enhancements

- [ ] WebSocket support for real-time updates
- [ ] Prometheus metrics endpoint
- [ ] Web UI for monitoring
- [ ] Slack/Discord notifications
- [ ] Custom webhook support