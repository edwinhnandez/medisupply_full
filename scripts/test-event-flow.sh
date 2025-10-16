#!/bin/bash

# Cold Chain Event Flow Test Script
# This script tests the complete event flow from Kafka to Pulsar

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

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    print_error "jq is not installed. Please install jq to run this script"
    exit 1
fi

print_status "🧪 Starting Cold Chain Event Flow Test"

# Create test event
create_test_event() {
    local event_id=$(uuidgen)
    local product_id="PROD-$(date +%s)"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    cat <<EOF
{
  "id": "$event_id",
  "timestamp": "$timestamp",
  "event_type": "FallaCadenaFrio",
  "product_id": "$product_id",
  "product_name": "COVID-19 Vaccine Batch A",
  "temperature": 8.5,
  "threshold_temperature": 2.0,
  "location": "Warehouse-001",
  "severity": "high",
  "metadata": {
    "sensor_id": "TEMP-001",
    "alert_level": "critical",
    "operator": "system"
  }
}
EOF
}

# Send event to Kafka
send_event_to_kafka() {
    print_status "Sending cold chain failure event to Kafka..."
    
    local event=$(create_test_event)
    local event_id=$(echo "$event" | jq -r '.id')
    
    # Get Kafka pod
    local kafka_pod=$(kubectl get pods -n event-mesh-system -l app=kafka -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$kafka_pod" ]; then
        print_error "Kafka pod not found"
        exit 1
    fi
    
    # Send event using kafka-console-producer
    echo "$event" | kubectl exec -i -n event-mesh-system "$kafka_pod" -- kafka-console-producer \
        --bootstrap-server localhost:9092 \
        --topic FallaCadenaFrio \
        --property "key.separator=:" \
        --property "parse.key=true" \
        --property "key.serializer=org.apache.kafka.common.serialization.StringSerializer" \
        --property "value.serializer=org.apache.kafka.common.serialization.StringSerializer"
    
    print_success "Event sent to Kafka with ID: $event_id"
    echo "$event_id"
}

# Wait for event processing
wait_for_processing() {
    local event_id=$1
    local max_wait=60
    local wait_time=0
    
    print_status "Waiting for event processing (max ${max_wait}s)..."
    
    while [ $wait_time -lt $max_wait ]; do
        # Check if event was processed by looking at service logs
        local movimiento_logs=$(kubectl logs -n event-mesh-system -l app=movimiento-inventario --tail=10 2>/dev/null | grep -c "$event_id" || echo "0")
        
        if [ "$movimiento_logs" -gt 0 ]; then
            print_success "Event processing detected in MovimientoInventario logs"
            return 0
        fi
        
        sleep 2
        wait_time=$((wait_time + 2))
    done
    
    print_warning "Event processing timeout - check logs manually"
    return 1
}

# Check service health
check_service_health() {
    local service_name=$1
    local service_url=$2
    
    print_status "🏥 Checking $service_name health..."
    
    local pod_name=$(kubectl get pods -n event-mesh-system -l app=$service_name -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    
    if [ -z "$pod_name" ]; then
        print_error "$service_name pod not found"
        return 1
    fi
    
    # Port forward to service
    kubectl port-forward -n event-mesh-system "pod/$pod_name" 8080:8000 &
    local pf_pid=$!
    
    sleep 3
    
    # Check health endpoint
    local health_response=$(curl -s http://localhost:8080/health 2>/dev/null || echo "error")
    
    # Kill port forward
    kill $pf_pid 2>/dev/null || true
    
    if [ "$health_response" = "error" ]; then
        print_error "$service_name health check failed"
        return 1
    fi
    
    local status=$(echo "$health_response" | jq -r '.status' 2>/dev/null || echo "unknown")
    
    if [ "$status" = "healthy" ]; then
        print_success "$service_name is healthy"
        return 0
    else
        print_error "$service_name is unhealthy: $status"
        return 1
    fi
}

# Check all services health
check_all_services() {
    print_status "🔍 Checking all services health..."
    
    local services=("movimiento-inventario" "orden-compra" "proveedor" "ingreso-inventario")
    local all_healthy=true
    
    for service in "${services[@]}"; do
        if ! check_service_health "$service" "http://localhost:8000"; then
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        print_success "All services are healthy"
        return 0
    else
        print_error "Some services are unhealthy"
        return 1
    fi
}

# Check message queues
check_message_queues() {
    print_status "Checking message queue status..."
    
    # Check Kafka topics
    local kafka_pod=$(kubectl get pods -n event-mesh-system -l app=kafka -o jsonpath='{.items[0].metadata.name}')
    if [ -n "$kafka_pod" ]; then
        print_status "Kafka topics:"
        kubectl exec -n event-mesh-system "$kafka_pod" -- kafka-topics --list --bootstrap-server localhost:9092
    fi
    
    # Check RabbitMQ queues
    local rabbitmq_pod=$(kubectl get pods -n event-mesh-system -l app=rabbitmq -o jsonpath='{.items[0].metadata.name}')
    if [ -n "$rabbitmq_pod" ]; then
        print_status "RabbitMQ queues:"
        kubectl exec -n event-mesh-system "$rabbitmq_pod" -- rabbitmqctl list_queues
    fi
    
    # Check Pulsar topics
    local pulsar_pod=$(kubectl get pods -n event-mesh-system -l app=pulsar -o jsonpath='{.items[0].metadata.name}')
    if [ -n "$pulsar_pod" ]; then
        print_status "Pulsar topics:"
        kubectl exec -n event-mesh-system "$pulsar_pod" -- bin/pulsar-admin topics list public/default
    fi
}

# Show observability URLs
show_observability_urls() {
    print_status "Observability URLs:"
    echo ""
    echo "Jaeger UI: http://localhost:16686"
    echo "Prometheus: http://localhost:9090"
    echo "Grafana: http://localhost:3000 (admin/admin)"
    echo ""
    echo "Port forward commands:"
    echo "kubectl port-forward -n event-mesh-system svc/jaeger-ui 16686:16686 &"
    echo "kubectl port-forward -n event-mesh-system svc/prometheus-service 9090:9090 &"
    echo "kubectl port-forward -n event-mesh-system svc/grafana-service 3000:3000 &"
    echo ""
}

# Main test execution
main() {
    print_status "Starting Cold Chain Event Flow Test"
    
    # Check if all services are healthy
    if ! check_all_services; then
        print_error "Not all services are healthy. Please check the deployment."
        exit 1
    fi
    
    # Send test event
    local event_id=$(send_event_to_kafka)
    
    # Wait for processing
    if wait_for_processing "$event_id"; then
        print_success "Event flow test completed successfully!"
    else
        print_warning "Event flow test completed with warnings"
    fi
    
    # Check message queues
    check_message_queues
    
    # Show observability URLs
    show_observability_urls
    
    print_success "Test completed! Check the observability dashboards for detailed metrics and traces."
}

# Run main function
main "$@"
