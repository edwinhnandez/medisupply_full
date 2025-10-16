package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

const (
	StockLowEventType        EventType = "StockBajo"
	PurchaseOrderEventType  EventType = "RecepcionProveedor"
	SupplierEventType       EventType = "InventarioRecibido"
)

// StockLowEvent represents a stock low event from MovimientoInventario
type StockLowEvent struct {
	ID           string                 `json:"id" dynamodbav:"id"`
	Timestamp    time.Time              `json:"timestamp" dynamodbav:"timestamp"`
	EventType    EventType             `json:"event_type" dynamodbav:"event_type"`
	ProductID    string                `json:"product_id" dynamodbav:"product_id"`
	ProductName  string                `json:"product_name" dynamodbav:"product_name"`
	CurrentStock int                   `json:"current_stock" dynamodbav:"current_stock"`
	MinimumStock int                   `json:"minimum_stock" dynamodbav:"minimum_stock"`
	Location     string                `json:"location" dynamodbav:"location"`
	UrgencyLevel string                `json:"urgency_level" dynamodbav:"urgency_level"`
	Metadata     map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
}

// PurchaseOrder represents a purchase order
type PurchaseOrder struct {
	ID              string                 `json:"id" dynamodbav:"id"`
	ProductID       string                 `json:"product_id" dynamodbav:"product_id"`
	ProductName     string                 `json:"product_name" dynamodbav:"product_name"`
	Quantity        int                    `json:"quantity" dynamodbav:"quantity"`
	SupplierID      string                 `json:"supplier_id" dynamodbav:"supplier_id"`
	SupplierName    string                 `json:"supplier_name" dynamodbav:"supplier_name"`
	Location        string                 `json:"location" dynamodbav:"location"`
	Status          string                 `json:"status" dynamodbav:"status"`
	UrgencyLevel    string                 `json:"urgency_level" dynamodbav:"urgency_level"`
	CreatedAt       time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" dynamodbav:"updated_at"`
	ExpectedDate    *time.Time             `json:"expected_date,omitempty" dynamodbav:"expected_date,omitempty"`
	ActualDate      *time.Time             `json:"actual_date,omitempty" dynamodbav:"actual_date,omitempty"`
	Metadata        map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
}

// Supplier represents a supplier
type Supplier struct {
	ID          string                 `json:"id" dynamodbav:"id"`
	Name        string                 `json:"name" dynamodbav:"name"`
	Email       string                 `json:"email" dynamodbav:"email"`
	Phone       string                 `json:"phone" dynamodbav:"phone"`
	Address     string                 `json:"address" dynamodbav:"address"`
	IsActive    bool                   `json:"is_active" dynamodbav:"is_active"`
	CreatedAt   time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" dynamodbav:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
}

// RecepcionProveedorEvent represents a supplier reception event
type RecepcionProveedorEvent struct {
	ID              string                 `json:"id" dynamodbav:"id"`
	Timestamp       time.Time              `json:"timestamp" dynamodbav:"timestamp"`
	EventType       EventType             `json:"event_type" dynamodbav:"event_type"`
	PurchaseOrderID string                 `json:"purchase_order_id" dynamodbav:"purchase_order_id"`
	ProductID       string                 `json:"product_id" dynamodbav:"product_id"`
	ProductName     string                 `json:"product_name" dynamodbav:"product_name"`
	Quantity        int                    `json:"quantity" dynamodbav:"quantity"`
	SupplierID      string                 `json:"supplier_id" dynamodbav:"supplier_id"`
	SupplierName    string                 `json:"supplier_name" dynamodbav:"supplier_name"`
	Location        string                 `json:"location" dynamodbav:"location"`
	Status          string                 `json:"status" dynamodbav:"status"`
	Metadata        map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
}

// EventSourcingEvent represents an event sourcing event
type EventSourcingEvent struct {
	ID            string                 `json:"id" dynamodbav:"id"`
	AggregateID   string                 `json:"aggregate_id" dynamodbav:"aggregate_id"`
	EventType     string                 `json:"event_type" dynamodbav:"event_type"`
	EventData     map[string]interface{} `json:"event_data" dynamodbav:"event_data"`
	Timestamp     time.Time              `json:"timestamp" dynamodbav:"timestamp"`
	Version       int                    `json:"version" dynamodbav:"version"`
	CorrelationID *string                `json:"correlation_id,omitempty" dynamodbav:"correlation_id,omitempty"`
	CausationID   *string                `json:"causation_id,omitempty" dynamodbav:"causation_id,omitempty"`
}

