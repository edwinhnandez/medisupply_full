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

# Default values
REGION="us-east-1"
PROFILE=""
DRY_RUN=false

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --region REGION     AWS region (default: us-east-1)"
    echo "  --profile PROFILE   AWS profile to use"
    echo "  --dry-run          Show what would be created without actually creating"
    echo "  --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --region us-west-2"
    echo "  $0 --profile production --region eu-west-1"
    echo "  $0 --dry-run"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --region)
            REGION="$2"
            shift 2
            ;;
        --profile)
            PROFILE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Build AWS CLI command
AWS_CMD="aws"
if [ -n "$PROFILE" ]; then
    AWS_CMD="$AWS_CMD --profile $PROFILE"
fi
AWS_CMD="$AWS_CMD --region $REGION"

# Function to create a table
create_table() {
    local table_name="$1"
    local key_schema="$2"
    local attribute_definitions="$3"
    
    print_status "Creating table: $table_name"
    
    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY RUN: Would create table $table_name"
        return 0
    fi
    
    # Check if table already exists
    if $AWS_CMD dynamodb describe-table --table-name "$table_name" &>/dev/null; then
        print_warning "Table $table_name already exists, skipping..."
        return 0
    fi
    
    # Create the table
    $AWS_CMD dynamodb create-table \
        --table-name "$table_name" \
        --attribute-definitions $attribute_definitions \
        --key-schema $key_schema \
        --billing-mode PAY_PER_REQUEST \
        --no-cli-pager
    
    print_success "Table $table_name created successfully"
}

# Function to wait for table to be active
wait_for_table() {
    local table_name="$1"
    
    if [ "$DRY_RUN" = true ]; then
        return 0
    fi
    
    print_status "Waiting for table $table_name to be active..."
    $AWS_CMD dynamodb wait table-exists --table-name "$table_name"
    print_success "Table $table_name is now active"
}

# Main execution
print_status "Setting up DynamoDB tables in region: $REGION"
if [ -n "$PROFILE" ]; then
    print_status "Using AWS profile: $PROFILE"
fi

# Define services and their tables
declare -A SERVICES=(
    ["movimiento-inventario"]="movimiento-inventario-events movimiento-inventario-read"
    ["orden-compra"]="orden-compra-events orden-compra-read"
    ["proveedor"]="proveedor-events proveedor-read"
    ["ingreso-inventario"]="ingreso-inventario-events ingreso-inventario-read"
)

# Create tables for each service
for service in "${!SERVICES[@]}"; do
    print_status "Creating tables for service: $service"
    
    # Create events table
    create_table "${service}-events" \
        "AttributeName=id,KeyType=HASH AttributeName=timestamp,KeyType=RANGE" \
        "AttributeName=id,AttributeType=S AttributeName=timestamp,AttributeType=S"
    
    # Create read table
    create_table "${service}-read" \
        "AttributeName=id,KeyType=HASH" \
        "AttributeName=id,AttributeType=S"
done

# Wait for all tables to be active
if [ "$DRY_RUN" = false ]; then
    print_status "Waiting for all tables to be active..."
    for service in "${!SERVICES[@]}"; do
        wait_for_table "${service}-events"
        wait_for_table "${service}-read"
    done
fi

# List all created tables
print_status "Listing all tables in region $REGION:"
$AWS_CMD dynamodb list-tables --no-cli-pager

print_success "DynamoDB setup completed!"
print_warning "Remember to update your microservice configurations to use external DynamoDB"
print_warning "Use the switch-dynamodb.sh script to update the configurations"
