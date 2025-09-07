// SPDX-License-Identifier: MIT
// PM2 Ecosystem Configuration for Monitor Daemon
// Supports multiple projects with independent monitoring

module.exports = {
  apps: [
    {
      name: 'monitor-daemon',
      script: './monitor-daemon',
      cwd: '/home/dev/workspace/oppie-thunder/.claude/hooks/monitor-daemon',
      args: '-config /home/dev/.monitor-daemon/config.json',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      watch: false,
      max_memory_restart: '100M',
      env: {
        NODE_ENV: 'production',
        DAEMON_MODE: 'multi-project'
      },
      error_file: '/home/dev/.monitor-daemon/logs/error.log',
      out_file: '/home/dev/.monitor-daemon/logs/out.log',
      log_file: '/home/dev/.monitor-daemon/logs/combined.log',
      time: true,
      merge_logs: true,
      log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
      
      // Restart strategies
      min_uptime: '10s',
      max_restarts: 10,
      restart_delay: 4000,
      
      // Graceful shutdown
      kill_timeout: 3000,
      wait_ready: true,
      listen_timeout: 3000,
    }
  ],

  // Deploy configuration (optional)
  deploy: {
    production: {
      user: process.env.USER,
      host: 'localhost',
      ref: 'origin/main',
      repo: 'git@github.com:good-night-oppie/oppie-thunder.git',
      path: '/home/dev/workspace/oppie-thunder',
      'post-deploy': 'cd .claude/hooks/monitor-daemon && go build && pm2 reload ecosystem.config.js --env production',
      'pre-deploy-local': ''
    }
  }
};