// NewStockLowEvent creates a new StockLowEvent
func NewStockLowEvent(productID, productName, location, urgencyLevel string, currentStock, minimumStock int) *StockLowEvent {
	return &StockLowEvent{
		ID:           uuid.New().String(),
		Timestamp:    time.Now().UTC(),
		EventType:    StockLowEventType,
		ProductID:    productID,
		ProductName:  productName,
		CurrentStock: currentStock,
		MinimumStock: minimumStock,
		Location:     location,
		UrgencyLevel: urgencyLevel,
		Metadata:     make(map[string]interface{}),
	}
}

// NewPurchaseOrder creates a new PurchaseOrder
func NewPurchaseOrder(productID, productName, supplierID, supplierName, location, urgencyLevel string, quantity int) *PurchaseOrder {
	now := time.Now().UTC()
	expectedDate := now.AddDate(0, 0, 7) // Default 7 days from now
	
	return &PurchaseOrder{
		ID:           uuid.New().String(),
		ProductID:    productID,
		ProductName:  productName,
		Quantity:     quantity,
		SupplierID:   supplierID,
		SupplierName: supplierName,
		Location:     location,
		Status:       "pending",
		UrgencyLevel: urgencyLevel,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpectedDate: &expectedDate,
		Metadata:     make(map[string]interface{}),
	}
}

// NewRecepcionProveedorEvent creates a new RecepcionProveedorEvent
func NewRecepcionProveedorEvent(purchaseOrderID, productID, productName, supplierID, supplierName, location, status string, quantity int) *RecepcionProveedorEvent {
	return &RecepcionProveedorEvent{
		ID:              uuid.New().String(),
		Timestamp:       time.Now().UTC(),
		EventType:       PurchaseOrderEventType,
		PurchaseOrderID: purchaseOrderID,
		ProductID:       productID,
		ProductName:     productName,
		Quantity:        quantity,
		SupplierID:      supplierID,
		SupplierName:    supplierName,
		Location:        location,
		Status:          status,
		Metadata:        make(map[string]interface{}),
	}
}

// NewEventSourcingEvent creates a new EventSourcingEvent
func NewEventSourcingEvent(aggregateID, eventType string, eventData map[string]interface{}, correlationID, causationID *string) *EventSourcingEvent {
	return &EventSourcingEvent{
		ID:            uuid.New().String(),
		AggregateID:   aggregateID,
		EventType:     eventType,
		EventData:     eventData,
		Timestamp:     time.Now().UTC(),
		Version:       1,
		CorrelationID: correlationID,
		CausationID:   causationID,
	}
}

// CalculateQuantity calculates the quantity to order based on urgency level
func (s *StockLowEvent) CalculateQuantity() int {
	baseQuantity := s.MinimumStock * 2 // Order 2x minimum stock
	
	urgencyMultipliers := map[string]float64{
		"low":      1.0,
		"medium":   1.5,
		"high":     2.0,
		"critical": 3.0,
	}
	
	multiplier := urgencyMultipliers[s.UrgencyLevel]
	if multiplier == 0 {
		multiplier = 1.0
	}
	
	return int(float64(baseQuantity) * multiplier)
}

// GetSupplierID returns the supplier ID for the product
func (s *StockLowEvent) GetSupplierID() string {
	// In a real implementation, this would look up the preferred supplier
	// For now, return a default supplier ID
	return "supplier-001"
}

// GetSupplierName returns the supplier name for the product
func (s *StockLowEvent) GetSupplierName() string {
	// In a real implementation, this would look up the supplier name
	// For now, return a default supplier name
	return "Default Supplier"
}

// UpdateStatus updates the purchase order status
func (po *PurchaseOrder) UpdateStatus(status string) {
	po.Status = status
	po.UpdatedAt = time.Now().UTC()
	
	if status == "received" {
		now := time.Now().UTC()
		po.ActualDate = &now
	}
}

// IsCompleted checks if the purchase order is completed
func (po *PurchaseOrder) IsCompleted() bool {
	return po.Status == "received" || po.Status == "completed"
}

// IsOverdue checks if the purchase order is overdue
func (po *PurchaseOrder) IsOverdue() bool {
	if po.ExpectedDate == nil {
		return false
	}
	return time.Now().UTC().After(*po.ExpectedDate) && !po.IsCompleted()
}
