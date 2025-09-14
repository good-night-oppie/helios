#!/bin/bash

set -e

echo "🌐 Helios Demo - Endpoint Exposure Tool"

# Configuration
FRONTEND_PORT=3000
BACKEND_PORT=3002
METHOD=${1:-"ngrok"}  # Default to ngrok for simplicity

show_help() {
    echo "Usage: $0 [METHOD]"
    echo ""
    echo "Methods:"
    echo "  cloudflare  - Use Cloudflare Tunnels (persistent URLs)"
    echo "  ngrok       - Use ngrok (quick setup, default)"
    echo "  localtunnel - Use localtunnel (no signup)"
    echo ""
    echo "Examples:"
    echo "  $0 ngrok"
    echo "  $0 cloudflare"
    echo "  $0 localtunnel"
}

check_demo_running() {
    echo "🔍 Checking if Helios demo is running..."

    if ! curl -s http://localhost:$FRONTEND_PORT > /dev/null; then
        echo "❌ Frontend not running on port $FRONTEND_PORT"
        echo "Please run: ./scripts/run.sh"
        exit 1
    fi

    if ! curl -s http://localhost:$BACKEND_PORT/api/stats > /dev/null; then
        echo "❌ Backend not running on port $BACKEND_PORT"
        echo "Please run: ./scripts/run.sh"
        exit 1
    fi

    echo "✅ Demo is running locally"
}

expose_with_ngrok() {
    echo "🚀 Exposing with ngrok..."

    if ! command -v ngrok &> /dev/null; then
        echo "❌ ngrok not installed"
        echo "Install: https://ngrok.com/download"
        exit 1
    fi

    echo "Starting ngrok tunnels..."
    echo "Frontend (UI): Starting tunnel on port $FRONTEND_PORT"
    echo "Backend (API): Starting tunnel on port $BACKEND_PORT"

    # Start both tunnels in background
    ngrok http $FRONTEND_PORT --log=stdout > /tmp/ngrok-frontend.log 2>&1 &
    NGROK_FRONTEND_PID=$!

    ngrok http $BACKEND_PORT --log=stdout > /tmp/ngrok-backend.log 2>&1 &
    NGROK_BACKEND_PID=$!

    # Wait for tunnels to start
    sleep 5

    # Extract URLs from ngrok API
    FRONTEND_URL=$(curl -s http://localhost:4040/api/tunnels | jq -r '.tunnels[] | select(.config.addr | contains("'$FRONTEND_PORT'")) | .public_url' | head -1)
    BACKEND_URL=$(curl -s http://localhost:4041/api/tunnels | jq -r '.tunnels[] | select(.config.addr | contains("'$BACKEND_PORT'")) | .public_url' | head -1)

    echo ""
    echo "🎉 Helios Demo is now publicly accessible!"
    echo ""
    echo "🌐 Frontend (Main Demo): $FRONTEND_URL"
    echo "📊 Backend API: $BACKEND_URL"
    echo ""
    echo "Share the frontend URL for the live demo presentation!"
    echo ""
    echo "Press Ctrl+C to stop tunnels"

    # Keep script running
    trap "kill $NGROK_FRONTEND_PID $NGROK_BACKEND_PID 2>/dev/null" EXIT
    wait
}

expose_with_cloudflare() {
    echo "🌩️ Exposing with Cloudflare Tunnels..."

    if ! command -v cloudflared &> /dev/null; then
        echo "❌ cloudflared not installed"
        echo "Install: curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb -o cloudflared.deb && sudo dpkg -i cloudflared.deb"
        exit 1
    fi

    echo "📝 Creating Cloudflare tunnel configuration..."

    # Create a simple tunnel config
    mkdir -p ~/.cloudflared

    cat > ~/.cloudflared/helios-config.yml << EOF
tunnel: helios-demo
credentials-file: ~/.cloudflared/helios-demo.json

ingress:
  - hostname: your-frontend-domain.trycloudflare.com
    service: http://localhost:$FRONTEND_PORT
  - hostname: your-backend-domain.trycloudflare.com
    service: http://localhost:$BACKEND_PORT
  - service: http_status:404
EOF

    echo "🚀 Starting Cloudflare tunnel (this creates temporary URLs)..."
    echo ""
    echo "Note: For persistent custom domains, set up a Cloudflare account first"
    echo ""

    # Use quick tunnel for temporary URLs
    cloudflared tunnel --url http://localhost:$FRONTEND_PORT &
    CLOUDFLARE_PID=$!

    sleep 5
    echo ""
    echo "🎉 Cloudflare tunnel started!"
    echo "Check the output above for your public URL"
    echo ""
    echo "Press Ctrl+C to stop tunnel"

    trap "kill $CLOUDFLARE_PID 2>/dev/null" EXIT
    wait
}

expose_with_localtunnel() {
    echo "🔗 Exposing with localtunnel..."

    if ! command -v lt &> /dev/null; then
        echo "Installing localtunnel..."
        npm install -g localtunnel
    fi

    echo "Starting localtunnel..."

    # Start tunnels with subdomain attempts
    lt --port $FRONTEND_PORT --subdomain helios-demo &
    LT_FRONTEND_PID=$!

    lt --port $BACKEND_PORT --subdomain helios-api &
    LT_BACKEND_PID=$!

    sleep 3

    echo ""
    echo "🎉 Localtunnel URLs:"
    echo "🌐 Frontend: https://helios-demo.loca.lt (or check terminal output above)"
    echo "📊 Backend: https://helios-api.loca.lt"
    echo ""
    echo "Press Ctrl+C to stop tunnels"

    trap "kill $LT_FRONTEND_PID $LT_BACKEND_PID 2>/dev/null" EXIT
    wait
}

# Main execution
case "$1" in
    "help"|"-h"|"--help")
        show_help
        exit 0
        ;;
    "cloudflare")
        check_demo_running
        expose_with_cloudflare
        ;;
    "ngrok")
        check_demo_running
        expose_with_ngrok
        ;;
    "localtunnel")
        check_demo_running
        expose_with_localtunnel
        ;;
    "")
        echo "No method specified, using ngrok (default)"
        check_demo_running
        expose_with_ngrok
        ;;
    *)
        echo "❌ Unknown method: $1"
        show_help
        exit 1
        ;;
esac