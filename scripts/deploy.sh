#!/bin/bash

# Cold Chain Event Mesh Deployment Script
# This script deploys the complete event mesh architecture with configurable DynamoDB

set -e

# Default values
DYNAMODB_TYPE="local"
AWS_REGION="us-east-1"
AWS_ACCESS_KEY=""
AWS_SECRET_KEY=""
SKIP_INFRASTRUCTURE=false
SKIP_MICROSERVICES=false
SKIP_TELEMETRY=false

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
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --dynamodb TYPE        DynamoDB type: 'local' or 'external' (default: local)"
    echo "  --region REGION        AWS region for external DynamoDB (default: us-east-1)"
    echo "  --access-key KEY       AWS access key for external DynamoDB"
    echo "  --secret-key KEY       AWS secret key for external DynamoDB"
    echo "  --skip-infrastructure  Skip infrastructure deployment"
    echo "  --skip-microservices   Skip microservices deployment"
    echo "  --skip-telemetry       Skip telemetry infrastructure"
    echo "  --help                 Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Deploy with local DynamoDB"
    echo "  $0 --dynamodb external --region us-west-2 --access-key AKIA... --secret-key ..."
    echo "  $0 --skip-telemetry                   # Deploy without telemetry"
    echo "  $0 --skip-infrastructure --skip-telemetry  # Deploy only microservices"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dynamodb)
            DYNAMODB_TYPE="$2"
            shift 2
            ;;
        --region)
            AWS_REGION="$2"
            shift 2
            ;;
        --access-key)
            AWS_ACCESS_KEY="$2"
            shift 2
            ;;
        --secret-key)
            AWS_SECRET_KEY="$2"
            shift 2
            ;;
        --skip-infrastructure)
            SKIP_INFRASTRUCTURE=true
            shift
            ;;
        --skip-microservices)
            SKIP_MICROSERVICES=true
            shift
            ;;
        --skip-telemetry)
            SKIP_TELEMETRY=true
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

# Validate DynamoDB type
if [[ "$DYNAMODB_TYPE" != "local" && "$DYNAMODB_TYPE" != "external" ]]; then
    print_error "Invalid DynamoDB type: $DYNAMODB_TYPE. Must be 'local' or 'external'"
    exit 1
fi

# Validate external DynamoDB credentials
if [[ "$DYNAMODB_TYPE" == "external" ]]; then
    if [[ -z "$AWS_ACCESS_KEY" || -z "$AWS_SECRET_KEY" ]]; then
        print_error "AWS access key and secret key are required for external DynamoDB"
        print_error "Use --access-key and --secret-key options"
        exit 1
    fi
fi

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if kubectl can connect to cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    exit 1
fi

print_status "Connected to Kubernetes cluster"

# Function to configure DynamoDB
configure_dynamodb() {
    print_status "Configuring DynamoDB: $DYNAMODB_TYPE"
    
    if [[ "$DYNAMODB_TYPE" == "external" ]]; then
        print_status "Configuring external DynamoDB with region: $AWS_REGION"
        
        # Update Python service (movimiento-inventario)
        sed -i.bak '/DYNAMODB_ENDPOINT_URL/d' movimiento-inventario/k8s-deployment.yaml
        sed -i.bak "s|DYNAMODB_REGION.*|DYNAMODB_REGION: \"$AWS_REGION\"|" movimiento-inventario/k8s-deployment.yaml
        sed -i.bak "s|DYNAMODB_ACCESS_KEY.*|DYNAMODB_ACCESS_KEY: \"$AWS_ACCESS_KEY\"|" movimiento-inventario/k8s-deployment.yaml
        sed -i.bak "s|DYNAMODB_SECRET_KEY.*|DYNAMODB_SECRET_KEY: \"$AWS_SECRET_KEY\"|" movimiento-inventario/k8s-deployment.yaml
        
        # Update Go services (orden-compra, proveedor)
        for service in orden-compra proveedor; do
            sed -i.bak '/DYNAMODB_ENDPOINT/d' $service/k8s-deployment.yaml
            sed -i.bak "s|DYNAMODB_REGION.*|DYNAMODB_REGION: \"$AWS_REGION\"|" $service/k8s-deployment.yaml
            sed -i.bak "s|AWS_ACCESS_KEY_ID.*|AWS_ACCESS_KEY_ID: \"$AWS_ACCESS_KEY\"|" $service/k8s-deployment.yaml
            sed -i.bak "s|AWS_SECRET_ACCESS_KEY.*|AWS_SECRET_ACCESS_KEY: \"$AWS_SECRET_KEY\"|" $service/k8s-deployment.yaml
        done
        
        print_success "External DynamoDB configuration applied"
        print_warning "Make sure DynamoDB tables exist in AWS region: $AWS_REGION"
        
    else
        print_status "Using local DynamoDB configuration"
        # Local configuration is already set in the YAML files
    fi
}

