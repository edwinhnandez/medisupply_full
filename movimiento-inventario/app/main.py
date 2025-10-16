import os
import signal
import sys
import time
from typing import Dict, Any
import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
import structlog

from .handlers import KafkaHandler, HealthCheckHandler
from .observability import observability, metrics_collector

logger = observability.get_logger()

# Global variables
kafka_handler = None
health_handler = None
app = FastAPI(
    title="Movimiento Inventario Service",
    description="Cold chain inventory movement service with CQRS pattern",
    version="1.0.0"
)


def get_config() -> Dict[str, Any]:
    """Get configuration from environment variables"""
    return {
        "kafka": {
            "bootstrap_servers": os.getenv("KAFKA_BOOTSTRAP_SERVERS", "kafka-broker:9092"),
            "input_topic": os.getenv("KAFKA_INPUT_TOPIC", "FallaCadenaFrio"),
            "output_topic": os.getenv("KAFKA_OUTPUT_TOPIC", "StockBajo"),
            "group_id": os.getenv("KAFKA_GROUP_ID", "movimiento-inventario-group"),
            "auto_offset_reset": os.getenv("KAFKA_AUTO_OFFSET_RESET", "earliest"),
            "enable_auto_commit": os.getenv("KAFKA_ENABLE_AUTO_COMMIT", "true").lower() == "true",
            "session_timeout_ms": int(os.getenv("KAFKA_SESSION_TIMEOUT_MS", "30000")),
            "heartbeat_interval_ms": int(os.getenv("KAFKA_HEARTBEAT_INTERVAL_MS", "10000")),
            "max_poll_interval_ms": int(os.getenv("KAFKA_MAX_POLL_INTERVAL_MS", "300000")),
            "acks": os.getenv("KAFKA_ACKS", "all"),
            "retries": int(os.getenv("KAFKA_RETRIES", "3")),
            "retry_backoff_ms": int(os.getenv("KAFKA_RETRY_BACKOFF_MS", "100")),
            "batch_size": int(os.getenv("KAFKA_BATCH_SIZE", "16384")),
            "linger_ms": int(os.getenv("KAFKA_LINGER_MS", "5")),
            "compression_type": os.getenv("KAFKA_COMPRESSION_TYPE", "snappy")
        },
        "dynamodb": {
            "endpoint_url": os.getenv("DYNAMODB_ENDPOINT_URL", "http://dynamodb-local:8000"),
            "region": os.getenv("DYNAMODB_REGION", "us-east-1"),
            "access_key": os.getenv("DYNAMODB_ACCESS_KEY", "dummy"),
            "secret_key": os.getenv("DYNAMODB_SECRET_KEY", "dummy")
        },
        "service": {
            "name": os.getenv("SERVICE_NAME", "movimiento-inventario"),
            "port": int(os.getenv("SERVICE_PORT", "8000")),
            "host": os.getenv("SERVICE_HOST", "0.0.0.0")
        }
    }


def setup_signal_handlers():
    """Setup signal handlers for graceful shutdown"""
    def signal_handler(signum, frame):
        logger.info("Received signal, shutting down gracefully", signal=signum)
        if kafka_handler:
            kafka_handler.stop_consuming()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)


def initialize_services():
    """Initialize all services"""
    global kafka_handler, health_handler
    
    try:
        config = get_config()
        
        # Initialize DynamoDB client for health checks
        import boto3
        dynamodb_client = boto3.client(
            "dynamodb",
            endpoint_url=config["dynamodb"]["endpoint_url"],
            region_name=config["dynamodb"]["region"],
            aws_access_key_id=config["dynamodb"]["access_key"],
            aws_secret_access_key=config["dynamodb"]["secret_key"]
        )
        
        # Initialize handlers
        kafka_handler = KafkaHandler(config["kafka"], config["dynamodb"])
        health_handler = HealthCheckHandler(dynamodb_client)
        
        logger.info("Services initialized successfully")
        
    except Exception as e:
        logger.error("Failed to initialize services", error=str(e))
        raise


@app.on_event("startup")
async def startup_event():
    """Application startup event"""
    try:
        logger.info("Starting Movimiento Inventario service")
        initialize_services()
        logger.info("Service started successfully")
    except Exception as e:
        logger.error("Failed to start service", error=str(e))
        raise


@app.on_event("shutdown")
async def shutdown_event():
    """Application shutdown event"""
    try:
        logger.info("Shutting down Movimiento Inventario service")
        if kafka_handler:
            kafka_handler.stop_consuming()
        logger.info("Service shut down successfully")
    except Exception as e:
        logger.error("Error during shutdown", error=str(e))


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    try:
        if not health_handler:
            raise HTTPException(status_code=503, detail="Health handler not initialized")
        
        health_status = health_handler.check_health()
        
        if health_status["status"] == "healthy":
            return JSONResponse(content=health_status, status_code=200)
        else:
            return JSONResponse(content=health_status, status_code=503)
            
    except Exception as e:
        logger.error("Health check failed", error=str(e))
        return JSONResponse(
            content={
                "status": "unhealthy",
                "timestamp": time.time(),
                "error": str(e)
            },
            status_code=503
        )


@app.get("/metrics")
async def metrics():
    """Metrics endpoint for Prometheus"""
    try:
        # This would typically return Prometheus metrics
        # For now, return a simple response
        return JSONResponse(content={
            "message": "Metrics endpoint",
            "timestamp": time.time()
        })
    except Exception as e:
        logger.error("Metrics endpoint failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/")
async def root():
    """Root endpoint"""
    return JSONResponse(content={
        "service": "Movimiento Inventario",
        "version": "1.0.0",
        "status": "running",
        "timestamp": time.time()
    })


def start_kafka_consumer():
    """Start Kafka consumer in a separate thread"""
    import threading
    
    def consumer_thread():
        try:
            if kafka_handler:
                kafka_handler.start_consuming()
        except Exception as e:
            logger.error("Kafka consumer thread failed", error=str(e))
    
    consumer_thread = threading.Thread(target=consumer_thread, daemon=True)
    consumer_thread.start()
    logger.info("Kafka consumer thread started")


def main():
    """Main entry point"""
    try:
        # Setup signal handlers
        setup_signal_handlers()
        
        # Get configuration
        config = get_config()
        
        # Start Kafka consumer
        start_kafka_consumer()
        
        # Start FastAPI server
        uvicorn.run(
            app,
            host=config["service"]["host"],
            port=config["service"]["port"],
            log_level="info",
            access_log=True
        )
        
    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt, shutting down")
    except Exception as e:
        logger.error("Service failed to start", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    main()
