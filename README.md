# Cold Chain Event Mesh Architecture

A complex event-driven architecture integrating Kafka, RabbitMQ, and Pulsar with polyglot microservices using CQRS pattern, all running locally on Kubernetes with full observability.

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

## Services

- **MovimientoInventario** (Python): Kafka consumer/producer, processes cold chain failures
- **OrdenCompra** (Golang): RabbitMQ consumer/producer, handles purchase orders
- **Proveedor** (Golang): RabbitMQ consumer/producer, manages supplier interactions
- **IngresoInventario** (.NET): Pulsar consumer, final inventory processing

## Technologies

- **Message Brokers**: Kafka, RabbitMQ, Pulsar
- **Event Mesh**: Custom Python-based bridge for cross-broker routing
- **Database**: DynamoDB with CQRS pattern (supports both local and external)
- **Observability**: OpenTelemetry, Jaeger, Prometheus, Grafana
- **Orchestration**: Kubernetes

## Quick Start

### Prerequisites

- Kubernetes cluster (Minikube, Docker Desktop, or cloud provider)
- kubectl configured to access your cluster
- Docker for building images

### Deployment Options

#### Option 1: Full Deployment (Recommended)
```bash
# Deploy everything with external DynamoDB
./scripts/deploy.sh --dynamodb-type external --aws-region us-west-2 --aws-access-key YOUR_KEY --aws-secret-key YOUR_SECRET

# Deploy everything with local DynamoDB
./scripts/deploy.sh --dynamodb-type local
```

#### Option 2: Fast Deployment (Development)
```bash
# Quick deployment for testing
./scripts/deploy-fast.sh
```

#### Option 3: Manual Deployment
```bash
# 1. Create namespace
kubectl create namespace event-mesh-system

# 2. Deploy infrastructure
kubectl apply -f infrastructure/kafka/kafka-simple.yaml -n event-mesh-system
kubectl apply -f infrastructure/kafka/create-topics-job.yaml -n event-mesh-system
kubectl apply -f infrastructure/rabbitmq/rabbitmq-cluster.yaml -n event-mesh-system
kubectl apply -f infrastructure/pulsar/pulsar-cluster.yaml -n event-mesh-system
kubectl apply -f infrastructure/dynamodb-local/dynamodb.yaml -n event-mesh-system

# 3. Deploy EventMesh Bridge
kubectl apply -f infrastructure/event-mesh/eventmesh-bridge.yaml -n event-mesh-system

# 4. Deploy microservices
kubectl apply -f movimiento-inventario/k8s-deployment.yaml -n event-mesh-system
kubectl apply -f orden-compra/k8s-deployment.yaml -n event-mesh-system
kubectl apply -f proveedor/k8s-deployment.yaml -n event-mesh-system
kubectl apply -f ingreso-inventario/k8s-deployment.yaml -n event-mesh-system

# 5. Deploy observability
kubectl apply -f telemetry/jaeger.yaml -n event-mesh-system
kubectl apply -f telemetry/prometheus.yaml -n event-mesh-system
kubectl apply -f telemetry/grafana.yaml -n event-mesh-system
```

## EventMesh Bridge

The EventMesh Bridge is a lightweight Python-based service that provides cross-broker event routing:

- **Kafka to RabbitMQ**: Routes `StockBajo` events from Kafka to RabbitMQ
- **Health Monitoring**: HTTP endpoints for health checks and metrics
- **Resource Efficient**: Minimal CPU and memory requirements
- **Observability**: Prometheus metrics and structured logging

### Bridge Endpoints

- **Health Check**: `http://eventmesh-bridge-service:8080/health`
- **Metrics**: `http://eventmesh-bridge-service:8080/metrics`

## Testing

### Test Event Flow
```bash
# Test the complete event flow
./scripts/test-event-flow.sh

# Test EventMesh Bridge specifically
./scripts/test-eventmesh-bridge.sh
```

### Manual Testing
```bash
# Check pod status
kubectl get pods -n event-mesh-system

# Check EventMesh Bridge health
kubectl port-forward -n event-mesh-system svc/eventmesh-bridge-service 8080:8080
curl http://localhost:8080/health

# View logs
kubectl logs -n event-mesh-system -l app=eventmesh-bridge
```

## Configuration

### DynamoDB Configuration

The system supports both local and external DynamoDB:

- **Local**: Uses DynamoDB Local for development
- **External**: Connects to AWS DynamoDB with provided credentials

See `DYNAMODB_CONFIG.md` for detailed configuration options.

### Resource Requirements

The system is optimized for local development with minimal resource requirements:

- **EventMesh Bridge**: 25m CPU, 64Mi memory
- **Microservices**: 50m CPU, 64-128Mi memory each
- **Infrastructure**: Reduced resource requests for local clusters

## Development

Each service includes:
- CQRS implementation with separate commands and queries
- OpenTelemetry instrumentation (simplified for local development)
- Comprehensive error handling and retry mechanisms
- Idempotency for duplicate message handling
- Structured logging with correlation IDs

## Troubleshooting

### Common Issues

1. **Pod Scheduling Issues**: Check cluster resources with `kubectl describe nodes`
2. **EventMesh Bridge Timeout**: The bridge may take 2-3 minutes to start due to dependency installation
3. **DynamoDB Connection**: Ensure tables exist and credentials are correct
4. **Kafka Connectivity**: Verify Kafka topics are created successfully

### Logs and Debugging

```bash
# Check all pod status
kubectl get pods -n event-mesh-system

# View specific service logs
kubectl logs -n event-mesh-system -l app=movimiento-inventario
kubectl logs -n event-mesh-system -l app=eventmesh-bridge

# Check resource usage
kubectl top pods -n event-mesh-system
```

## Architecture Benefits

- **Polyglot Services**: Python, Golang, and .NET services
- **Event-Driven**: Asynchronous processing with message brokers
- **CQRS Pattern**: Separation of read and write operations
- **Observability**: Full tracing, metrics, and logging
- **Scalability**: Kubernetes-native deployment
- **Flexibility**: Support for both local and cloud deployments