# Configure DynamoDB before deployment
configure_dynamodb

# Create namespace
print_status "Creating namespace..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: event-mesh-system
EOF

print_success "Namespace created"

# Deploy infrastructure components
if [[ "$SKIP_INFRASTRUCTURE" == false ]]; then
    print_status "Deploying infrastructure components..."

    # Deploy Kafka
    print_status "Deploying Kafka cluster..."
    kubectl apply -f infrastructure/kafka/kafka-simple.yaml
    kubectl wait --for=condition=ready pod -l app=kafka-simple -n event-mesh-system --timeout=300s
    print_success "Kafka deployed"

    # Create Kafka topics
    print_status "Creating Kafka topics..."
    kubectl apply -f infrastructure/kafka/create-topics-job.yaml
    kubectl wait --for=condition=complete job/kafka-create-topics -n event-mesh-system --timeout=300s
    print_success "Kafka topics created"

    # Deploy RabbitMQ
    print_status "Deploying RabbitMQ cluster..."
    kubectl apply -f infrastructure/rabbitmq/rabbitmq-cluster.yaml
    kubectl wait --for=condition=ready pod -l app=rabbitmq -n event-mesh-system --timeout=300s
    print_success "RabbitMQ deployed"

    # Deploy Pulsar
    print_status "Deploying Pulsar cluster..."
    kubectl apply -f infrastructure/pulsar/pulsar-cluster.yaml
    kubectl wait --for=condition=ready pod -l app=pulsar -n event-mesh-system --timeout=300s
    print_success "Pulsar deployed"

    # Deploy DynamoDB Local (only if using local DynamoDB)
    if [[ "$DYNAMODB_TYPE" == "local" ]]; then
        print_status "Deploying DynamoDB Local..."
        kubectl apply -f infrastructure/dynamodb-local/dynamodb.yaml
        kubectl wait --for=condition=ready pod -l app=dynamodb-local -n event-mesh-system --timeout=300s
        print_success "DynamoDB Local deployed"

        # Wait for DynamoDB setup job
        print_status "Waiting for DynamoDB tables to be created..."
        kubectl wait --for=condition=complete job/dynamodb-setup -n event-mesh-system --timeout=300s
        print_success "DynamoDB tables created"
    else
        print_status "Skipping DynamoDB Local deployment (using external DynamoDB)"
    fi

    # Deploy Knative Eventing
    print_status "Deploying Knative Eventing..."
    kubectl apply -f infrastructure/knative/knative-install.yaml
    kubectl apply -f infrastructure/knative/knative-brokers.yaml
    kubectl apply -f infrastructure/knative/knative-sources.yaml
    kubectl apply -f infrastructure/knative/knative-triggers.yaml
    
    # Wait for Knative components to be ready
    print_status "Waiting for Knative Eventing to be ready..."
    kubectl wait --for=condition=ready pod -l app=kafka-to-rabbitmq-bridge -n event-mesh-system --timeout=120s
    kubectl wait --for=condition=ready pod -l app=rabbitmq-to-pulsar-bridge -n event-mesh-system --timeout=120s
    kubectl wait --for=condition=ready pod -l app=rabbitmq-event-source -n event-mesh-system --timeout=120s
    kubectl wait --for=condition=ready pod -l app=pulsar-event-source -n event-mesh-system --timeout=120s
    print_success "Knative Eventing deployed"
else
    print_status "Skipping infrastructure deployment"
fi

