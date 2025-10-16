#!/bin/bash

# Quick deployment script for external DynamoDB
# This script prompts for AWS credentials and deploys with external DynamoDB

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
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

echo "🚀 External DynamoDB Deployment Script"
echo "======================================"
echo ""

# Get AWS credentials
read -p "Enter AWS Region (default: us-east-1): " AWS_REGION
AWS_REGION=${AWS_REGION:-us-east-1}

read -p "Enter AWS Access Key ID: " AWS_ACCESS_KEY
if [ -z "$AWS_ACCESS_KEY" ]; then
    print_error "AWS Access Key ID is required"
    exit 1
fi

read -s -p "Enter AWS Secret Access Key: " AWS_SECRET_KEY
echo ""
if [ -z "$AWS_SECRET_KEY" ]; then
    print_error "AWS Secret Access Key is required"
    exit 1
fi

echo ""
print_status "Deploying with external DynamoDB..."
print_status "Region: $AWS_REGION"
print_status "Access Key: ${AWS_ACCESS_KEY:0:8}..."

# Run the main deployment script with external DynamoDB
./scripts/deploy.sh \
    --dynamodb external \
    --region "$AWS_REGION" \
    --access-key "$AWS_ACCESS_KEY" \
    --secret-key "$AWS_SECRET_KEY"

echo ""
print_warning "Important: Make sure DynamoDB tables exist in AWS region $AWS_REGION"
print_warning "Run: ./scripts/setup-aws-dynamodb.sh --region $AWS_REGION"
