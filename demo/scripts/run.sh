#!/bin/bash

set -e

echo "🚀 Starting Helios Demo..."

# Change to demo directory
cd "$(dirname "$0")/.."

# Check if image exists
if ! docker image inspect helios/demo:latest > /dev/null 2>&1; then
    echo "🏗️  Image not found. Building..."
    ./scripts/build.sh
fi

# Stop any existing container
docker stop helios-demo 2>/dev/null || true
docker rm helios-demo 2>/dev/null || true

# Run the demo
echo "🌌 Starting Helios Parallel Universe Engine..."
docker run -d \
    --name helios-demo \
    -p 3000:3000 \
    -p 3001:3001 \
    helios/demo:latest

# Wait for services to start
echo "⏳ Waiting for services to start..."
sleep 5

# Check health
echo "🔍 Checking service health..."
for i in {1..30}; do
    if curl -s http://localhost:3001/api/stats > /dev/null; then
        echo "✅ Backend is healthy!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Backend failed to start"
        docker logs helios-demo
        exit 1
    fi
    sleep 2
done

echo ""
echo "🎉 Helios Demo is running!"
echo ""
echo "🌐 Open your browser and visit:"
echo "   http://localhost:3000"
echo ""
echo "📊 Backend API available at:"
echo "   http://localhost:3001/api/stats"
echo ""
echo "📝 To stop the demo:"
echo "   docker stop helios-demo"
echo ""
echo "📋 To view logs:"
echo "   docker logs -f helios-demo"