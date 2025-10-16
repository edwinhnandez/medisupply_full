# Cold Chain Event Mesh - Deployment Guide

This guide provides step-by-step instructions for deploying the complete Cold Chain Event Mesh architecture.

## Prerequisites

### Required Tools
- **Kubernetes Cluster**: Minikube, Docker Desktop, or cloud provider (GKE, EKS, AKS)
- **kubectl**: Kubernetes command-line tool
- **Docker**: For building container images
- **jq**: JSON processor (for testing scripts)

### System Requirements
- **CPU**: Minimum 4 cores
- **Memory**: Minimum 8GB RAM
- **Storage**: 20GB free space
- **Network**: Internet connection for pulling images

## Quick Start

### 1. Clone and Navigate
```bash
git clone <repository-url>
cd cold-chain-event-mesh
```

### 2. Deploy Everything
```bash
./scripts/deploy.sh
```

### 3. Test the Event Flow
```bash
./scripts/test-event-flow.sh
```

## Manual Deployment

If you prefer to deploy components manually:

### 1. Create Namespace
```bash
kubectl create namespace event-mesh-system
```

### 2. Deploy Infrastructure
```bash
# Deploy Kafka
kubectl apply -f infrastructure/kafka/kafka-cluster.yaml

# Deploy RabbitMQ
kubectl apply -f infrastructure/rabbitmq/rabbitmq-cluster.yaml

# Deploy Pulsar
kubectl apply -f infrastructure/pulsar/pulsar-cluster.yaml

# Deploy DynamoDB Local
kubectl apply -f infrastructure/dynamodb-local/dynamodb.yaml

# Wait for DynamoDB setup
kubectl wait --for=condition=complete job/dynamodb-setup -n event-mesh-system
```

### 3. Deploy EventMesh
```bash
kubectl apply -f infrastructure/event-mesh/eventmesh-configmap.yaml
kubectl apply -f infrastructure/event-mesh/eventmesh-bridge-config.yaml
kubectl apply -f infrastructure/event-mesh/eventmesh-deployment.yaml
```

### 4. Deploy Telemetry
```bash
# Deploy Jaeger
kubectl apply -f telemetry/jaeger.yaml

# Deploy Prometheus
kubectl apply -f telemetry/prometheus.yaml

# Deploy Grafana
kubectl apply -f telemetry/grafana.yaml
```

### 5. Deploy Microservices
```bash
# Deploy MovimientoInventario (Python)
kubectl apply -f movimiento-inventario/k8s-deployment.yaml

# Deploy OrdenCompra (Golang)
kubectl apply -f orden-compra/k8s-deployment.yaml

# Deploy Proveedor (Golang)
kubectl apply -f proveedor/k8s-deployment.yaml

# Deploy IngresoInventario (.NET)
kubectl apply -f ingreso-inventario/k8s-deployment.yaml
```

## Accessing Services

### Port Forwarding
```bash
# Jaeger UI
kubectl port-forward -n event-mesh-system svc/jaeger-ui 16686:16686

# Prometheus
kubectl port-forward -n event-mesh-system svc/prometheus-service 9090:9090

# Grafana
kubectl port-forward -n event-mesh-system svc/grafana-service 3000:3000
```

### Service URLs
- **Jaeger UI**: http://localhost:16686
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## Testing the Event Flow

### 1. Automated Test
```bash
./scripts/test-event-flow.sh
```

### 2. Manual Test
```bash
# Create a test event
cat <<EOF | kubectl exec -i -n event-mesh-system <kafka-pod> -- kafka-console-producer --bootstrap-server localhost:9092 --topic FallaCadenaFrio
{
  "id": "$(uuidgen)",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "event_type": "FallaCadenaFrio",
  "product_id": "PROD-001",
  "product_name": "COVID-19 Vaccine",
  "temperature": 8.5,
  "threshold_temperature": 2.0,
  "location": "Warehouse-001",
  "severity": "high",
  "metadata": {
    "sensor_id": "TEMP-001",
    "alert_level": "critical"
  }
}
EOF
```

## Monitoring and Observability

### Jaeger - Distributed Tracing
- View complete request traces across all services
- Analyze performance bottlenecks
- Debug event flow issues

### Prometheus - Metrics Collection
- Monitor service health
- Track business metrics
- Set up alerts

### Grafana - Visualization
- Pre-configured dashboards
- Real-time metrics
- Custom visualizations

## Troubleshooting

### Common Issues

#### 1. Pods Not Starting
```bash
# Check pod status
kubectl get pods -n event-mesh-system

# Check pod logs
kubectl logs -n event-mesh-system <pod-name>

# Check events
kubectl get events -n event-mesh-system
```

#### 2. Services Not Healthy
```bash
# Check service endpoints
kubectl get endpoints -n event-mesh-system

# Test health endpoints
kubectl port-forward -n event-mesh-system svc/<service-name> 8080:8000
curl http://localhost:8080/health
```

#### 3. Message Queue Issues
```bash
# Check Kafka topics
kubectl exec -n event-mesh-system <kafka-pod> -- kafka-topics --list --bootstrap-server localhost:9092

# Check RabbitMQ queues
kubectl exec -n event-mesh-system <rabbitmq-pod> -- rabbitmqctl list_queues

# Check Pulsar topics
kubectl exec -n event-mesh-system <pulsar-pod> -- bin/pulsar-admin topics list public/default
```

### Logs and Debugging

#### View Service Logs
```bash
# All services
kubectl logs -n event-mesh-system -l app=movimiento-inventario
kubectl logs -n event-mesh-system -l app=orden-compra
kubectl logs -n event-mesh-system -l app=proveedor
kubectl logs -n event-mesh-system -l app=ingreso-inventario
```

#### Follow Logs in Real-time
```bash
kubectl logs -n event-mesh-system -l app=movimiento-inventario -f
```

## Architecture Overview

```
External System → Kafka (FallaCadenaFrio) → MovimientoInventario (Python)
                                                      ↓
                                              Kafka (StockBajo)
                                                      ↓
                                              EventMesh Bridge
                                                      ↓
                                              RabbitMQ (StockBajo) → OrdenCompra (Golang)
                                                                           ↓
                                                                   RabbitMQ (RecepcionProveedor)
                                                                           ↓
                                                                   Proveedor (Golang)
                                                                           ↓
                                                                   RabbitMQ (InventarioRecibido)
                                                                           ↓
                                                                   EventMesh Bridge
                                                                           ↓
                                                                   Pulsar (InventarioRecibido) → IngresoInventario (.NET)
```

## Cleanup

To remove all resources:

```bash
kubectl delete namespace event-mesh-system
```

## Production Considerations

### Security
- Use proper secrets management
- Enable TLS for all communications
- Implement RBAC policies
- Use network policies

### Scalability
- Configure resource limits and requests
- Use horizontal pod autoscaling
- Implement proper load balancing
- Monitor resource usage

### Reliability
- Use persistent volumes for stateful services
- Implement proper health checks
- Set up monitoring and alerting
- Plan for disaster recovery

## Support

For issues and questions:
1. Check the troubleshooting section
2. Review service logs
3. Check the observability dashboards
4. Create an issue in the repository
