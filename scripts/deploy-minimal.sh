#!/bin/bash

# Minimal deployment script for resource-constrained environments
# This script deploys only the essential components to get a basic event mesh running

set -e

echo "Starting minimal Event Mesh deployment..."

# Create namespace
kubectl create namespace event-mesh-system --dry-run=client -o yaml | kubectl apply -f -

echo "Deploying minimal infrastructure components..."

# Deploy only RabbitMQ (lightest message broker)
kubectl apply -f infrastructure/rabbitmq/rabbitmq-cluster.yaml

echo "Waiting for RabbitMQ to be ready..."
kubectl wait --for=condition=ready pod -l app=rabbitmq -n event-mesh-system --timeout=300s

echo "RabbitMQ is ready!"

# Deploy DynamoDB Local
kubectl apply -f infrastructure/dynamodb-local/dynamodb.yaml

echo "Waiting for DynamoDB to be ready..."
kubectl wait --for=condition=ready pod -l app=dynamodb-local -n event-mesh-system --timeout=300s

echo "DynamoDB is ready!"

# Deploy only one microservice (movimiento-inventario) for testing
kubectl apply -f movimiento-inventario/k8s-deployment.yaml

echo "Waiting for movimiento-inventario to be ready..."
kubectl wait --for=condition=ready pod -l app=movimiento-inventario -n event-mesh-system --timeout=300s

echo "Movimiento Inventario is ready!"

# Deploy basic observability (only Jaeger)
kubectl apply -f telemetry/jaeger.yaml

echo "Waiting for Jaeger to be ready..."
kubectl wait --for=condition=ready pod -l app=jaeger -n event-mesh-system --timeout=300s

echo "Jaeger is ready!"

echo ""
echo "Minimal Event Mesh deployment completed!"
echo ""
echo "Current status:"
kubectl get pods -n event-mesh-system

echo ""
echo "Access URLs:"
echo "RabbitMQ Management: http://localhost:15672 (guest/guest)"
echo "Jaeger UI: http://localhost:16686"
echo ""
echo "To access services, run:"
echo "kubectl port-forward -n event-mesh-system svc/rabbitmq-service 15672:15672"
echo "kubectl port-forward -n event-mesh-system svc/jaeger 16686:16686"
echo ""
echo "To test the minimal setup:"
echo "1. Access RabbitMQ Management UI"
echo "2. Create a test queue"
echo "3. Send a test message"
echo "4. Check Jaeger for traces"