from pydantic import BaseModel, Field
from typing import Optional, Dict, Any
from datetime import datetime
from enum import Enum
import uuid


class EventType(str, Enum):
    FALLA_CADENA_FRIO = "FallaCadenaFrio"
    STOCK_BAJO = "StockBajo"


class ColdChainFailureEvent(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    event_type: EventType = Field(default=EventType.FALLA_CADENA_FRIO)
    product_id: str
    product_name: str
    temperature: float
    threshold_temperature: float
    location: str
    severity: str  # "low", "medium", "high", "critical"
    metadata: Dict[str, Any] = Field(default_factory=dict)


class StockLowEvent(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    event_type: EventType = Field(default=EventType.STOCK_BAJO)
    product_id: str
    product_name: str
    current_stock: int
    minimum_stock: int
    location: str
    urgency_level: str  # "low", "medium", "high", "critical"
    metadata: Dict[str, Any] = Field(default_factory=dict)


class InventoryMovement(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    product_id: str
    product_name: str
    movement_type: str  # "in", "out", "adjustment", "loss"
    quantity: int
    location: str
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    reason: str
    metadata: Dict[str, Any] = Field(default_factory=dict)


class Product(BaseModel):
    id: str
    name: str
    current_stock: int
    minimum_stock: int
    maximum_stock: int
    location: str
    temperature_controlled: bool
    last_updated: datetime = Field(default_factory=datetime.utcnow)
    metadata: Dict[str, Any] = Field(default_factory=dict)


class EventSourcingEvent(BaseModel):
    id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    aggregate_id: str
    event_type: str
    event_data: Dict[str, Any]
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    version: int = 1
    correlation_id: Optional[str] = None
    causation_id: Optional[str] = None
