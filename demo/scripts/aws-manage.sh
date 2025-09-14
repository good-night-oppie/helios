#!/bin/bash

set -e

echo "🔧 Helios Demo - AWS Management Tools"
echo "====================================="

# Load deployment info if available
if [ -f ~/.helios-deployment-info ]; then
    source ~/.helios-deployment-info
    echo "✅ Loaded deployment info: $INSTANCE_ID"
else
    echo "❌ No deployment info found. Please deploy first with ./scripts/aws-deploy.sh"
    exit 1
fi

# Function to show usage
show_usage() {
    echo
    echo "Usage: $0 [command]"
    echo
    echo "Commands:"
    echo "  status      - Check instance and service status"
    echo "  logs        - View application logs"
    echo "  connect     - SSH into the instance"
    echo "  restart     - Restart Helios services"
    echo "  stop        - Stop Helios services"
    echo "  start       - Start Helios services"
    echo "  terminate   - Terminate the EC2 instance"
    echo "  info        - Show deployment information"
    echo
}

# Parse command
case "${1:-info}" in
    "status")
        echo "📊 Checking instance status..."
        aws ec2 describe-instances --instance-ids "$INSTANCE_ID" --query 'Reservations[0].Instances[0].State.Name' --output text
        echo
        echo "🔍 Checking service status..."
        ssh -i ~/.ssh/helios-demo-key.pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" 'ps aux | grep node | grep -v grep || echo "No Node.js processes found"'
        ;;

    "logs")
        echo "📄 Viewing application logs..."
        ssh -i ~/.ssh/helios-demo-key.pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" 'tail -f /var/log/cloud-init-output.log'
        ;;

    "connect")
        echo "🔌 Connecting to instance..."
        ssh -i ~/.ssh/helios-demo-key.pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP"
        ;;

    "restart")
        echo "🔄 Restarting Helios services..."
        ssh -i ~/.ssh/helios-demo-key.pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" << 'EOF'
# Kill existing processes
pkill -f "node server.js" || true
pkill -f "npm start" || true
sleep 2

# Start services again
cd /home/ubuntu/helios-demo
echo "🚀 Starting backend..."
cd backend && node server.js &
echo $! > /tmp/helios-backend.pid

sleep 3

echo "🚀 Starting frontend..."
cd ../frontend && npm start &
echo $! > /tmp/helios-frontend.pid

echo "✅ Services restarted"
EOF
        ;;

    "stop")
        echo "⏹️ Stopping Helios services..."
        ssh -i ~/.ssh/helios-demo-key.pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" << 'EOF'
pkill -f "node server.js" || true
pkill -f "npm start" || true
rm -f /tmp/helios-*.pid
echo "✅ Services stopped"
EOF
        ;;

    "start")
        echo "▶️ Starting Helios services..."
        ssh -i ~/.ssh/helios-demo-key.pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" << 'EOF'
cd /home/ubuntu/helios-demo
echo "🚀 Starting backend..."
cd backend && node server.js &
echo $! > /tmp/helios-backend.pid

sleep 3

echo "🚀 Starting frontend..."
cd ../frontend && npm start &
echo $! > /tmp/helios-frontend.pid

echo "✅ Services started"
EOF
        ;;

    "terminate")
        echo "⚠️  WARNING: This will permanently delete the instance!"
        read -p "Are you sure you want to terminate instance $INSTANCE_ID? (y/N): " confirm
        if [[ $confirm =~ ^[Yy]$ ]]; then
            echo "🗑️ Terminating instance..."
            aws ec2 terminate-instances --instance-ids "$INSTANCE_ID"
            echo "✅ Instance termination initiated"
            echo "💰 This will stop all charges for this instance"
            rm -f ~/.helios-deployment-info
        else
            echo "❌ Termination cancelled"
        fi
        ;;

    "info")
        echo "📋 Deployment Information"
        echo "========================"
        echo "Instance ID: $INSTANCE_ID"
        echo "Public IP: $PUBLIC_IP"
        echo "Deployment: $DEPLOYMENT_DATE"
        echo "Tag: $TAG_NAME"
        echo
        echo "🌐 Access URLs:"
        echo "   Frontend: http://$PUBLIC_IP:3000"
        echo "   Backend:  http://$PUBLIC_IP:3002"
        echo
        echo "💰 Estimated cost: ~\$0.50/day"
        echo
        echo "Use './scripts/aws-manage.sh status' to check if services are running"
        ;;

    *)
        echo "❌ Unknown command: $1"
        show_usage
        exit 1
        ;;
esac