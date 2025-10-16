#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
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

# Function to show usage
show_usage() {
    echo "Usage: $0 [local|external] [options]"
    echo ""
    echo "Options:"
    echo "  local     - Switch to local DynamoDB (DynamoDB Local)"
    echo "  external  - Switch to external DynamoDB (AWS)"
    echo ""
    echo "For external DynamoDB, you can provide:"
    echo "  --region REGION              AWS region (default: us-east-1)"
    echo "  --access-key ACCESS_KEY      AWS access key ID"
    echo "  --secret-key SECRET_KEY      AWS secret access key"
    echo ""
    echo "Examples:"
    echo "  $0 local"
    echo "  $0 external --region us-west-2 --access-key AKIA... --secret-key ..."
    echo "  $0 external  # Will prompt for credentials"
}

# Function to switch to local DynamoDB
switch_to_local() {
    print_status "Switching to local DynamoDB configuration..."
    
    # Update movimiento-inventario
    sed -i.bak 's|DYNAMODB_ENDPOINT_URL.*|DYNAMODB_ENDPOINT_URL: "http://dynamodb-local:8000"|' movimiento-inventario/k8s-deployment.yaml
    sed -i.bak 's|DYNAMODB_REGION.*|DYNAMODB_REGION: "us-east-1"|' movimiento-inventario/k8s-deployment.yaml
    sed -i.bak 's|DYNAMODB_ACCESS_KEY.*|DYNAMODB_ACCESS_KEY: "dummy"|' movimiento-inventario/k8s-deployment.yaml
    sed -i.bak 's|DYNAMODB_SECRET_KEY.*|DYNAMODB_SECRET_KEY: "dummy"|' movimiento-inventario/k8s-deployment.yaml
    
    # Update orden-compra
    sed -i.bak 's|AWS_REGION.*|AWS_REGION: "us-east-1"|' orden-compra/k8s-deployment.yaml
    sed -i.bak 's|AWS_ACCESS_KEY_ID.*|AWS_ACCESS_KEY_ID: "dummy"|' orden-compra/k8s-deployment.yaml
    sed -i.bak 's|AWS_SECRET_ACCESS_KEY.*|AWS_SECRET_ACCESS_KEY: "dummy"|' orden-compra/k8s-deployment.yaml
    
    # Update proveedor
    sed -i.bak 's|AWS_REGION.*|AWS_REGION: "us-east-1"|' proveedor/k8s-deployment.yaml
    sed -i.bak 's|AWS_ACCESS_KEY_ID.*|AWS_ACCESS_KEY_ID: "dummy"|' proveedor/k8s-deployment.yaml
    sed -i.bak 's|AWS_SECRET_ACCESS_KEY.*|AWS_SECRET_ACCESS_KEY: "dummy"|' proveedor/k8s-deployment.yaml
    
    print_success "Switched to local DynamoDB configuration"
    print_warning "Make sure DynamoDB Local is running in your cluster"
}

