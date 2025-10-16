# Apache EventMesh Integration Guide

This guide explains the Apache EventMesh integration in the Cold Chain Event Mesh system, replacing the simple bridge with a full-featured event mesh solution.

## Overview

Apache EventMesh is a serverless event middleware designed for building distributed event-driven applications. It provides:

- **Multi-protocol Support**: TCP, HTTP, and gRPC
- **CloudEvents Compliance**: Standardized event format
- **Cross-broker Bridging**: Seamless integration between different message brokers
- **High Performance**: Optimized for high-throughput event processing
- **Observability**: Built-in metrics and monitoring

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Kafka       │    │   RabbitMQ      │    │     Pulsar      │
│                 │    │                 │    │                 │
│ FallaCadenaFrio │    │ StockBajo       │    │ InventarioRecibido│
│ StockBajo       │    │ RecepcionProveedor│   │                 │
└─────────────────┘    │ InventarioRecibido│   └─────────────────┘
         │              └─────────────────┘              │
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Apache EventMesh│
                    │                 │
                    │ • Kafka Bridge  │
                    │ • RabbitMQ Bridge│
                    │ • Pulsar Bridge │
                    │ • Event Routing │
                    └─────────────────┘
```

## Event Flow

1. **External System** → **Kafka** (`FallaCadenaFrio`)
2. **MovimientoInventario** processes event and produces to **Kafka** (`StockBajo`)
3. **EventMesh** bridges `StockBajo` from **Kafka** to **RabbitMQ**
4. **OrdenCompra** consumes from **RabbitMQ** and produces `RecepcionProveedor`
5. **Proveedor** consumes from **RabbitMQ** and produces `InventarioRecibido`
6. **EventMesh** bridges `InventarioRecibido` from **RabbitMQ** to **Pulsar**
7. **IngresoInventario** consumes from **Pulsar** and completes the workflow

## Configuration

### EventMesh Configuration

The EventMesh configuration is defined in `infrastructure/event-mesh/eventmesh-config-working.yaml`:

#### Application Configuration
```properties
# EventMesh Server Ports
eventmesh.server.port=10000
eventmesh.http.server.port=10105
eventmesh.grpc.server.port=10205

# Registry Configuration (disabled for standalone)
eventmesh.server.registry.enable=false

# Store Configuration
eventmesh.server.store.pluginType=standalone
eventmesh.server.store.standalone.selectorType=file
eventmesh.server.store.standalone.file.filePath=/opt/eventmesh/data

# Metrics Configuration
eventmesh.server.metrics.enable=true
eventmesh.server.metrics.prometheus.port=9091
```

#### Connector Configuration
```properties
# Kafka Connector
eventmesh.connector.kafka.enable=true
eventmesh.connector.kafka.bootstrap.servers=kafka-simple:9092
eventmesh.connector.kafka.group.id=eventmesh-kafka

# RabbitMQ Connector
eventmesh.connector.rabbitmq.enable=true
eventmesh.connector.rabbitmq.host=rabbitmq-service
eventmesh.connector.rabbitmq.port=5672
eventmesh.connector.rabbitmq.username=guest
eventmesh.connector.rabbitmq.password=guest

# Pulsar Connector
eventmesh.connector.pulsar.enable=true
eventmesh.connector.pulsar.serviceUrl=pulsar://pulsar-broker:6650
```

#### Bridge Configuration
```properties
# Kafka to RabbitMQ Bridge
eventmesh.bridge.kafka-to-rabbitmq.enable=true
eventmesh.bridge.kafka-to-rabbitmq.source.topic=StockBajo
eventmesh.bridge.kafka-to-rabbitmq.target.exchange=stock-bajo-exchange
eventmesh.bridge.kafka-to-rabbitmq.target.routingKey=stock.bajo

# RabbitMQ to Pulsar Bridge
eventmesh.bridge.rabbitmq-to-pulsar.enable=true
eventmesh.bridge.rabbitmq-to-pulsar.source.queue=inventario-recibido-queue
eventmesh.bridge.rabbitmq-to-pulsar.target.topic=InventarioRecibido
```

## Deployment

### Using the Deploy Script
```bash
# Deploy with EventMesh
./scripts/deploy.sh

# Deploy with external DynamoDB and EventMesh
./scripts/deploy.sh --dynamodb external --region us-west-2 --access-key YOUR_KEY --secret-key YOUR_SECRET
```

### Manual Deployment
```bash
# Deploy EventMesh configuration
kubectl apply -f infrastructure/event-mesh/eventmesh-config-working.yaml

# Deploy EventMesh runtime
kubectl apply -f infrastructure/event-mesh/eventmesh-deployment-working.yaml

# Wait for EventMesh to be ready
kubectl wait --for=condition=ready pod -l app=eventmesh -n event-mesh-system --timeout=300s
```

## Testing

### Test EventMesh Deployment
```bash
./scripts/test-eventmesh.sh
```

### Manual Testing
```bash
# Check EventMesh status
kubectl get pods -n event-mesh-system -l app=eventmesh

