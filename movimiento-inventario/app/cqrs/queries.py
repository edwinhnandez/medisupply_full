from abc import ABC, abstractmethod
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
import structlog

from ..models import Product, InventoryMovement, ColdChainFailureEvent, StockLowEvent

logger = structlog.get_logger()


class Query(ABC):
    """Base query interface for CQRS pattern"""
    
    @abstractmethod
    def execute(self) -> Dict[str, Any]:
        pass


class GetProductQuery(Query):
    """Query to get a single product by ID"""
    
    def __init__(self, product_id: str, dynamodb_client):
        self.product_id = product_id
        self.dynamodb = dynamodb_client
    
    def execute(self) -> Dict[str, Any]:
        """Get product by ID"""
        try:
            response = self.dynamodb.get_item(
                TableName="movimiento-inventario-read",
                Key={"id": self.product_id}
            )
            
            if "Item" in response:
                return {
                    "success": True,
                    "product": response["Item"]
                }
            else:
                return {
                    "success": False,
                    "error": "Product not found",
                    "product_id": self.product_id
                }
                
        except Exception as e:
            logger.error("Failed to get product", 
                        product_id=self.product_id, 
                        error=str(e))
            return {
                "success": False,
                "error": str(e),
                "product_id": self.product_id
            }


class ListProductsQuery(Query):
    """Query to list all products with optional filtering"""
    
    def __init__(self, dynamodb_client, location: Optional[str] = None, 
                 low_stock_only: bool = False, limit: int = 100):
        self.dynamodb = dynamodb_client
        self.location = location
        self.low_stock_only = low_stock_only
        self.limit = limit
    
    def execute(self) -> Dict[str, Any]:
        """List products with optional filtering"""
        try:
            # Build scan parameters
            scan_params = {
                "TableName": "movimiento-inventario-read",
                "Limit": self.limit
            }
            
            # Add filter expressions if needed
            filter_expressions = []
            expression_attribute_values = {}
            expression_attribute_names = {}
            
            if self.location:
                filter_expressions.append("#location = :location")
                expression_attribute_names["#location"] = "location"
                expression_attribute_values[":location"] = self.location
            
            if self.low_stock_only:
                filter_expressions.append("#current_stock <= #minimum_stock")
                expression_attribute_names["#current_stock"] = "current_stock"
                expression_attribute_names["#minimum_stock"] = "minimum_stock"
            
            if filter_expressions:
                scan_params["FilterExpression"] = " AND ".join(filter_expressions)
                scan_params["ExpressionAttributeNames"] = expression_attribute_names
                scan_params["ExpressionAttributeValues"] = expression_attribute_values
            
            response = self.dynamodb.scan(**scan_params)
            
            products = response.get("Items", [])
            
            # Apply additional filtering for low stock if needed
            if self.low_stock_only:
                products = [
                    p for p in products 
                    if p.get("current_stock", 0) <= p.get("minimum_stock", 0)
                ]
            
            return {
                "success": True,
                "products": products,
                "count": len(products)
            }
            
        except Exception as e:
            logger.error("Failed to list products", error=str(e))
            return {
                "success": False,
                "error": str(e),
                "products": []
            }


