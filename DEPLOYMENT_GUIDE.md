# Cold Chain Event Mesh - Deployment Guide

This guide explains how to deploy the Cold Chain Event Mesh system with both local and external DynamoDB configurations.

## Quick Start

### Local DynamoDB (Development)
```bash
./scripts/deploy.sh
```

### External DynamoDB (Production)
```bash
./scripts/deploy-external-dynamodb.sh
```

## Deployment Options

### 1. Full Deployment with Local DynamoDB
```bash
./scripts/deploy.sh
```
- Deploys all infrastructure components
- Uses DynamoDB Local for development
- Includes telemetry stack (Jaeger, Prometheus, Grafana)
- Deploys all microservices

### 2. Full Deployment with External DynamoDB
```bash
./scripts/deploy.sh --dynamodb external --region us-east-1 --access-key YOUR_KEY --secret-key YOUR_SECRET
```
- Deploys all infrastructure components (except DynamoDB Local)
- Uses AWS DynamoDB for production
- Includes telemetry stack
- Deploys all microservices

### 3. Interactive External DynamoDB Deployment
```bash
./scripts/deploy-external-dynamodb.sh
```
- Prompts for AWS credentials interactively
- Automatically configures external DynamoDB
- Deploys complete system

### 4. Selective Deployment
```bash
# Deploy only infrastructure
./scripts/deploy.sh --skip-microservices --skip-telemetry

# Deploy only microservices (infrastructure already exists)
./scripts/deploy.sh --skip-infrastructure --skip-telemetry

# Deploy without telemetry
./scripts/deploy.sh --skip-telemetry
```

## Prerequisites

### For Local DynamoDB
- Kubernetes cluster (Minikube, Docker Desktop, etc.)
- kubectl configured
- Docker images built and loaded

### For External DynamoDB
- All local prerequisites
- AWS account with DynamoDB access
- AWS credentials with appropriate permissions
- DynamoDB tables created (or use setup script)

## Building Docker Images

Before deployment, build and load the Docker images:

```bash
# Build Python service
cd movimiento-inventario
docker build -t movimiento-inventario:latest .
minikube image load movimiento-inventario:latest

# Build Go services
cd ../orden-compra
docker build -t orden-compra:latest .
minikube image load orden-compra:latest

cd ../proveedor
docker build -t proveedor:latest .
minikube image load proveedor:latest
```

## DynamoDB Configuration

### Local DynamoDB
- Automatically deployed with the system
- Uses dummy credentials
- Tables created automatically
- Perfect for development and testing

### External DynamoDB
- Requires AWS account
- Tables must be created manually or with setup script
- Uses real AWS credentials
- Production-ready

#### Creating DynamoDB Tables
```bash
# Create all required tables
./scripts/setup-aws-dynamodb.sh --region us-east-1

# Or create tables manually in AWS Console
```

#### Required Tables
- `movimiento-inventario-events` & `movimiento-inventario-read`
- `orden-compra-events` & `orden-compra-read`
- `proveedor-events` & `proveedor-read`
- `ingreso-inventario-events` & `ingreso-inventario-read`

## Infrastructure Components

### Message Brokers
- **Kafka**: Simple deployment with topic creation
- **RabbitMQ**: Cluster with queue/exchange setup
- **Pulsar**: Cluster with topic auto-creation

### Event Mesh
- **Simple Bridge**: Lightweight replacement for EventMesh
- Handles cross-broker event routing

### Database
- **DynamoDB Local**: For development (when using local)
- **AWS DynamoDB**: For production (when using external)

### Telemetry
- **Jaeger**: Distributed tracing
- **Prometheus**: Metrics collection
- **Grafana**: Visualization dashboard

## Microservices

### MovimientoInventario (Python)
- Consumes from Kafka: `FallaCadenaFrio`
- Produces to Kafka: `StockBajo`
- Uses DynamoDB for CQRS pattern

### OrdenCompra (Go)
- Consumes from RabbitMQ: `StockBajo`
- Produces to RabbitMQ: `RecepcionProveedor`
- Uses DynamoDB for CQRS pattern

### Proveedor (Go)
- Consumes from RabbitMQ: `RecepcionProveedor`
- Produces to RabbitMQ: `InventarioRecibido`
- Uses DynamoDB for CQRS pattern

### IngresoInventario (.NET)
- Consumes from Pulsar: `InventarioRecibido`
- Final processing step
- Uses DynamoDB for CQRS pattern

## Monitoring and Observability

### Accessing Services
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

## Testing the System

### Test Event Flow
```bash
./scripts/test-event-flow.sh
```

### Manual Testing
1. Send event to Kafka topic `FallaCadenaFrio`
2. Monitor processing through all services
3. Check DynamoDB tables for data
4. View traces in Jaeger

## Troubleshooting

### Common Issues

#### Pods Not Starting
```bash
# Check pod status
kubectl get pods -n event-mesh-system

# Check pod logs
kubectl logs -f deployment/movimiento-inventario -n event-mesh-system
```

#### DynamoDB Connection Issues
```bash
# For local DynamoDB
kubectl logs -f deployment/dynamodb-local -n event-mesh-system

# For external DynamoDB
# Check AWS credentials and region
# Verify table existence
aws dynamodb list-tables --region us-east-1
```

#### Kafka Connection Issues
```bash
# Check Kafka status
kubectl logs -f deployment/kafka-simple -n event-mesh-system

# List topics
kubectl exec -it deployment/kafka-simple -n event-mesh-system -- kafka-topics --list --bootstrap-server localhost:9092
```

### Resource Constraints
If you encounter resource issues:
1. Increase Minikube memory: `minikube config set memory 8192`
2. Restart Minikube: `minikube stop && minikube start`
3. Use selective deployment to reduce resource usage

## Security Considerations

### Local Development
- Uses dummy credentials
- No external network access required
- Suitable for development only

### Production Deployment
- Use IAM roles instead of access keys when possible
- Store credentials in AWS Secrets Manager
- Enable DynamoDB encryption at rest
- Use VPC endpoints for private network access
- Implement least privilege access

## Cleanup

### Remove All Resources
```bash
kubectl delete namespace event-mesh-system
```

### Remove Docker Images
```bash
minikube image rm movimiento-inventario:latest
minikube image rm orden-compra:latest
minikube image rm proveedor:latest
```

## Support

For issues or questions:
1. Check the logs: `kubectl logs -f deployment/SERVICE_NAME -n event-mesh-system`
2. Verify resource status: `kubectl get all -n event-mesh-system`
3. Check configuration: Review deployment YAML files
4. Test connectivity: Use the troubleshooting commands above