# Check EventMesh logs
kubectl logs -n event-mesh-system deployment/eventmesh

# Test EventMesh health endpoint
kubectl port-forward -n event-mesh-system svc/eventmesh-service 10105:10105
curl http://localhost:10105/health

# Test EventMesh metrics
kubectl port-forward -n event-mesh-system svc/eventmesh-service 9091:9091
curl http://localhost:9091/metrics
```

## Monitoring

### EventMesh Metrics
EventMesh exposes Prometheus metrics on port 9091:

- **Event Processing Metrics**: Throughput, latency, error rates
- **Bridge Metrics**: Message transfer rates, bridge health
- **System Metrics**: Memory usage, thread pools, connection counts

### Health Checks
EventMesh provides health check endpoints:

- **HTTP Health**: `http://eventmesh-service:10105/health`
- **gRPC Health**: `grpc://eventmesh-service:10205/health`
- **TCP Health**: `tcp://eventmesh-service:10000`

### Logging
EventMesh logs are available in the pod:

```bash
kubectl logs -n event-mesh-system deployment/eventmesh
```

## API Usage

### HTTP API
```bash
# Send event via HTTP
curl -X POST http://eventmesh-service:10105/eventmesh \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "StockBajo",
    "data": "{\"productId\": \"123\", \"quantity\": 100}",
    "eventType": "inventory.stock.low"
  }'

# Subscribe to events via HTTP
curl -X POST http://eventmesh-service:10105/eventmesh/subscribe \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "StockBajo",
    "url": "http://orden-compra-service:8000/events"
  }'
```

### gRPC API
```bash
# Send event via gRPC
grpcurl -plaintext -d '{
  "topic": "StockBajo",
  "data": "{\"productId\": \"123\", \"quantity\": 100}",
  "eventType": "inventory.stock.low"
}' eventmesh-service:10205 eventmesh.EventMeshService/Publish
```

## Troubleshooting

### Common Issues

#### EventMesh Not Starting
```bash
# Check pod status
kubectl describe pod -n event-mesh-system -l app=eventmesh

# Check logs
kubectl logs -n event-mesh-system deployment/eventmesh

# Check configuration
kubectl exec -n event-mesh-system deployment/eventmesh -- cat /opt/eventmesh/conf/application.properties
```

#### Bridge Not Working
```bash
# Check bridge configuration
kubectl exec -n event-mesh-system deployment/eventmesh -- cat /opt/eventmesh/conf/bridge.properties

# Check connector configuration
kubectl exec -n event-mesh-system deployment/eventmesh -- cat /opt/eventmesh/conf/connector.properties

# Test broker connectivity
kubectl exec -n event-mesh-system deployment/eventmesh -- nc -z kafka-simple 9092
kubectl exec -n event-mesh-system deployment/eventmesh -- nc -z rabbitmq-service 5672
kubectl exec -n event-mesh-system deployment/eventmesh -- nc -z pulsar-broker 6650
```

#### Performance Issues
```bash
# Check metrics
kubectl port-forward -n event-mesh-system svc/eventmesh-service 9091:9091
curl http://localhost:9091/metrics | grep eventmesh

# Check resource usage
kubectl top pod -n event-mesh-system -l app=eventmesh
```

### Configuration Tuning

#### Performance Tuning
```properties
# Increase thread pools
eventmesh.server.threads.publicExecutor=16
eventmesh.server.threads.privateExecutor=16

# Increase batch size
eventmesh.server.msgBatchSize=1000

# Adjust retry settings
eventmesh.server.msgSendMaxRetryTime=5
eventmesh.server.msgRetryDelay=500
```

#### Memory Tuning
```properties
# Adjust JVM settings
JAVA_OPTS="-Xms512m -Xmx1024m -XX:+UseG1GC -XX:MaxGCPauseMillis=10"
```

## Security

### Authentication
EventMesh supports authentication via tokens:

```properties
eventmesh.server.auth.enable=true
eventmesh.server.auth.tokenPath=/opt/eventmesh/conf/token.properties
```

### SSL/TLS
EventMesh supports SSL/TLS encryption:

```properties
eventmesh.server.ssl.enable=true
eventmesh.server.ssl.server.certPath=/opt/eventmesh/conf/server.crt
eventmesh.server.ssl.server.keyPath=/opt/eventmesh/conf/server.key
```

## Best Practices

1. **Resource Allocation**: Allocate sufficient memory and CPU for EventMesh
2. **Monitoring**: Enable metrics and set up alerts
3. **Logging**: Configure appropriate log levels
4. **Security**: Enable authentication and SSL in production
5. **Backup**: Regular backup of EventMesh configuration and data
6. **Testing**: Test bridge functionality thoroughly before production

## Support

For issues or questions:
1. Check EventMesh logs: `kubectl logs -n event-mesh-system deployment/eventmesh`
2. Test connectivity: Use the test script `./scripts/test-eventmesh.sh`
3. Check configuration: Verify all configuration files are correct
4. Monitor metrics: Use Prometheus metrics for performance analysis
