import json
import time
from typing import Dict, Any, Optional
from confluent_kafka import Consumer, Producer, KafkaError
import boto3
from botocore.exceptions import ClientError
import structlog

from .models import ColdChainFailureEvent, StockLowEvent
from .cqrs.commands import ProcessColdChainFailureCommand, CreateProductCommand
from .cqrs.queries import GetProductQuery, ListProductsQuery, GetInventoryMovementsQuery
from .observability import observability, metrics_collector, correlation_context

logger = observability.get_logger()


class KafkaHandler:
    """Handle Kafka message consumption and production"""
    
    def __init__(self, kafka_config: Dict[str, Any], dynamodb_config: Dict[str, Any]):
        self.kafka_config = kafka_config
        self.dynamodb_config = dynamodb_config
        self.dynamodb_client = self._create_dynamodb_client()
        self.consumer = self._create_consumer()
        self.producer = self._create_producer()
        self.running = False
    
    def _create_dynamodb_client(self):
        """Create DynamoDB client"""
        try:
            if self.dynamodb_config.get("endpoint_url"):
                return boto3.client(
                    "dynamodb",
                    endpoint_url=self.dynamodb_config["endpoint_url"],
                    region_name=self.dynamodb_config.get("region", "us-east-1"),
                    aws_access_key_id=self.dynamodb_config.get("access_key", "dummy"),
                    aws_secret_access_key=self.dynamodb_config.get("secret_key", "dummy")
                )
            else:
                return boto3.client(
                    "dynamodb",
                    region_name=self.dynamodb_config.get("region", "us-east-1")
                )
        except Exception as e:
            logger.error("Failed to create DynamoDB client", error=str(e))
            raise
    
    def _create_consumer(self):
        """Create Kafka consumer"""
        try:
            consumer_config = {
                "bootstrap.servers": self.kafka_config["bootstrap_servers"],
                "group.id": self.kafka_config.get("group_id", "movimiento-inventario-group"),
                "auto.offset.reset": self.kafka_config.get("auto_offset_reset", "earliest"),
                "enable.auto.commit": self.kafka_config.get("enable_auto_commit", True),
                "session.timeout.ms": self.kafka_config.get("session_timeout_ms", 30000),
                "heartbeat.interval.ms": self.kafka_config.get("heartbeat_interval_ms", 10000),
                "max.poll.interval.ms": self.kafka_config.get("max_poll_interval_ms", 300000)
            }
            
            consumer = Consumer(consumer_config)
            consumer.subscribe([self.kafka_config["input_topic"]])
            
            logger.info("Kafka consumer created", 
                       bootstrap_servers=self.kafka_config["bootstrap_servers"],
                       input_topic=self.kafka_config["input_topic"])
            
            return consumer
            
        except Exception as e:
            logger.error("Failed to create Kafka consumer", error=str(e))
            raise
    
    def _create_producer(self):
        """Create Kafka producer"""
        try:
            producer_config = {
                "bootstrap.servers": self.kafka_config["bootstrap_servers"],
                "acks": self.kafka_config.get("acks", "all"),
                "retries": self.kafka_config.get("retries", 3),
                "retry.backoff.ms": self.kafka_config.get("retry_backoff_ms", 100),
                "batch.size": self.kafka_config.get("batch_size", 16384),
                "linger.ms": self.kafka_config.get("linger_ms", 5),
                "compression.type": self.kafka_config.get("compression_type", "snappy")
            }
            
            producer = Producer(producer_config)
            
            logger.info("Kafka producer created", 
                       bootstrap_servers=self.kafka_config["bootstrap_servers"],
                       output_topic=self.kafka_config["output_topic"])
            
            return producer
            
        except Exception as e:
            logger.error("Failed to create Kafka producer", error=str(e))
            raise
    
    def start_consuming(self):
        """Start consuming messages from Kafka"""
        self.running = True
        logger.info("Starting Kafka consumer", input_topic=self.kafka_config["input_topic"])
        
        try:
            while self.running:
                msg = self.consumer.poll(timeout=1.0)
                
                if msg is None:
                    continue
                
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        logger.debug("Reached end of partition", 
                                   partition=msg.partition(),
                                   offset=msg.offset())
                        continue
                    else:
                        logger.error("Kafka consumer error", error=str(msg.error()))
                        metrics_collector.record_error("kafka_consumer_error", str(msg.error()))
                        continue
                
                # Process message
                self._process_message(msg)
                
        except KeyboardInterrupt:
            logger.info("Received interrupt signal, shutting down consumer")
        except Exception as e:
            logger.error("Unexpected error in consumer loop", error=str(e))
            metrics_collector.record_error("consumer_loop_error", str(e))
        finally:
            self.stop_consuming()
    
    def stop_consuming(self):
        """Stop consuming messages"""
        self.running = False
        if self.consumer:
            self.consumer.close()
        if self.producer:
            self.producer.flush()
        logger.info("Kafka consumer stopped")
    
    def _process_message(self, msg):
        """Process a single Kafka message"""
        start_time = time.time()
        
        try:
            # Extract message data
            message_key = msg.key().decode("utf-8") if msg.key() else None
            message_value = msg.value().decode("utf-8") if msg.value() else None
            message_headers = msg.headers() or []
            
            # Extract correlation ID from headers
            correlation_id = None
            causation_id = None
            for header in message_headers:
                if header[0] == "correlation-id":
                    correlation_id = header[1].decode("utf-8")
                elif header[0] == "causation-id":
                    causation_id = header[1].decode("utf-8")
            
            # Set correlation context
            correlation_context.set_correlation_id(correlation_id)
            correlation_context.set_causation_id(causation_id)
            
            logger.info("Processing message", 
                       topic=msg.topic(),
                       partition=msg.partition(),
                       offset=msg.offset(),
                       key=message_key,
                       correlation_id=correlation_id)
            
            # Parse message
            try:
                message_data = json.loads(message_value)
            except json.JSONDecodeError as e:
                logger.error("Failed to parse message JSON", 
                           error=str(e),
                           message_value=message_value)
                metrics_collector.record_error("json_parse_error", str(e))
                return
            
            # Create event object
            event = ColdChainFailureEvent(**message_data)
            
            # Process the event
            result = self._process_cold_chain_failure(event)
            
            # Record metrics
            processing_time = time.time() - start_time
            metrics_collector.record_processing_time(processing_time, "FallaCadenaFrio")
            metrics_collector.record_event_processed("FallaCadenaFrio", result["success"])
            
            # Produce output event if needed
            if result["success"] and result.get("stock_low_event"):
                self._produce_stock_low_event(result["stock_low_event"])
            
            logger.info("Message processed successfully", 
                       event_id=event.id,
                       processing_time=processing_time,
                       success=result["success"])
            
        except Exception as e:
            logger.error("Failed to process message", 
                        error=str(e),
                        topic=msg.topic(),
                        partition=msg.partition(),
                        offset=msg.offset())
            metrics_collector.record_error("message_processing_error", str(e))
            metrics_collector.record_event_processed("FallaCadenaFrio", False)
    
    def _process_cold_chain_failure(self, event: ColdChainFailureEvent) -> Dict[str, Any]:
        """Process cold chain failure event"""
        try:
            # Create and execute command
            command = ProcessColdChainFailureCommand(event, self.dynamodb_client)
            result = command.execute()
            
            return result
            
        except Exception as e:
            logger.error("Failed to process cold chain failure", 
                        event_id=event.id,
                        error=str(e))
            return {
                "success": False,
                "error": str(e),
                "correlation_id": event.id
            }
    
    def _produce_stock_low_event(self, stock_low_event: Dict[str, Any]):
        """Produce stock low event to output topic"""
        try:
            # Prepare message
            message_key = stock_low_event["product_id"]
            message_value = json.dumps(stock_low_event)
            
            # Prepare headers
            headers = [
                ("correlation-id", correlation_context.correlation_id.encode("utf-8") if correlation_context.correlation_id else b""),
                ("causation-id", stock_low_event["id"].encode("utf-8")),
                ("event-type", b"StockBajo"),
                ("content-type", b"application/json")
            ]
            
            # Produce message
            self.producer.produce(
                topic=self.kafka_config["output_topic"],
                key=message_key.encode("utf-8") if message_key else None,
                value=message_value.encode("utf-8"),
                headers=headers,
                callback=self._delivery_callback
            )
            
            logger.info("Stock low event produced", 
                       product_id=stock_low_event["product_id"],
                       topic=self.kafka_config["output_topic"])
            
        except Exception as e:
            logger.error("Failed to produce stock low event", 
                        error=str(e),
                        product_id=stock_low_event.get("product_id"))
            metrics_collector.record_error("produce_error", str(e))
    
    def _delivery_callback(self, err, msg):
        """Callback for message delivery confirmation"""
        if err:
            logger.error("Message delivery failed", error=str(err))
            metrics_collector.record_error("delivery_error", str(err))
        else:
            logger.debug("Message delivered successfully", 
                        topic=msg.topic(),
                        partition=msg.partition(),
                        offset=msg.offset())


class HealthCheckHandler:
    """Handle health check requests"""
    
    def __init__(self, dynamodb_client):
        self.dynamodb_client = dynamodb_client
    
    def check_health(self) -> Dict[str, Any]:
        """Check service health"""
        try:
            # Check DynamoDB connection
            self.dynamodb_client.describe_table(TableName="movimiento-inventario-read")
            
            return {
                "status": "healthy",
                "timestamp": time.time(),
                "checks": {
                    "dynamodb": "ok"
                }
            }
            
        except ClientError as e:
            logger.error("Health check failed - DynamoDB", error=str(e))
            return {
                "status": "unhealthy",
                "timestamp": time.time(),
                "checks": {
                    "dynamodb": "error"
                },
                "error": str(e)
            }
        except Exception as e:
            logger.error("Health check failed", error=str(e))
            return {
                "status": "unhealthy",
                "timestamp": time.time(),
                "checks": {
                    "dynamodb": "unknown"
                },
                "error": str(e)
            }
