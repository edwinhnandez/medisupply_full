package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

const (
	PurchaseOrderEventType     EventType = "RecepcionProveedor"
	InventoryReceivedEventType EventType = "InventarioRecibido"
)

// RecepcionProveedorEvent represents a purchase order reception event from OrdenCompra
type RecepcionProveedorEvent struct {
	ID              string                 `json:"id" dynamodbav:"id"`
	Timestamp       time.Time              `json:"timestamp" dynamodbav:"timestamp"`
	Type            string                 `json:"type" dynamodbav:"type"`
	EventType       EventType              `json:"event_type" dynamodbav:"event_type"`
	PurchaseOrderID string                 `json:"purchase_order_id" dynamodbav:"purchase_order_id"`
	ProductID       string                 `json:"product_id" dynamodbav:"product_id"`
	ProductoID      string                 `json:"producto_id" dynamodbav:"producto_id"`
	ProductName     string                 `json:"product_name" dynamodbav:"product_name"`
	Quantity        int                    `json:"quantity" dynamodbav:"quantity"`
	Cantidad        int                    `json:"cantidad" dynamodbav:"cantidad"`
	SupplierID      string                 `json:"supplier_id" dynamodbav:"supplier_id"`
	ProveedorID     string                 `json:"proveedor_id" dynamodbav:"proveedor_id"`
	SupplierName    string                 `json:"supplier_name" dynamodbav:"supplier_name"`
	Location        string                 `json:"location" dynamodbav:"location"`
	Status          string                 `json:"status" dynamodbav:"status"`
	Estado          string                 `json:"estado" dynamodbav:"estado"`
	FechaRecepcion  time.Time              `json:"fecha_recepcion" dynamodbav:"fecha_recepcion"`
	Metadata        map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
}

// InventoryReceivedEvent represents an inventory received event
type InventoryReceivedEvent struct {
	ID              string                 `json:"id" dynamodbav:"id"`
	Timestamp       time.Time              `json:"timestamp" dynamodbav:"timestamp"`
	EventType       EventType              `json:"event_type" dynamodbav:"event_type"`
	PurchaseOrderID string                 `json:"purchase_order_id" dynamodbav:"purchase_order_id"`
	ProductID       string                 `json:"product_id" dynamodbav:"product_id"`
	ProductName     string                 `json:"product_name" dynamodbav:"product_name"`
	Quantity        int                    `json:"quantity" dynamodbav:"quantity"`
	SupplierID      string                 `json:"supplier_id" dynamodbav:"supplier_id"`
	SupplierName    string                 `json:"supplier_name" dynamodbav:"supplier_name"`
	Location        string                 `json:"location" dynamodbav:"location"`
	Status          string                 `json:"status" dynamodbav:"status"`
	ReceivedAt      time.Time              `json:"received_at" dynamodbav:"received_at"`
	QualityCheck    string                 `json:"quality_check" dynamodbav:"quality_check"`
	Temperature     *float64               `json:"temperature,omitempty" dynamodbav:"temperature,omitempty"`
	BatchNumber     string                 `json:"batch_number" dynamodbav:"batch_number"`
	ExpiryDate      *time.Time             `json:"expiry_date,omitempty" dynamodbav:"expiry_date,omitempty"`
	Metadata        map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
}

