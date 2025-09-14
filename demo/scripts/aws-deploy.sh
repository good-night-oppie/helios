#!/bin/bash

set -e

echo "🚀 Helios Demo - AWS EC2 Deployment"
echo "====================================="
echo

# Load configuration
if [ -f ~/.helios-aws-config ]; then
    source ~/.helios-aws-config
    echo "✅ Loaded AWS configuration"
else
    echo "❌ AWS configuration not found. Please run ./scripts/aws-setup.sh first"
    exit 1
fi

# Get the latest Ubuntu 22.04 LTS AMI ID
echo "🔍 Finding latest Ubuntu 22.04 LTS AMI..."
AMI_ID=$(aws ec2 describe-images \
    --owners 099720109477 \
    --filters \
        "Name=name,Values=ubuntu/images/hvm-ssd*/ubuntu-jammy-22.04-amd64-server-*" \
        "Name=state,Values=available" \
    --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
    --output text)

echo "✅ Using AMI: $AMI_ID"

# Instance configuration
INSTANCE_TYPE="t3.small"
TAG_NAME="Helios-Demo-$(date +%Y%m%d-%H%M%S)"

echo "🏗️ Launching EC2 instance..."
echo "   Instance Type: $INSTANCE_TYPE"
echo "   AMI: $AMI_ID"
echo "   Key Pair: $KEY_NAME"
echo "   Security Group: $SECURITY_GROUP_ID"
echo

# Launch instance
INSTANCE_ID=$(aws ec2 run-instances \
    --image-id "$AMI_ID" \
    --count 1 \
    --instance-type "$INSTANCE_TYPE" \
    --key-name "$KEY_NAME" \
    --security-group-ids "$SECURITY_GROUP_ID" \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$TAG_NAME}]" \
    --user-data file://scripts/ec2-user-data.sh \
    --query 'Instances[0].InstanceId' \
    --output text)

echo "✅ Instance launched: $INSTANCE_ID"
echo

# Wait for instance to be running
echo "⏳ Waiting for instance to be in running state..."
aws ec2 wait instance-running --instance-ids "$INSTANCE_ID"
echo "✅ Instance is running"

# Get public IP
PUBLIC_IP=$(aws ec2 describe-instances \
    --instance-ids "$INSTANCE_ID" \
    --query 'Reservations[0].Instances[0].PublicIpAddress' \
    --output text)

echo "✅ Public IP: $PUBLIC_IP"
echo

# Wait for SSH to be available
echo "⏳ Waiting for SSH to be available..."
for i in {1..30}; do
    if ssh -i ~/.ssh/"$KEY_NAME".pem -o ConnectTimeout=5 -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" 'echo "SSH Ready"' > /dev/null 2>&1; then
        echo "✅ SSH is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Timeout waiting for SSH"
        exit 1
    fi
    echo "   Attempt $i/30 - waiting 10 seconds..."
    sleep 10
done

# Deploy the Helios demo
echo "📦 Deploying Helios demo to EC2..."

# Create deployment script
cat > /tmp/deploy-helios.sh << 'EOF'
#!/bin/bash
set -e

echo "🔧 Setting up Helios demo on EC2..."

# Update system
sudo apt update

# Install Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Install Docker
sudo apt install -y docker.io
sudo usermod -aG docker ubuntu
sudo systemctl enable docker
sudo systemctl start docker

# Install Git
sudo apt install -y git

# Clone Helios demo (replace with your actual repository)
cd /home/ubuntu
git clone https://github.com/your-username/helios-demo.git || {
    echo "📁 Creating demo files manually..."
    mkdir -p helios-demo
    cd helios-demo
    # We'll upload the demo files manually since we don't have a git repo yet
}

echo "✅ Basic setup complete"
EOF

# Copy and run deployment script
scp -i ~/.ssh/"$KEY_NAME".pem -o StrictHostKeyChecking=no /tmp/deploy-helios.sh ubuntu@"$PUBLIC_IP":/home/ubuntu/
ssh -i ~/.ssh/"$KEY_NAME".pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" 'chmod +x /home/ubuntu/deploy-helios.sh && /home/ubuntu/deploy-helios.sh'

# Upload demo files
echo "📤 Uploading Helios demo files..."
scp -i ~/.ssh/"$KEY_NAME".pem -o StrictHostKeyChecking=no -r ../backend ../frontend ../scripts ubuntu@"$PUBLIC_IP":/home/ubuntu/helios-demo/

# Start the demo
echo "🚀 Starting Helios demo..."
ssh -i ~/.ssh/"$KEY_NAME".pem -o StrictHostKeyChecking=no ubuntu@"$PUBLIC_IP" << 'EOF'
cd /home/ubuntu/helios-demo
chmod +x scripts/*.sh

# Install dependencies and start services
echo "📦 Installing backend dependencies..."
cd backend && npm install && cd ..

echo "📦 Installing frontend dependencies..."
cd frontend && npm install && cd ..

echo "🚀 Starting services..."
# Start backend
cd backend && node server.js &
BACKEND_PID=$!

# Wait a moment for backend to start
sleep 5

# Start frontend
cd ../frontend && npm start &
FRONTEND_PID=$!

echo "✅ Helios demo is starting..."
echo "Backend PID: $BACKEND_PID"
echo "Frontend PID: $FRONTEND_PID"

# Save PIDs for later management
echo $BACKEND_PID > /tmp/helios-backend.pid
echo $FRONTEND_PID > /tmp/helios-frontend.pid

EOF

echo
echo "🎉 Deployment Complete!"
echo "======================="
echo "✅ Instance ID: $INSTANCE_ID"
echo "✅ Public IP: $PUBLIC_IP"
echo "✅ SSH Key: ~/.ssh/$KEY_NAME.pem"
echo
echo "🌐 Access URLs:"
echo "   Frontend: http://$PUBLIC_IP:3000"
echo "   Backend:  http://$PUBLIC_IP:3002"
echo
echo "📝 SSH Access:"
echo "   ssh -i ~/.ssh/$KEY_NAME.pem ubuntu@$PUBLIC_IP"
echo
echo "🔧 Service Management:"
echo "   Check status: ssh -i ~/.ssh/$KEY_NAME.pem ubuntu@$PUBLIC_IP 'ps aux | grep node'"
echo "   View logs:    ssh -i ~/.ssh/$KEY_NAME.pem ubuntu@$PUBLIC_IP 'tail -f /var/log/cloud-init-output.log'"
echo
echo "💰 Cost Estimate: ~\$0.50/day (t3.small)"
echo
echo "⚠️  Don't forget to terminate the instance when done:"
echo "   aws ec2 terminate-instances --instance-ids $INSTANCE_ID"
echo

# Save instance info for easy management
cat > ~/.helios-deployment-info << EOF
# Helios Demo Deployment Info
INSTANCE_ID=$INSTANCE_ID
PUBLIC_IP=$PUBLIC_IP
DEPLOYMENT_DATE=$(date)
TAG_NAME=$TAG_NAME
EOF

echo "📄 Deployment info saved to ~/.helios-deployment-info"
echo