#!/bin/bash

# Knative Eventing Testing Script
# This script tests the Knative Eventing deployment and functionality

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

echo "🚀 Knative Eventing Testing Script"
echo "=================================="
echo ""

NAMESPACE="event-mesh-system"

# Check if Knative Eventing components are running
print_status "Checking Knative Eventing deployment status..."

# Check Knative Brokers
print_status "Checking Knative Brokers..."
BROKERS=$(kubectl get brokers -n ${NAMESPACE} -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")
if [[ -n "$BROKERS" ]]; then
    print_success "Knative Brokers found: $BROKERS"
    for broker in $BROKERS; do
        BROKER_STATUS=$(kubectl get broker $broker -n ${NAMESPACE} -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
        if [[ "$BROKER_STATUS" == "True" ]]; then
            print_success "  ✓ Broker $broker is Ready"
        else
            print_warning "  ⚠ Broker $broker status: $BROKER_STATUS"
        fi
    done
else
    print_error "No Knative Brokers found"
    exit 1
fi

# Check Knative Triggers
print_status "Checking Knative Triggers..."
TRIGGERS=$(kubectl get triggers -n ${NAMESPACE} -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")
if [[ -n "$TRIGGERS" ]]; then
    print_success "Knative Triggers found: $TRIGGERS"
    for trigger in $TRIGGERS; do
        TRIGGER_STATUS=$(kubectl get trigger $trigger -n ${NAMESPACE} -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
        if [[ "$TRIGGER_STATUS" == "True" ]]; then
            print_success "  ✓ Trigger $trigger is Ready"
        else
            print_warning "  ⚠ Trigger $trigger status: $TRIGGER_STATUS"
        fi
    done
else
    print_error "No Knative Triggers found"
    exit 1
fi

# Check Bridge Services
print_status "Checking Bridge Services..."

# Kafka to RabbitMQ Bridge
if kubectl get pods -n ${NAMESPACE} -l app=kafka-to-rabbitmq-bridge | grep -q "Running"; then
    print_success "Kafka to RabbitMQ Bridge is running"
    
    # Test health endpoint
    print_status "Testing Kafka to RabbitMQ Bridge health..."
    kubectl port-forward -n ${NAMESPACE} svc/kafka-to-rabbitmq-bridge 8080:8080 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 5
    
    HEALTH_STATUS=$(curl -s http://localhost:8080/health | jq -r '.status' 2>/dev/null || echo "error")
    
    if [[ "$HEALTH_STATUS" == "healthy" ]]; then
        print_success "Kafka to RabbitMQ Bridge health check: UP"
    else
        print_error "Kafka to RabbitMQ Bridge health check: DOWN. Status: ${HEALTH_STATUS}"
    fi
    kill $PF_PID 2>/dev/null || true
else
    print_error "Kafka to RabbitMQ Bridge is not running"
fi

# RabbitMQ to Pulsar Bridge
if kubectl get pods -n ${NAMESPACE} -l app=rabbitmq-to-pulsar-bridge | grep -q "Running"; then
    print_success "RabbitMQ to Pulsar Bridge is running"
    
    # Test health endpoint
    print_status "Testing RabbitMQ to Pulsar Bridge health..."
    kubectl port-forward -n ${NAMESPACE} svc/rabbitmq-to-pulsar-bridge 8081:8080 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 5
    
    HEALTH_STATUS=$(curl -s http://localhost:8081/health | jq -r '.status' 2>/dev/null || echo "error")
    
    if [[ "$HEALTH_STATUS" == "healthy" ]]; then
        print_success "RabbitMQ to Pulsar Bridge health check: UP"
    else
        print_error "RabbitMQ to Pulsar Bridge health check: DOWN. Status: ${HEALTH_STATUS}"
    fi
    kill $PF_PID 2>/dev/null || true
else
    print_error "RabbitMQ to Pulsar Bridge is not running"
fi

# Check Event Sources
print_status "Checking Event Sources..."

# RabbitMQ Event Source
if kubectl get pods -n ${NAMESPACE} -l app=rabbitmq-event-source | grep -q "Running"; then
    print_success "RabbitMQ Event Source is running"
else
    print_error "RabbitMQ Event Source is not running"
fi

# Pulsar Event Source
if kubectl get pods -n ${NAMESPACE} -l app=pulsar-event-source | grep -q "Running"; then
    print_success "Pulsar Event Source is running"
else
    print_error "Pulsar Event Source is not running"
fi

# Test Event Flow Simulation
print_status "Testing Event Flow Simulation..."

# Create a test CloudEvent
TEST_EVENT='{
  "specversion": "1.0",
  "type": "com.coldchain.stockbajo",
  "source": "test-script",
  "id": "test-event-001",
  "time": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'",
  "datacontenttype": "application/json",
  "data": {
    "id": "test-stock-low-001",
    "productId": "P001",
    "quantity": 5,
    "location": "WH1",
    "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
  }
}'

print_status "Sending test event to Kafka Broker..."
# Get the Kafka broker URL
KAFKA_BROKER_URL=$(kubectl get broker kafka-broker -n ${NAMESPACE} -o jsonpath='{.status.address.url}' 2>/dev/null || echo "")

if [[ -n "$KAFKA_BROKER_URL" ]]; then
    # Port forward to the broker
    kubectl port-forward -n ${NAMESPACE} svc/kafka-broker-kn-channel 8082:80 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 5
    
    # Send test event
    RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null -X POST \
        -H "Content-Type: application/json" \
        -d "$TEST_EVENT" \
        http://localhost:8082)
    
    if [[ "$RESPONSE" == "202" ]]; then
        print_success "Test event sent successfully to Kafka Broker (HTTP $RESPONSE)"
    else
        print_warning "Test event sent with status: HTTP $RESPONSE"
    fi
    
    kill $PF_PID 2>/dev/null || true
else
    print_warning "Kafka Broker URL not available, skipping event test"
fi

# Summary
echo ""
echo "📊 Knative Eventing Test Summary"
echo "================================"
echo ""

# Count running components
BROKER_COUNT=$(kubectl get brokers -n ${NAMESPACE} --no-headers 2>/dev/null | wc -l || echo "0")
TRIGGER_COUNT=$(kubectl get triggers -n ${NAMESPACE} --no-headers 2>/dev/null | wc -l || echo "0")
BRIDGE_COUNT=$(kubectl get pods -n ${NAMESPACE} -l app=kafka-to-rabbitmq-bridge,app=rabbitmq-to-pulsar-bridge --no-headers | grep "Running" | wc -l || echo "0")
SOURCE_COUNT=$(kubectl get pods -n ${NAMESPACE} -l app=rabbitmq-event-source,app=pulsar-event-source --no-headers | grep "Running" | wc -l || echo "0")

print_status "Components Status:"
echo "  • Knative Brokers: $BROKER_COUNT"
echo "  • Knative Triggers: $TRIGGER_COUNT"
echo "  • Bridge Services: $BRIDGE_COUNT/2"
echo "  • Event Sources: $SOURCE_COUNT/2"

if [[ "$BROKER_COUNT" -gt 0 && "$TRIGGER_COUNT" -gt 0 && "$BRIDGE_COUNT" -eq 2 && "$SOURCE_COUNT" -eq 2 ]]; then
    print_success "🎉 All Knative Eventing components are running successfully!"
    echo ""
    echo "Next steps:"
    echo "  • Test the complete event flow with: ./scripts/test-event-flow.sh"
    echo "  • Monitor events in Knative: kubectl get events -n event-mesh-system"
    echo "  • Check broker status: kubectl get brokers -n event-mesh-system"
    echo "  • Check trigger status: kubectl get triggers -n event-mesh-system"
else
    print_warning "⚠ Some components may not be fully ready yet."
    echo ""
    echo "Troubleshooting:"
    echo "  • Check pod status: kubectl get pods -n event-mesh-system"
    echo "  • Check broker status: kubectl get brokers -n event-mesh-system"
    echo "  • Check trigger status: kubectl get triggers -n event-mesh-system"
    echo "  • View logs: kubectl logs -n event-mesh-system -l app=kafka-to-rabbitmq-bridge"
fi

echo ""
print_success "Knative Eventing test completed!"