// Supplier represents a supplier
type Supplier struct {
	ID        string                 `json:"id" dynamodbav:"id"`
	Name      string                 `json:"name" dynamodbav:"name"`
	Email     string                 `json:"email" dynamodbav:"email"`
	Phone     string                 `json:"phone" dynamodbav:"phone"`
	Address   string                 `json:"address" dynamodbav:"address"`
	IsActive  bool                   `json:"is_active" dynamodbav:"is_active"`
	CreatedAt time.Time              `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" dynamodbav:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata" dynamodbav:"metadata"`
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

// NewInventoryReceivedEvent creates a new InventoryReceivedEvent
func NewInventoryReceivedEvent(purchaseOrderID, productID, productName, supplierID, supplierName, location, status string, quantity int) *InventoryReceivedEvent {
	return &InventoryReceivedEvent{
		ID:              uuid.New().String(),
		Timestamp:       time.Now().UTC(),
		EventType:       InventoryReceivedEventType,
		PurchaseOrderID: purchaseOrderID,
		ProductID:       productID,
		ProductName:     productName,
		Quantity:        quantity,
		SupplierID:      supplierID,
		SupplierName:    supplierName,
		Location:        location,
		Status:          status,
		ReceivedAt:      time.Now().UTC(),
		QualityCheck:    "pending",
		BatchNumber:     generateBatchNumber(),
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

// ProcessReception processes the reception event and creates inventory received event
func (r *RecepcionProveedorEvent) ProcessReception() *InventoryReceivedEvent {
	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Create inventory received event
	event := NewInventoryReceivedEvent(
		r.PurchaseOrderID,
		r.ProductID,
		r.ProductName,
		r.SupplierID,
		r.SupplierName,
		r.Location,
		"received",
		r.Quantity,
	)

	// Add correlation information
	event.Metadata["correlation_id"] = r.Metadata["correlation_id"]
	event.Metadata["causation_id"] = r.Metadata["causation_id"]
	event.Metadata["purchase_order_id"] = r.PurchaseOrderID
	event.Metadata["reception_event_id"] = r.ID

	// Simulate quality check
	event.QualityCheck = "passed"

	// Simulate temperature check for temperature-controlled products
	if r.Metadata["temperature_controlled"] == true {
		temp := 2.5 // Simulate temperature reading
		event.Temperature = &temp
	}

	// Set expiry date (simulate 30 days from now)
	expiryDate := time.Now().UTC().AddDate(0, 0, 30)
	event.ExpiryDate = &expiryDate

	return event
}

// IsTemperatureControlled checks if the product is temperature controlled
func (r *RecepcionProveedorEvent) IsTemperatureControlled() bool {
	if tempControlled, ok := r.Metadata["temperature_controlled"]; ok {
		if tc, ok := tempControlled.(bool); ok {
			return tc
		}
	}
	return false
}

// GetUrgencyLevel gets the urgency level from metadata
func (r *RecepcionProveedorEvent) GetUrgencyLevel() string {
	if urgency, ok := r.Metadata["urgency_level"]; ok {
		if u, ok := urgency.(string); ok {
			return u
		}
	}
	return "medium"
}

// generateBatchNumber generates a batch number for the received inventory
func generateBatchNumber() string {
	return "BATCH-" + uuid.New().String()[:8]
}

// UpdateStatus updates the inventory received event status
func (i *InventoryReceivedEvent) UpdateStatus(status string) {
	i.Status = status
	if status == "processed" {
		i.QualityCheck = "completed"
	}
}

// IsProcessed checks if the inventory has been processed
func (i *InventoryReceivedEvent) IsProcessed() bool {
	return i.Status == "processed" || i.Status == "completed"
}

// GetQualityStatus returns the quality check status
func (i *InventoryReceivedEvent) GetQualityStatus() string {
	return i.QualityCheck
}

// SetTemperature sets the temperature reading
func (i *InventoryReceivedEvent) SetTemperature(temp float64) {
	i.Temperature = &temp
}

// SetExpiryDate sets the expiry date
func (i *InventoryReceivedEvent) SetExpiryDate(expiryDate time.Time) {
	i.ExpiryDate = &expiryDate
}

// RecepcionProveedor represents a recepcion proveedor entity
type RecepcionProveedor struct {
	ID             string    `json:"id" dynamodbav:"id"`
	ProveedorID    string    `json:"proveedor_id" dynamodbav:"proveedor_id"`
	ProductoID     string    `json:"producto_id" dynamodbav:"producto_id"`
	Cantidad       int       `json:"cantidad" dynamodbav:"cantidad"`
	FechaRecepcion time.Time `json:"fecha_recepcion" dynamodbav:"fecha_recepcion"`
	Estado         string    `json:"estado" dynamodbav:"estado"`
	CreatedAt      time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

// InventarioRecibidoEvent represents an inventario recibido event
type InventarioRecibidoEvent struct {
	ID             string    `json:"id" dynamodbav:"id"`
	ProveedorID    string    `json:"proveedor_id" dynamodbav:"proveedor_id"`
	ProductoID     string    `json:"producto_id" dynamodbav:"producto_id"`
	Cantidad       int       `json:"cantidad" dynamodbav:"cantidad"`
	FechaRecepcion time.Time `json:"fecha_recepcion" dynamodbav:"fecha_recepcion"`
	Estado         string    `json:"estado" dynamodbav:"estado"`
	Timestamp      time.Time `json:"timestamp" dynamodbav:"timestamp"`
}
