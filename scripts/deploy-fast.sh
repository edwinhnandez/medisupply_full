#!/bin/bash

# Fast Deployment Script for Cold Chain Event Mesh
# This script deploys the system with a practical EventMesh bridge that starts quickly

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

echo "🚀 Fast Cold Chain Event Mesh Deployment"
echo "========================================"
echo ""

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
print_status "Deploying infrastructure components..."

# Deploy Kafka
print_status "Deploying Kafka cluster..."
kubectl apply -f infrastructure/kafka/kafka-simple.yaml
kubectl wait --for=condition=ready pod -l app=kafka-simple -n event-mesh-system --timeout=120s
print_success "Kafka deployed"

# Create Kafka topics
print_status "Creating Kafka topics..."
kubectl apply -f infrastructure/kafka/create-topics-job.yaml
kubectl wait --for=condition=complete job/kafka-create-topics -n event-mesh-system --timeout=120s
print_success "Kafka topics created"

# Deploy RabbitMQ
print_status "Deploying RabbitMQ cluster..."
kubectl apply -f infrastructure/rabbitmq/rabbitmq-cluster.yaml
kubectl wait --for=condition=ready pod -l app=rabbitmq -n event-mesh-system --timeout=120s
print_success "RabbitMQ deployed"

# Deploy Pulsar
print_status "Deploying Pulsar cluster..."
kubectl apply -f infrastructure/pulsar/pulsar-cluster.yaml
kubectl wait --for=condition=ready pod -l app=pulsar -n event-mesh-system --timeout=120s
print_success "Pulsar deployed"

# Deploy DynamoDB Local
print_status "Deploying DynamoDB Local..."
kubectl apply -f infrastructure/dynamodb-local/dynamodb.yaml
kubectl wait --for=condition=ready pod -l app=dynamodb-local -n event-mesh-system --timeout=120s
print_success "DynamoDB Local deployed"

# Wait for DynamoDB setup job
print_status "Waiting for DynamoDB tables to be created..."
kubectl wait --for=condition=complete job/dynamodb-setup -n event-mesh-system --timeout=120s
print_success "DynamoDB tables created"

# Deploy Knative Eventing (Fast)
print_status "Deploying Knative Eventing (Fast)..."
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

# Deploy microservices
print_status "Deploying microservices..."

# Deploy MovimientoInventario
print_status "Deploying MovimientoInventario service..."
kubectl apply -f movimiento-inventario/k8s-deployment.yaml
kubectl wait --for=condition=ready pod -l app=movimiento-inventario -n event-mesh-system --timeout=120s
print_success "MovimientoInventario deployed"

# Deploy OrdenCompra
print_status "Deploying OrdenCompra service..."
kubectl apply -f orden-compra/k8s-deployment.yaml
kubectl wait --for=condition=ready pod -l app=orden-compra -n event-mesh-system --timeout=120s
print_success "OrdenCompra deployed"

# Deploy Proveedor
print_status "Deploying Proveedor service..."
kubectl apply -f proveedor/k8s-deployment.yaml
kubectl wait --for=condition=ready pod -l app=proveedor -n event-mesh-system --timeout=120s
print_success "Proveedor deployed"

# Display deployment summary
print_success "Fast deployment completed successfully!"

echo ""
echo "Deployment Summary:"
echo "==================="
echo "✅ Kafka: Deployed and ready"
echo "✅ RabbitMQ: Deployed and ready"
echo "✅ Pulsar: Deployed and ready"
echo "✅ DynamoDB Local: Deployed and ready"
echo "✅ EventMesh Bridge: Deployed and ready"
echo "✅ MovimientoInventario: Deployed and ready"
echo "✅ OrdenCompra: Deployed and ready"
echo "✅ Proveedor: Deployed and ready"
echo ""

echo "Service URLs:"
echo "============="
echo "EventMesh Bridge Health: http://localhost:8080/health"
echo "EventMesh Bridge Metrics: http://localhost:8080/metrics"
echo ""

echo "Port Forward Commands:"
echo "======================"
echo "kubectl port-forward -n event-mesh-system svc/eventmesh-bridge-service 8080:8080"
echo ""

echo "Test Commands:"
echo "=============="
echo "curl http://localhost:8080/health"
echo "curl http://localhost:8080/metrics"
echo "./scripts/test-eventmesh-bridge.sh"
echo ""

echo "Event Flow:"
echo "==========="
echo "1. External System → Kafka (FallaCadenaFrio)"
echo "2. MovimientoInventario → Kafka (StockBajo)"
echo "3. EventMesh Bridge → RabbitMQ (StockBajo)"
echo "4. OrdenCompra → RabbitMQ (RecepcionProveedor)"
echo "5. Proveedor → RabbitMQ (InventarioRecibido)"
echo "6. EventMesh Bridge → Pulsar (InventarioRecibido)"
echo "7. IngresoInventario → Pulsar (completion)"
echo ""

print_success "Cold Chain Event Mesh is ready for testing!"