# Deploy telemetry infrastructure
if [[ "$SKIP_TELEMETRY" == false ]]; then
    print_status "Deploying telemetry infrastructure..."

    # Deploy Jaeger
    print_status "Deploying Jaeger..."
    kubectl apply -f telemetry/jaeger.yaml
    kubectl wait --for=condition=ready pod -l app=jaeger -n event-mesh-system --timeout=300s
    print_success "Jaeger deployed"

    # Deploy Prometheus
    print_status "Deploying Prometheus..."
    kubectl apply -f telemetry/prometheus.yaml
    kubectl wait --for=condition=ready pod -l app=prometheus -n event-mesh-system --timeout=300s
    print_success "Prometheus deployed"

    # Deploy Grafana
    print_status "Deploying Grafana..."
    kubectl apply -f telemetry/grafana.yaml
    kubectl wait --for=condition=ready pod -l app=grafana -n event-mesh-system --timeout=300s
    print_success "Grafana deployed"
else
    print_status "Skipping telemetry infrastructure deployment"
fi

# Deploy microservices
if [[ "$SKIP_MICROSERVICES" == false ]]; then
    print_status "Deploying microservices..."

    # Deploy MovimientoInventario
    print_status "Deploying MovimientoInventario service..."
    kubectl apply -f movimiento-inventario/k8s-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=movimiento-inventario -n event-mesh-system --timeout=300s
    print_success "MovimientoInventario deployed"

    # Deploy OrdenCompra
    print_status "Deploying OrdenCompra service..."
    kubectl apply -f orden-compra/k8s-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=orden-compra -n event-mesh-system --timeout=300s
    print_success "OrdenCompra deployed"

    # Deploy Proveedor
    print_status "Deploying Proveedor service..."
    kubectl apply -f proveedor/k8s-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=proveedor -n event-mesh-system --timeout=300s
    print_success "Proveedor deployed"

    # Deploy IngresoInventario
    print_status "Deploying IngresoInventario service..."
    kubectl apply -f ingreso-inventario/k8s-deployment.yaml
    kubectl wait --for=condition=ready pod -l app=ingreso-inventario -n event-mesh-system --timeout=300s
    print_success "IngresoInventario deployed"
else
    print_status "Skipping microservices deployment"
fi

# Display deployment summary
print_success "Deployment completed successfully!"

echo ""
echo "Deployment Summary:"
echo "==================="
echo "DynamoDB Type: $DYNAMODB_TYPE"
if [[ "$DYNAMODB_TYPE" == "external" ]]; then
    echo "AWS Region: $AWS_REGION"
fi
echo "Infrastructure: $([ "$SKIP_INFRASTRUCTURE" == true ] && echo "Skipped" || echo "Deployed")"
echo "Microservices: $([ "$SKIP_MICROSERVICES" == true ] && echo "Skipped" || echo "Deployed")"
echo "Telemetry: $([ "$SKIP_TELEMETRY" == true ] && echo "Skipped" || echo "Deployed")"
echo ""

if [[ "$SKIP_TELEMETRY" == false ]]; then
    echo "Service URLs:"
    echo "============="
    echo "Jaeger UI: http://localhost:16686"
    echo "Prometheus: http://localhost:9090"
    echo "Grafana: http://localhost:3000 (admin/admin)"
    echo ""
    
    echo "Port Forward Commands:"
    echo "======================"
    echo "kubectl port-forward -n event-mesh-system svc/jaeger-ui 16686:16686"
    echo "kubectl port-forward -n event-mesh-system svc/prometheus-service 9090:9090"
    echo "kubectl port-forward -n event-mesh-system svc/grafana-service 3000:3000"
    echo ""
fi

if [[ "$SKIP_MICROSERVICES" == false ]]; then
    echo "Test the Event Flow:"
    echo "===================="
    echo "Run: ./scripts/test-event-flow.sh"
    echo ""
fi

if [[ "$DYNAMODB_TYPE" == "external" ]]; then
    echo "External DynamoDB Setup:"
    echo "========================"
    echo "Make sure the following tables exist in AWS region $AWS_REGION:"
    echo "- movimiento-inventario-events, movimiento-inventario-read"
    echo "- orden-compra-events, orden-compra-read"
    echo "- proveedor-events, proveedor-read"
    echo "- ingreso-inventario-events, ingreso-inventario-read"
    echo ""
    echo "To create tables: ./scripts/setup-aws-dynamodb.sh --region $AWS_REGION"
    echo ""
fi

print_success "Cold Chain Event Mesh is ready!"
