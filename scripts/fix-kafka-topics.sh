#!/bin/bash

set -e

echo "🔧 Fixing Kafka topics..."

# Wait for Kafka to be ready
echo "⏳ Waiting for Kafka to be ready..."
kubectl wait --for=condition=Ready pod -l app=kafka -n event-mesh-system --timeout=300s

# Delete the failing topic setup job
echo "🗑️ Deleting failing topic setup job..."
kubectl delete job kafka-topic-setup -n event-mesh-system --ignore-not-found=true

# Wait a bit for cleanup
sleep 10

# Create topics manually using kubectl exec
echo "📝 Creating Kafka topics manually..."

# Create FallaCadenaFrio topic
kubectl exec kafka-0 -n event-mesh-system -- kafka-topics.sh --create --topic FallaCadenaFrio --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1 || echo "Topic FallaCadenaFrio might already exist"

# Create StockBajo topic  
kubectl exec kafka-0 -n event-mesh-system -- kafka-topics.sh --create --topic StockBajo --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1 || echo "Topic StockBajo might already exist"

# List topics to verify
echo "📋 Listing Kafka topics..."
kubectl exec kafka-0 -n event-mesh-system -- kafka-topics.sh --list --bootstrap-server localhost:9092

echo "✅ Kafka topics setup complete!"
