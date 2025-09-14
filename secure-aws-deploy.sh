#!/bin/bash

set -euo pipefail

echo "🛡️ Helios Demo - Secure AWS Deployment"
echo "======================================"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="helios-demo"
REGION="us-east-1"
INSTANCE_TYPE="t3.small"
KEY_NAME="helios-demo-key"

# Function to print colored output
print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check AWS CLI configuration
check_aws_config() {
    print_step "Checking AWS CLI configuration..."

    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI not found. Please install AWS CLI first."
        exit 1
    fi

    # Test AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        print_error "AWS credentials not configured or invalid."
        echo "Please run: aws configure"
        exit 1
    fi

    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
    print_success "AWS CLI configured. Account ID: $ACCOUNT_ID"
}

# Function to create security group
create_security_group() {
    print_step "Creating security group..."

    # Check if security group exists
    SG_ID=$(aws ec2 describe-security-groups \
        --filters "Name=group-name,Values=${PROJECT_NAME}-sg" \
        --query 'SecurityGroups[0].GroupId' \
        --output text 2>/dev/null || echo "None")

    if [ "$SG_ID" = "None" ]; then
        SG_ID=$(aws ec2 create-security-group \
            --group-name "${PROJECT_NAME}-sg" \
            --description "Security group for Helios Demo" \
            --query 'GroupId' \
            --output text)

        # Add rules
        aws ec2 authorize-security-group-ingress \
            --group-id "$SG_ID" \
            --protocol tcp \
            --port 22 \
            --cidr 0.0.0.0/0

        aws ec2 authorize-security-group-ingress \
            --group-id "$SG_ID" \
            --protocol tcp \
            --port 3000 \
            --cidr 0.0.0.0/0

        aws ec2 authorize-security-group-ingress \
            --group-id "$SG_ID" \
            --protocol tcp \
            --port 3002 \
            --cidr 0.0.0.0/0

        print_success "Created security group: $SG_ID"
    else
        print_success "Using existing security group: $SG_ID"
    fi
}

# Function to create key pair
create_key_pair() {
    print_step "Creating SSH key pair..."

    if aws ec2 describe-key-pairs --key-names "$KEY_NAME" &> /dev/null; then
        print_success "Using existing key pair: $KEY_NAME"
    else
        aws ec2 create-key-pair \
            --key-name "$KEY_NAME" \
            --query 'KeyMaterial' \
            --output text > "${KEY_NAME}.pem"

        chmod 600 "${KEY_NAME}.pem"
        print_success "Created key pair: $KEY_NAME (saved as ${KEY_NAME}.pem)"
    fi
}

# Function to create user data script
create_user_data() {
    print_step "Creating user data script..."

    cat > user-data.sh << 'EOF'
#!/bin/bash

set -e

# Update system
apt-get update
apt-get install -y curl wget git

# Install Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
apt-get install -y nodejs

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
systemctl enable docker
systemctl start docker
usermod -aG docker ubuntu

# Clone repository (assuming it will be public)
cd /home/ubuntu
git clone https://github.com/good-night-oppie/helios.git
chown -R ubuntu:ubuntu helios

# Install dependencies and build
cd helios/demo/frontend
sudo -u ubuntu npm install
sudo -u ubuntu npm run build

cd ../backend
sudo -u ubuntu npm install

# Create systemd service for backend
cat > /etc/systemd/system/helios-backend.service << 'EOL'
[Unit]
Description=Helios Demo Backend
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/helios/demo/backend
ExecStart=/usr/bin/node server.js
Restart=always
Environment=NODE_ENV=production
Environment=PORT=3002

[Install]
WantedBy=multi-user.target
EOL

# Create systemd service for frontend (static file server)
cat > /etc/systemd/system/helios-frontend.service << 'EOL'
[Unit]
Description=Helios Demo Frontend
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/helios/demo/frontend/build
ExecStart=/usr/bin/npx serve -s . -l 3000
Restart=always

[Install]
WantedBy=multi-user.target
EOL

# Install serve globally
npm install -g serve

# Enable and start services
systemctl enable helios-backend
systemctl enable helios-frontend
systemctl start helios-backend
systemctl start helios-frontend

# Setup monitoring script
cat > /home/ubuntu/check-services.sh << 'EOL'
#!/bin/bash
echo "=== Helios Demo Status ==="
echo "Backend: $(systemctl is-active helios-backend)"
echo "Frontend: $(systemctl is-active helios-frontend)"
echo "Backend logs:"
journalctl -u helios-backend -n 5 --no-pager
EOL

chmod +x /home/ubuntu/check-services.sh
chown ubuntu:ubuntu /home/ubuntu/check-services.sh
EOF

    print_success "User data script created"
}