class GetInventoryMovementsQuery(Query):
    """Query to get inventory movements for a product"""
    
    def __init__(self, product_id: str, dynamodb_client, 
                 start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None,
                 movement_type: Optional[str] = None,
                 limit: int = 100):
        self.product_id = product_id
        self.dynamodb = dynamodb_client
        self.start_date = start_date
        self.end_date = end_date
        self.movement_type = movement_type
        self.limit = limit
    
    def execute(self) -> Dict[str, Any]:
        """Get inventory movements for a product"""
        try:
            # Build query parameters
            query_params = {
                "TableName": "movimiento-inventario-events",
                "IndexName": "product-timestamp-index",  # Assuming GSI exists
                "KeyConditionExpression": "#product_id = :product_id",
                "ExpressionAttributeNames": {
                    "#product_id": "product_id"
                },
                "ExpressionAttributeValues": {
                    ":product_id": self.product_id
                },
                "ScanIndexForward": False,  # Most recent first
                "Limit": self.limit
            }
            
            # Add date range filtering
            if self.start_date or self.end_date:
                date_conditions = []
                if self.start_date:
                    date_conditions.append("#timestamp >= :start_date")
                    query_params["ExpressionAttributeValues"][":start_date"] = self.start_date.isoformat()
                if self.end_date:
                    date_conditions.append("#timestamp <= :end_date")
                    query_params["ExpressionAttributeValues"][":end_date"] = self.end_date.isoformat()
                
                if date_conditions:
                    query_params["FilterExpression"] = " AND ".join(date_conditions)
                    query_params["ExpressionAttributeNames"]["#timestamp"] = "timestamp"
            
            # Add movement type filtering
            if self.movement_type:
                if "FilterExpression" in query_params:
                    query_params["FilterExpression"] += " AND #movement_type = :movement_type"
                else:
                    query_params["FilterExpression"] = "#movement_type = :movement_type"
                query_params["ExpressionAttributeNames"]["#movement_type"] = "movement_type"
                query_params["ExpressionAttributeValues"][":movement_type"] = self.movement_type
            
            response = self.dynamodb.query(**query_params)
            
            movements = response.get("Items", [])
            
            return {
                "success": True,
                "movements": movements,
                "count": len(movements)
            }
            
        except Exception as e:
            logger.error("Failed to get inventory movements", 
                        product_id=self.product_id, 
                        error=str(e))
            return {
                "success": False,
                "error": str(e),
                "movements": []
            }


class GetColdChainFailuresQuery(Query):
    """Query to get cold chain failure events"""
    
    def __init__(self, dynamodb_client, 
                 product_id: Optional[str] = None,
                 severity: Optional[str] = None,
                 start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None,
                 limit: int = 100):
        self.dynamodb = dynamodb_client
        self.product_id = product_id
        self.severity = severity
        self.start_date = start_date
        self.end_date = end_date
        self.limit = limit
    
    def execute(self) -> Dict[str, Any]:
        """Get cold chain failure events"""
        try:
            # Build scan parameters
            scan_params = {
                "TableName": "movimiento-inventario-events",
                "FilterExpression": "#event_type = :event_type",
                "ExpressionAttributeNames": {
                    "#event_type": "event_type"
                },
                "ExpressionAttributeValues": {
                    ":event_type": "FallaCadenaFrio"
                },
                "Limit": self.limit
            }
            
            # Add additional filters
            additional_filters = []
            
            if self.product_id:
                additional_filters.append("#product_id = :product_id")
                scan_params["ExpressionAttributeNames"]["#product_id"] = "product_id"
                scan_params["ExpressionAttributeValues"][":product_id"] = self.product_id
            
            if self.severity:
                additional_filters.append("#severity = :severity")
                scan_params["ExpressionAttributeNames"]["#severity"] = "severity"
                scan_params["ExpressionAttributeValues"][":severity"] = self.severity
            
            if self.start_date:
                additional_filters.append("#timestamp >= :start_date")
                scan_params["ExpressionAttributeNames"]["#timestamp"] = "timestamp"
                scan_params["ExpressionAttributeValues"][":start_date"] = self.start_date.isoformat()
            
            if self.end_date:
                additional_filters.append("#timestamp <= :end_date")
                scan_params["ExpressionAttributeNames"]["#timestamp"] = "timestamp"
                scan_params["ExpressionAttributeValues"][":end_date"] = self.end_date.isoformat()
            
            if additional_filters:
                scan_params["FilterExpression"] += " AND " + " AND ".join(additional_filters)
            
            response = self.dynamodb.scan(**scan_params)
            
            failures = response.get("Items", [])
            
            return {
                "success": True,
                "failures": failures,
                "count": len(failures)
            }
            
        except Exception as e:
            logger.error("Failed to get cold chain failures", error=str(e))
            return {
                "success": False,
                "error": str(e),
                "failures": []
            }


