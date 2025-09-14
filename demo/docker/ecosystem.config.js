module.exports = {
  apps: [
    {
      name: 'helios-backend',
      script: './backend/server.js',
      instances: 1,
      exec_mode: 'fork',
      env: {
        NODE_ENV: 'production',
        PORT: 3001
      },
      log_file: './logs/backend.log',
      error_file: './logs/backend-error.log',
      out_file: './logs/backend-out.log',
      log_date_format: 'YYYY-MM-DD HH:mm Z',
      merge_logs: true,
      max_restarts: 5,
      restart_delay: 2000,
      watch: false,
      autorestart: true
    },
    {
      name: 'helios-frontend',
      script: 'serve',
      args: [
        '-s', './frontend/build',
        '-l', '3000',
        '--cors'
      ],
      instances: 1,
      exec_mode: 'fork',
      env: {
        NODE_ENV: 'production'
      },
      log_file: './logs/frontend.log',
      error_file: './logs/frontend-error.log',
      out_file: './logs/frontend-out.log',
      log_date_format: 'YYYY-MM-DD HH:mm Z',
      merge_logs: true,
      max_restarts: 3,
      restart_delay: 1000,
      watch: false,
      autorestart: true
    }
  ]
};