#!/bin/sh

echo "🚀 Starting Helios Demo..."

# Create logs directory
mkdir -p logs

# Start PM2 with ecosystem config
pm2 start ecosystem.config.js

# Keep container running and show logs
pm2 logs --lines 50