class GetStockLowEventsQuery(Query):
    """Query to get stock low events"""
    
    def __init__(self, dynamodb_client,
                 product_id: Optional[str] = None,
                 urgency_level: Optional[str] = None,
                 start_date: Optional[datetime] = None,
                 end_date: Optional[datetime] = None,
                 limit: int = 100):
        self.dynamodb = dynamodb_client
        self.product_id = product_id
        self.urgency_level = urgency_level
        self.start_date = start_date
        self.end_date = end_date
        self.limit = limit
    
    def execute(self) -> Dict[str, Any]:
        """Get stock low events"""
        try:
            # Build scan parameters
            scan_params = {
                "TableName": "movimiento-inventario-events",
                "FilterExpression": "#event_type = :event_type",
                "ExpressionAttributeNames": {
                    "#event_type": "event_type"
                },
                "ExpressionAttributeValues": {
                    ":event_type": "StockBajo"
                },
                "Limit": self.limit
            }
            
            # Add additional filters
            additional_filters = []
            
            if self.product_id:
                additional_filters.append("#product_id = :product_id")
                scan_params["ExpressionAttributeNames"]["#product_id"] = "product_id"
                scan_params["ExpressionAttributeValues"][":product_id"] = self.product_id
            
            if self.urgency_level:
                additional_filters.append("#urgency_level = :urgency_level")
                scan_params["ExpressionAttributeNames"]["#urgency_level"] = "urgency_level"
                scan_params["ExpressionAttributeValues"][":urgency_level"] = self.urgency_level
            
            if self.start_date:
                additional_filters.append("#timestamp >= :start_date")
                scan_params["ExpressionAttributeNames"]["#timestamp"] = "timestamp"
                scan_params["ExpressionAttributeValues"][":start_date"] = self.start_date.isoformat()
            
            if self.end_date:
                additional_filters.append("#timestamp <= :end_date")
                scan_params["ExpressionAttributeNames"]["#timestamp"] = "timestamp"
                scan_params["ExpressionAttributeValues"][":end_date"] = self.end_date.isoformat()
            
            if additional_filters:
                scan_params["FilterExpression"] += " AND " + " AND ".join(additional_filters)
            
            response = self.dynamodb.scan(**scan_params)
            
            events = response.get("Items", [])
            
            return {
                "success": True,
                "events": events,
                "count": len(events)
            }
            
        except Exception as e:
            logger.error("Failed to get stock low events", error=str(e))
            return {
                "success": False,
                "error": str(e),
                "events": []
            }


class GetProductStockHistoryQuery(Query):
    """Query to get stock history for a product over time"""
    
    def __init__(self, product_id: str, dynamodb_client,
                 days: int = 30):
        self.product_id = product_id
        self.dynamodb = dynamodb_client
        self.days = days
    
    def execute(self) -> Dict[str, Any]:
        """Get stock history for a product"""
        try:
            end_date = datetime.utcnow()
            start_date = end_date - timedelta(days=self.days)
            
            # Get all movements for the product in the date range
            movements_query = GetInventoryMovementsQuery(
                product_id=product_id,
                dynamodb_client=dynamodb_client,
                start_date=start_date,
                end_date=end_date,
                limit=1000
            )
            
            movements_result = movements_query.execute()
            
            if not movements_result["success"]:
                return movements_result
            
            # Process movements to create stock history
            movements = movements_result["movements"]
            stock_history = []
            current_stock = 0
            
            # Sort movements by timestamp
            movements.sort(key=lambda x: x.get("timestamp", ""))
            
            for movement in movements:
                movement_type = movement.get("movement_type", "")
                quantity = movement.get("quantity", 0)
                
                if movement_type == "in":
                    current_stock += quantity
                elif movement_type in ["out", "loss"]:
                    current_stock -= quantity
                elif movement_type == "adjustment":
                    current_stock = quantity  # Direct adjustment
                
                stock_history.append({
                    "timestamp": movement.get("timestamp"),
                    "stock": current_stock,
                    "movement_type": movement_type,
                    "quantity": quantity,
                    "reason": movement.get("reason", "")
                })
            
            return {
                "success": True,
                "stock_history": stock_history,
                "current_stock": current_stock,
                "period_days": self.days
            }
            
        except Exception as e:
            logger.error("Failed to get stock history", 
                        product_id=self.product_id, 
                        error=str(e))
            return {
                "success": False,
                "error": str(e),
                "stock_history": []
            }