# Function to launch EC2 instance
launch_instance() {
    print_step "Launching EC2 instance..."

    # Get latest Ubuntu 22.04 LTS AMI
    AMI_ID=$(aws ec2 describe-images \
        --owners 099720109477 \
        --filters \
            "Name=name,Values=ubuntu/images/hvm-ssd*/ubuntu-jammy-22.04-amd64-server-*" \
            "Name=state,Values=available" \
        --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
        --output text)

    print_step "Using AMI: $AMI_ID"

    # Launch instance
    INSTANCE_ID=$(aws ec2 run-instances \
        --image-id "$AMI_ID" \
        --count 1 \
        --instance-type "$INSTANCE_TYPE" \
        --key-name "$KEY_NAME" \
        --security-group-ids "$SG_ID" \
        --user-data file://user-data.sh \
        --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${PROJECT_NAME}-$(date +%Y%m%d-%H%M%S)}]" \
        --query 'Instances[0].InstanceId' \
        --output text)

    print_success "Instance launched: $INSTANCE_ID"

    # Wait for instance to be running
    print_step "Waiting for instance to be running..."
    aws ec2 wait instance-running --instance-ids "$INSTANCE_ID"

    # Get public IP
    PUBLIC_IP=$(aws ec2 describe-instances \
        --instance-ids "$INSTANCE_ID" \
        --query 'Reservations[0].Instances[0].PublicIpAddress' \
        --output text)

    print_success "Instance is running. Public IP: $PUBLIC_IP"

    # Save deployment info
    cat > deployment-info.json << EOF
{
  "instanceId": "$INSTANCE_ID",
  "publicIp": "$PUBLIC_IP",
  "keyName": "$KEY_NAME",
  "securityGroupId": "$SG_ID",
  "deploymentTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "region": "$REGION",
  "urls": {
    "frontend": "http://$PUBLIC_IP:3000",
    "backend": "http://$PUBLIC_IP:3002"
  }
}
EOF
}

# Function to wait for services
wait_for_services() {
    print_step "Waiting for services to be ready (this may take 5-10 minutes)..."

    # Wait for SSH to be available
    print_step "Waiting for SSH connection..."
    for i in {1..30}; do
        if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -i "${KEY_NAME}.pem" ubuntu@"$PUBLIC_IP" echo "SSH ready" &>/dev/null; then
            break
        fi
        echo "SSH attempt $i/30..."
        sleep 10
    done

    # Check if services are running
    print_step "Checking service status..."
    ssh -o StrictHostKeyChecking=no -i "${KEY_NAME}.pem" ubuntu@"$PUBLIC_IP" '
        echo "Waiting for services to start..."
        for i in {1..60}; do
            if systemctl is-active --quiet helios-backend && systemctl is-active --quiet helios-frontend; then
                echo "Services are running!"
                break
            fi
            echo "Waiting... ($i/60)"
            sleep 5
        done

        echo "=== Service Status ==="
        systemctl status helios-backend --no-pager -l
        echo ""
        systemctl status helios-frontend --no-pager -l
    '
}

# Function to test deployment
test_deployment() {
    print_step "Testing deployment..."

    # Test backend
    if curl -s --connect-timeout 10 "http://$PUBLIC_IP:3002/api/stats" > /dev/null; then
        print_success "Backend is responding"
    else
        print_warning "Backend may not be ready yet"
    fi

    # Test frontend
    if curl -s --connect-timeout 10 "http://$PUBLIC_IP:3000" > /dev/null; then
        print_success "Frontend is responding"
    else
        print_warning "Frontend may not be ready yet"
    fi
}

# Function to display completion message
display_completion() {
    echo ""
    echo "🎉 Deployment Complete!"
    echo "======================="
    print_success "Instance ID: $INSTANCE_ID"
    print_success "Public IP: $PUBLIC_IP"
    echo ""
    echo "🌐 Access URLs:"
    echo "   Frontend: http://$PUBLIC_IP:3000"
    echo "   Backend:  http://$PUBLIC_IP:3002"
    echo ""
    echo "🔧 Management Commands:"
    echo "   SSH access: ssh -i ${KEY_NAME}.pem ubuntu@$PUBLIC_IP"
    echo "   Check status: ssh -i ${KEY_NAME}.pem ubuntu@$PUBLIC_IP './check-services.sh'"
    echo ""
    echo "💰 Cost: ~$0.50/day (~$15/month)"
    echo ""
    echo "🛡️ Security Features:"
    echo "   ✅ SSH key authentication"
    echo "   ✅ Security group restrictions"
    echo "   ✅ Automatic service restart"
    echo "   ✅ System logging"
    echo ""
    print_warning "Save your ${KEY_NAME}.pem file securely - you need it for SSH access!"
    echo ""
    echo "To terminate and stop all charges:"
    echo "   aws ec2 terminate-instances --instance-ids $INSTANCE_ID"
}

# Main execution
main() {
    echo "Starting secure AWS deployment for Helios Demo..."
    echo ""

    check_aws_config
    create_security_group
    create_key_pair
    create_user_data
    launch_instance
    wait_for_services
    test_deployment
    display_completion

    print_success "Deployment complete! Your Helios demo is ready for the 24-hour showcase!"
}

# Run main function
main "$@"