# Function to switch to external DynamoDB
switch_to_external() {
    local region=${REGION:-"us-east-1"}
    local access_key=${ACCESS_KEY}
    local secret_key=${SECRET_KEY}
    
    print_status "Switching to external DynamoDB configuration..."
    
    # Prompt for credentials if not provided
    if [ -z "$access_key" ]; then
        read -p "Enter AWS Access Key ID: " access_key
    fi
    
    if [ -z "$secret_key" ]; then
        read -s -p "Enter AWS Secret Access Key: " secret_key
        echo
    fi
    
    if [ -z "$access_key" ] || [ -z "$secret_key" ]; then
        print_error "Access key and secret key are required for external DynamoDB"
        exit 1
    fi
    
    # Update movimiento-inventario
    sed -i.bak '/DYNAMODB_ENDPOINT_URL/d' movimiento-inventario/k8s-deployment.yaml
    sed -i.bak "s|DYNAMODB_REGION.*|DYNAMODB_REGION: \"$region\"|" movimiento-inventario/k8s-deployment.yaml
    sed -i.bak "s|DYNAMODB_ACCESS_KEY.*|DYNAMODB_ACCESS_KEY: \"$access_key\"|" movimiento-inventario/k8s-deployment.yaml
    sed -i.bak "s|DYNAMODB_SECRET_KEY.*|DYNAMODB_SECRET_KEY: \"$secret_key\"|" movimiento-inventario/k8s-deployment.yaml
    
    # Update orden-compra
    sed -i.bak "s|AWS_REGION.*|AWS_REGION: \"$region\"|" orden-compra/k8s-deployment.yaml
    sed -i.bak "s|AWS_ACCESS_KEY_ID.*|AWS_ACCESS_KEY_ID: \"$access_key\"|" orden-compra/k8s-deployment.yaml
    sed -i.bak "s|AWS_SECRET_ACCESS_KEY.*|AWS_SECRET_ACCESS_KEY: \"$secret_key\"|" orden-compra/k8s-deployment.yaml
    
    # Update proveedor
    sed -i.bak "s|AWS_REGION.*|AWS_REGION: \"$region\"|" proveedor/k8s-deployment.yaml
    sed -i.bak "s|AWS_ACCESS_KEY_ID.*|AWS_ACCESS_KEY_ID: \"$access_key\"|" proveedor/k8s-deployment.yaml
    sed -i.bak "s|AWS_SECRET_ACCESS_KEY.*|AWS_SECRET_ACCESS_KEY: \"$secret_key\"|" proveedor/k8s-deployment.yaml
    
    print_success "Switched to external DynamoDB configuration"
    print_warning "Make sure the required DynamoDB tables exist in your AWS account"
    print_warning "See DYNAMODB_CONFIG.md for table creation instructions"
}

# Function to create DynamoDB tables
create_tables() {
    local region=${REGION:-"us-east-1"}
    
    print_status "Creating DynamoDB tables in region: $region"
    
    # Check if AWS CLI is available
    if ! command -v aws &> /dev/null; then
        print_error "AWS CLI is not installed. Please install it first."
        exit 1
    fi
    
    # Create tables for each service
    local services=("movimiento-inventario" "orden-compra" "proveedor" "ingreso-inventario")
    
    for service in "${services[@]}"; do
        print_status "Creating tables for $service..."
        
        # Create events table
        aws dynamodb create-table \
            --table-name "${service}-events" \
            --attribute-definitions \
                AttributeName=id,AttributeType=S \
                AttributeName=timestamp,AttributeType=S \
            --key-schema \
                AttributeName=id,KeyType=HASH \
                AttributeName=timestamp,KeyType=RANGE \
            --billing-mode PAY_PER_REQUEST \
            --region "$region" \
            --no-cli-pager || print_warning "Table ${service}-events might already exist"
        
        # Create read table
        aws dynamodb create-table \
            --table-name "${service}-read" \
            --attribute-definitions \
                AttributeName=id,AttributeType=S \
            --key-schema \
                AttributeName=id,KeyType=HASH \
            --billing-mode PAY_PER_REQUEST \
            --region "$region" \
            --no-cli-pager || print_warning "Table ${service}-read might already exist"
    done
    
    print_success "DynamoDB tables creation completed"
}

# Main script logic
case "$1" in
    "local")
        switch_to_local
        ;;
    "external")
        # Parse additional arguments
        while [[ $# -gt 1 ]]; do
            case $2 in
                --region)
                    REGION="$3"
                    shift 2
                    ;;
                --access-key)
                    ACCESS_KEY="$3"
                    shift 2
                    ;;
                --secret-key)
                    SECRET_KEY="$3"
                    shift 2
                    ;;
                --create-tables)
                    CREATE_TABLES=true
                    shift
                    ;;
                *)
                    print_error "Unknown option: $2"
                    show_usage
                    exit 1
                    ;;
            esac
        done
        
        switch_to_external
        
        if [ "$CREATE_TABLES" = true ]; then
            create_tables
        fi
        ;;
    *)
        show_usage
        exit 1
        ;;
esac

print_status "Configuration updated. You may need to restart the services:"
print_status "kubectl rollout restart deployment/movimiento-inventario -n event-mesh-system"
print_status "kubectl rollout restart deployment/orden-compra -n event-mesh-system"
print_status "kubectl rollout restart deployment/proveedor -n event-mesh-system"
