#!/bin/bash

set -e

echo "🔧 AWS Configuration Setup for Helios Demo Deployment"
echo "======================================================="
echo

# Function to prompt for input with default
prompt_with_default() {
    local prompt="$1"
    local default="$2"
    local result

    if [ -n "$default" ]; then
        read -p "$prompt [$default]: " result
        echo "${result:-$default}"
    else
        read -p "$prompt: " result
        echo "$result"
    fi
}

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "❌ AWS CLI not found. Please install AWS CLI first."
    exit 1
fi

echo "✅ AWS CLI found: $(aws --version)"
echo

# Configure AWS credentials
echo "📝 AWS Credentials Configuration"
echo "You need to provide your AWS credentials. Get them from:"
echo "   https://console.aws.amazon.com/iam/home#/security_credentials"
echo

# Get current configuration if exists
CURRENT_ACCESS_KEY=$(aws configure get aws_access_key_id 2>/dev/null || echo "")
CURRENT_REGION=$(aws configure get region 2>/dev/null || echo "us-east-1")

AWS_ACCESS_KEY_ID=$(prompt_with_default "AWS Access Key ID" "$CURRENT_ACCESS_KEY")
AWS_SECRET_ACCESS_KEY=$(prompt_with_default "AWS Secret Access Key" "")
AWS_DEFAULT_REGION=$(prompt_with_default "Default region" "$CURRENT_REGION")
AWS_DEFAULT_OUTPUT=$(prompt_with_default "Default output format" "json")

echo
echo "🔐 Configuring AWS CLI..."

# Configure AWS CLI
aws configure set aws_access_key_id "$AWS_ACCESS_KEY_ID"
aws configure set aws_secret_access_key "$AWS_SECRET_ACCESS_KEY"
aws configure set region "$AWS_DEFAULT_REGION"
aws configure set output "$AWS_DEFAULT_OUTPUT"

echo "✅ AWS CLI configured successfully!"
echo

# Test AWS connectivity
echo "🧪 Testing AWS connectivity..."
if aws sts get-caller-identity > /dev/null 2>&1; then
    echo "✅ AWS connectivity test successful!"
    echo
    echo "📊 Your AWS Identity:"
    aws sts get-caller-identity
    echo
else
    echo "❌ AWS connectivity test failed!"
    echo "Please check your credentials and try again."
    exit 1
fi

# Create key pair for EC2 if it doesn't exist
KEY_NAME="helios-demo-key"
echo "🔑 Checking for EC2 Key Pair: $KEY_NAME"

if aws ec2 describe-key-pairs --key-names "$KEY_NAME" > /dev/null 2>&1; then
    echo "✅ Key pair '$KEY_NAME' already exists"
else
    echo "🔧 Creating new key pair: $KEY_NAME"
    aws ec2 create-key-pair --key-name "$KEY_NAME" --query 'KeyMaterial' --output text > ~/.ssh/"$KEY_NAME".pem
    chmod 600 ~/.ssh/"$KEY_NAME".pem
    echo "✅ Key pair created and saved to ~/.ssh/$KEY_NAME.pem"
fi

# Create security group if it doesn't exist
SECURITY_GROUP_NAME="helios-demo-sg"
echo "🛡️ Checking for Security Group: $SECURITY_GROUP_NAME"

if aws ec2 describe-security-groups --group-names "$SECURITY_GROUP_NAME" > /dev/null 2>&1; then
    echo "✅ Security group '$SECURITY_GROUP_NAME' already exists"
    SECURITY_GROUP_ID=$(aws ec2 describe-security-groups --group-names "$SECURITY_GROUP_NAME" --query 'SecurityGroups[0].GroupId' --output text)
else
    echo "🔧 Creating security group: $SECURITY_GROUP_NAME"
    SECURITY_GROUP_ID=$(aws ec2 create-security-group \
        --group-name "$SECURITY_GROUP_NAME" \
        --description "Security group for Helios Demo" \
        --query 'GroupId' --output text)

    # Add inbound rules
    echo "🔧 Adding security group rules..."
    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp \
        --port 22 \
        --cidr 0.0.0.0/0 \
        --description "SSH access"

    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp \
        --port 80 \
        --cidr 0.0.0.0/0 \
        --description "HTTP access"

    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp \
        --port 443 \
        --cidr 0.0.0.0/0 \
        --description "HTTPS access"

    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp \
        --port 3000 \
        --cidr 0.0.0.0/0 \
        --description "Helios Frontend"

    aws ec2 authorize-security-group-ingress \
        --group-id "$SECURITY_GROUP_ID" \
        --protocol tcp \
        --port 3002 \
        --cidr 0.0.0.0/0 \
        --description "Helios Backend"

    echo "✅ Security group created with ID: $SECURITY_GROUP_ID"
fi

# Save configuration for deployment script
cat > ~/.helios-aws-config << EOF
# Helios Demo AWS Configuration
AWS_REGION=$AWS_DEFAULT_REGION
KEY_NAME=$KEY_NAME
SECURITY_GROUP_ID=$SECURITY_GROUP_ID
SECURITY_GROUP_NAME=$SECURITY_GROUP_NAME
EOF

echo
echo "🎉 AWS Setup Complete!"
echo "========================================"
echo "✅ AWS CLI configured and tested"
echo "✅ Key pair: $KEY_NAME"
echo "✅ Security group: $SECURITY_GROUP_NAME ($SECURITY_GROUP_ID)"
echo "✅ Configuration saved to ~/.helios-aws-config"
echo
echo "🚀 You can now run: ./scripts/aws-deploy.sh"
echo "   to deploy the Helios demo to EC2!"
echo