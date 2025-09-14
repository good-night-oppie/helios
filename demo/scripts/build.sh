#!/bin/bash

set -e

echo "🏗️  Building Helios Demo..."

# Change to demo directory
cd "$(dirname "$0")/.."

# Build Docker image
echo "📦 Building Docker image..."
docker build -f docker/Dockerfile -t helios/demo:latest .

echo "✅ Build complete!"
echo ""
echo "🚀 To run the demo:"
echo "   docker run -p 3000:3000 -p 3001:3001 helios/demo:latest"
echo ""
echo "🌐 Or use docker-compose:"
echo "   cd docker && docker-compose up"