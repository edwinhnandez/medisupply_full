from abc import ABC, abstractmethod
from typing import Dict, Any, Optional
from datetime import datetime
import uuid
import structlog

from ..models import (
    ColdChainFailureEvent, 
    StockLowEvent, 
    InventoryMovement, 
    Product,
    EventSourcingEvent
)

logger = structlog.get_logger()


class Command(ABC):
    """Base command interface for CQRS pattern"""
    
    @abstractmethod
    def execute(self) -> Dict[str, Any]:
        pass


class ProcessColdChainFailureCommand(Command):
    """Command to process cold chain failure events"""
    
    def __init__(self, event: ColdChainFailureEvent, dynamodb_client):
        self.event = event
        self.dynamodb = dynamodb_client
        self.table_name = "movimiento-inventario-events"
    
    def execute(self) -> Dict[str, Any]:
        """Process cold chain failure and determine if stock adjustment is needed"""
        try:
            logger.info("Processing cold chain failure", 
                       event_id=self.event.id, 
                       product_id=self.event.product_id,
                       temperature=self.event.temperature)
            
            # Store the event for audit trail
            self._store_event()
            
            # Check if product needs stock adjustment based on temperature failure
            stock_adjustment = self._calculate_stock_adjustment()
            
            if stock_adjustment:
                # Create inventory movement record
                movement = InventoryMovement(
                    product_id=self.event.product_id,
                    product_name=self.event.product_name,
                    movement_type="loss",
                    quantity=stock_adjustment["quantity"],
                    location=self.event.location,
                    reason=f"Cold chain failure - temperature: {self.event.temperature}°C",
                    metadata={
                        "failure_event_id": self.event.id,
                        "temperature": self.event.temperature,
                        "threshold": self.event.threshold_temperature,
                        "severity": self.event.severity
                    }
                )
                
                # Store inventory movement
                self._store_inventory_movement(movement)
                
                # Update product stock
                self._update_product_stock(movement)
                
                # Check if stock is now low
                if self._is_stock_low(movement.product_id):
                    stock_low_event = self._create_stock_low_event(movement)
                    return {
                        "success": True,
                        "stock_adjustment": stock_adjustment,
                        "stock_low_event": stock_low_event.dict(),
                        "correlation_id": self.event.id
                    }
            
            return {
                "success": True,
                "stock_adjustment": None,
                "stock_low_event": None,
                "correlation_id": self.event.id
            }
            
        except Exception as e:
            logger.error("Failed to process cold chain failure", 
                        event_id=self.event.id, 
                        error=str(e))
            raise
    
    def _store_event(self):
        """Store the cold chain failure event"""
        event_record = {
            "id": self.event.id,
            "timestamp": self.event.timestamp.isoformat(),
            "event_type": self.event.event_type,
            "product_id": self.event.product_id,
            "product_name": self.event.product_name,
            "temperature": self.event.temperature,
            "threshold_temperature": self.event.threshold_temperature,
            "location": self.event.location,
            "severity": self.event.severity,
            "metadata": self.event.metadata
        }
        
        self.dynamodb.put_item(
            TableName=self.table_name,
            Item=event_record
        )
    
    def _calculate_stock_adjustment(self) -> Optional[Dict[str, Any]]:
        """Calculate stock adjustment based on temperature failure severity"""
        severity_multipliers = {
            "low": 0.05,      # 5% loss
            "medium": 0.15,   # 15% loss
            "high": 0.30,    # 30% loss
            "critical": 0.50  # 50% loss
        }
        
        multiplier = severity_multipliers.get(self.event.severity, 0.10)
        
        # Get current product stock
        try:
            response = self.dynamodb.get_item(
                TableName="movimiento-inventario-read",
                Key={"id": self.event.product_id}
            )
            
            if "Item" in response:
                current_stock = response["Item"].get("current_stock", 0)
                loss_quantity = int(current_stock * multiplier)
                
                if loss_quantity > 0:
                    return {
                        "quantity": loss_quantity,
                        "reason": f"Temperature failure - {self.event.severity} severity",
                        "multiplier": multiplier
                    }
        except Exception as e:
            logger.warning("Could not calculate stock adjustment", error=str(e))
        
        return None
    
    def _store_inventory_movement(self, movement: InventoryMovement):
        """Store inventory movement record"""
        movement_record = {
            "id": movement.id,
            "product_id": movement.product_id,
            "product_name": movement.product_name,
            "movement_type": movement.movement_type,
            "quantity": movement.quantity,
            "location": movement.location,
            "timestamp": movement.timestamp.isoformat(),
            "reason": movement.reason,
            "metadata": movement.metadata
        }
        
        self.dynamodb.put_item(
            TableName="movimiento-inventario-events",
            Item=movement_record
        )
    
    def _update_product_stock(self, movement: InventoryMovement):
        """Update product stock in read model"""
        try:
            # Get current stock
            response = self.dynamodb.get_item(
                TableName="movimiento-inventario-read",
                Key={"id": movement.product_id}
            )
            
            current_stock = 0
            if "Item" in response:
                current_stock = response["Item"].get("current_stock", 0)
            
            # Calculate new stock
            new_stock = max(0, current_stock - movement.quantity)
            
            # Update stock
            self.dynamodb.put_item(
                TableName="movimiento-inventario-read",
                Item={
                    "id": movement.product_id,
                    "product_name": movement.product_name,
                    "current_stock": new_stock,
                    "last_updated": datetime.utcnow().isoformat(),
                    "metadata": {
                        "last_movement_id": movement.id,
                        "last_movement_type": movement.movement_type,
                        "last_movement_quantity": movement.quantity
                    }
                }
            )
            
        except Exception as e:
            logger.error("Failed to update product stock", 
                        product_id=movement.product_id, 
                        error=str(e))
            raise
    
    def _is_stock_low(self, product_id: str) -> bool:
        """Check if product stock is below minimum threshold"""
        try:
            response = self.dynamodb.get_item(
                TableName="movimiento-inventario-read",
                Key={"id": product_id}
            )
            
            if "Item" in response:
                current_stock = response["Item"].get("current_stock", 0)
                minimum_stock = response["Item"].get("minimum_stock", 0)
                return current_stock <= minimum_stock
                
        except Exception as e:
            logger.warning("Could not check stock level", 
                          product_id=product_id, 
                          error=str(e))
        
        return False
    
    def _create_stock_low_event(self, movement: InventoryMovement) -> StockLowEvent:
        """Create stock low event"""
        try:
            response = self.dynamodb.get_item(
                TableName="movimiento-inventario-read",
                Key={"id": movement.product_id}
            )
            
            if "Item" in response:
                item = response["Item"]
                current_stock = item.get("current_stock", 0)
                minimum_stock = item.get("minimum_stock", 0)
                
                # Determine urgency level
                urgency_level = "low"
                if current_stock == 0:
                    urgency_level = "critical"
                elif current_stock <= minimum_stock * 0.5:
                    urgency_level = "high"
                elif current_stock <= minimum_stock:
                    urgency_level = "medium"
                
                return StockLowEvent(
                    product_id=movement.product_id,
                    product_name=movement.product_name,
                    current_stock=current_stock,
                    minimum_stock=minimum_stock,
                    location=movement.location,
                    urgency_level=urgency_level,
                    metadata={
                        "last_movement_id": movement.id,
                        "cold_chain_failure_id": self.event.id,
                        "temperature": self.event.temperature
                    }
                )
                
        except Exception as e:
            logger.error("Failed to create stock low event", 
                        product_id=movement.product_id, 
                        error=str(e))
        
        # Fallback event
        return StockLowEvent(
            product_id=movement.product_id,
            product_name=movement.product_name,
            current_stock=0,
            minimum_stock=0,
            location=movement.location,
            urgency_level="critical",
            metadata={"error": "Could not determine stock levels"}
        )


