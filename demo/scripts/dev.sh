#!/bin/bash

set -e

echo "🔧 Starting Helios Demo in Development Mode..."

# Change to demo directory
cd "$(dirname "$0")/.."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18 or later."
    exit 1
fi

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    echo "❌ npm is not installed. Please install npm."
    exit 1
fi

# Install backend dependencies
echo "📦 Installing backend dependencies..."
cd backend
if [ ! -d "node_modules" ]; then
    npm install
fi

# Start backend in background
echo "🚀 Starting backend server..."
npm start &
BACKEND_PID=$!

# Wait for backend to start
sleep 3

# Install frontend dependencies
echo "📦 Installing frontend dependencies..."
cd ../frontend
if [ ! -d "node_modules" ]; then
    npm install
fi

# Start frontend
echo "🌐 Starting frontend server..."
echo ""
echo "🎉 Development servers starting..."
echo ""
echo "📊 Backend running at: http://localhost:3001"
echo "🌐 Frontend will open at: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop both servers"

# Start frontend (this will block)
npm start

# Clean up background process when script exits
trap "kill $BACKEND_PID 2>/dev/null" EXIT