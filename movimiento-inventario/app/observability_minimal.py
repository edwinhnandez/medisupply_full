import os
import logging
from typing import Dict, Any, Optional
import structlog
from pythonjsonlogger import jsonlogger


class ObservabilitySetup:
    """Minimal observability setup without problematic OpenTelemetry imports"""
    
    def __init__(self, service_name: str = "movimiento-inventario"):
        self.service_name = service_name
        self.logger = None
        self._setup_logging()
    
    def _setup_logging(self):
        """Setup structured logging"""
        try:
            # Configure structlog
            structlog.configure(
                processors=[
                    structlog.stdlib.filter_by_level,
                    structlog.stdlib.add_logger_name,
                    structlog.stdlib.add_log_level,
                    structlog.stdlib.PositionalArgumentsFormatter(),
                    structlog.processors.TimeStamper(fmt="iso"),
                    structlog.processors.StackInfoRenderer(),
                    structlog.processors.format_exc_info,
                    structlog.processors.UnicodeDecoder(),
                    structlog.processors.JSONRenderer()
                ],
                context_class=dict,
                logger_factory=structlog.stdlib.LoggerFactory(),
                wrapper_class=structlog.stdlib.BoundLogger,
                cache_logger_on_first_use=True,
            )
            
            # Setup standard logging
            logging.basicConfig(
                level=logging.INFO,
                format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            
            self.logger = structlog.get_logger(self.service_name)
            
        except Exception as e:
            logging.error(f"Failed to setup logging: {e}")
            self.logger = logging.getLogger(self.service_name)
    
    def get_logger(self):
        """Get structured logger"""
        return self.logger
    
    def create_span(self, name: str, attributes: Dict[str, Any] = None):
        """Create a dummy span (no-op for minimal version)"""
        return None
    
    def add_span_event(self, span, name: str, attributes: Dict[str, Any] = None):
        """Add event to span (no-op for minimal version)"""
        pass
    
    def set_span_status(self, span, status_code, description: str = None):
        """Set span status (no-op for minimal version)"""
        pass
    
    def finish_span(self, span):
        """Finish span (no-op for minimal version)"""
        pass


# Global observability instance
observability = ObservabilitySetup()

# Global metrics collector (simplified)
class MetricsCollector:
    def __init__(self):
        self.logger = observability.get_logger()
    
    def increment_counter(self, name: str, value: int = 1, attributes: Dict[str, Any] = None):
        """Increment a counter metric (log-based for minimal version)"""
        try:
            self.logger.info(f"Metric: {name} incremented by {value}", attributes=attributes or {})
        except Exception as e:
            logging.error(f"Failed to increment counter: {e}")
    
    def record_histogram(self, name: str, value: float, attributes: Dict[str, Any] = None):
        """Record a histogram metric (log-based for minimal version)"""
        try:
            self.logger.info(f"Metric: {name} recorded value {value}", attributes=attributes or {})
        except Exception as e:
            logging.error(f"Failed to record histogram: {e}")

metrics_collector = MetricsCollector()

# Global correlation context (simplified)
class CorrelationContext:
    def __init__(self):
        self.correlation_id = None
        self.causation_id = None
    
    def set_correlation_id(self, correlation_id: str):
        self.correlation_id = correlation_id
    
    def set_causation_id(self, causation_id: str):
        self.causation_id = causation_id
    
    def get_correlation_id(self) -> Optional[str]:
        return self.correlation_id
    
    def get_causation_id(self) -> Optional[str]:
        return self.causation_id

correlation_context = CorrelationContext()