class CreateProductCommand(Command):
    """Command to create a new product"""
    
    def __init__(self, product: Product, dynamodb_client):
        self.product = product
        self.dynamodb = dynamodb_client
    
    def execute(self) -> Dict[str, Any]:
        """Create a new product"""
        try:
            # Store in read model
            self.dynamodb.put_item(
                TableName="movimiento-inventario-read",
                Item={
                    "id": self.product.id,
                    "name": self.product.name,
                    "current_stock": self.product.current_stock,
                    "minimum_stock": self.product.minimum_stock,
                    "maximum_stock": self.product.maximum_stock,
                    "location": self.product.location,
                    "temperature_controlled": self.product.temperature_controlled,
                    "last_updated": self.product.last_updated.isoformat(),
                    "metadata": self.product.metadata
                }
            )
            
            # Store creation event
            event = EventSourcingEvent(
                aggregate_id=self.product.id,
                event_type="ProductCreated",
                event_data=self.product.dict(),
                correlation_id=str(uuid.uuid4())
            )
            
            self.dynamodb.put_item(
                TableName="movimiento-inventario-events",
                Item={
                    "id": event.id,
                    "timestamp": event.timestamp.isoformat(),
                    "aggregate_id": event.aggregate_id,
                    "event_type": event.event_type,
                    "event_data": event.event_data,
                    "version": event.version,
                    "correlation_id": event.correlation_id,
                    "causation_id": event.causation_id
                }
            )
            
            return {
                "success": True,
                "product_id": self.product.id,
                "correlation_id": event.correlation_id
            }
            
        except Exception as e:
            logger.error("Failed to create product", 
                        product_id=self.product.id, 
                        error=str(e))
            raise
