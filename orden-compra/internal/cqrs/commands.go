package cqrs

import (
	"context"
	"fmt"

	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"orden-compra/internal/models"
)

// Command represents a command in the CQRS pattern
type Command interface {
	Execute(ctx context.Context) (map[string]interface{}, error)
}

// ProcessStockLowCommand processes stock low events and creates purchase orders
type ProcessStockLowCommand struct {
	Event         *models.StockLowEvent
	DynamoDB      *dynamodb.DynamoDB
	Logger        *log.Logger
	CorrelationID *string
	CausationID   *string
}

// NewProcessStockLowCommand creates a new ProcessStockLowCommand
func NewProcessStockLowCommand(event *models.StockLowEvent, dynamoDB *dynamodb.DynamoDB, logger *log.Logger, correlationID, causationID *string) *ProcessStockLowCommand {
	return &ProcessStockLowCommand{
		Event:         event,
		DynamoDB:      dynamoDB,
		Logger:        logger,
		CorrelationID: correlationID,
		CausationID:   causationID,
	}
}

// Execute processes the stock low event and creates a purchase order
func (c *ProcessStockLowCommand) Execute(ctx context.Context) (map[string]interface{}, error) {
	c.Logger.Printf("Processing stock low event - event_id: %s, product_id: %s, urgency: %s, correlation_id: %v", c.Event.ID, c.Event.ProductID, c.Event.UrgencyLevel, c.CorrelationID)

	// Calculate quantity to order
	quantity := c.Event.CalculateQuantity()

	// Get supplier information
	supplierID := c.Event.GetSupplierID()
	supplierName := c.Event.GetSupplierName()

	// Create purchase order
	purchaseOrder := models.NewPurchaseOrder(
		c.Event.ProductID,
		c.Event.ProductName,
		supplierID,
		supplierName,
		c.Event.Location,
		c.Event.UrgencyLevel,
		quantity,
	)

	// Add correlation information
	purchaseOrder.Metadata["correlation_id"] = c.CorrelationID
	purchaseOrder.Metadata["causation_id"] = c.CausationID
	purchaseOrder.Metadata["stock_low_event_id"] = c.Event.ID

	// Store purchase order in read model
	if err := c.storePurchaseOrder(ctx, purchaseOrder); err != nil {
		c.Logger.Printf("Failed to store purchase order: %v", err)
		return nil, fmt.Errorf("failed to store purchase order: %w", err)
	}

	// Store event sourcing event
	if err := c.storeEventSourcingEvent(ctx, purchaseOrder); err != nil {
		c.Logger.Printf("Failed to store event sourcing event: %v", err)
		return nil, fmt.Errorf("failed to store event sourcing event: %w", err)
	}

	// Create reception event
	receptionEvent := models.NewRecepcionProveedorEvent(
		purchaseOrder.ID,
		purchaseOrder.ProductID,
		purchaseOrder.ProductName,
		purchaseOrder.SupplierID,
		purchaseOrder.SupplierName,
		purchaseOrder.Location,
		"pending",
		purchaseOrder.Quantity,
	)

	// Add correlation information
	receptionEvent.Metadata["correlation_id"] = c.CorrelationID
	receptionEvent.Metadata["causation_id"] = c.CausationID
	receptionEvent.Metadata["purchase_order_id"] = purchaseOrder.ID
	receptionEvent.Metadata["stock_low_event_id"] = c.Event.ID

	c.Logger.Printf("Purchase order created successfully - purchase_order_id: %s, product_id: %s, quantity: %d, supplier_id: %s", purchaseOrder.ID, purchaseOrder.ProductID, purchaseOrder.Quantity, purchaseOrder.SupplierID)

	return map[string]interface{}{
		"success":           true,
		"purchase_order_id": purchaseOrder.ID,
		"reception_event":   receptionEvent,
		"correlation_id":    c.CorrelationID,
	}, nil
}

// storePurchaseOrder stores the purchase order in the read model
func (c *ProcessStockLowCommand) storePurchaseOrder(ctx context.Context, purchaseOrder *models.PurchaseOrder) error {
	item, err := dynamodbattribute.MarshalMap(purchaseOrder)
	if err != nil {
		return fmt.Errorf("failed to marshal purchase order: %w", err)
	}

	_, err = c.DynamoDB.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("orden-compra-read"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

// storeEventSourcingEvent stores the event sourcing event
func (c *ProcessStockLowCommand) storeEventSourcingEvent(ctx context.Context, purchaseOrder *models.PurchaseOrder) error {
	eventData := map[string]interface{}{
		"purchase_order": purchaseOrder,
		"stock_low_event": map[string]interface{}{
			"id":            c.Event.ID,
			"product_id":    c.Event.ProductID,
			"urgency_level": c.Event.UrgencyLevel,
		},
	}

	event := models.NewEventSourcingEvent(
		purchaseOrder.ID,
		"PurchaseOrderCreated",
		eventData,
		c.CorrelationID,
		c.CausationID,
	)

	item, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event sourcing event: %w", err)
	}

	_, err = c.DynamoDB.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("orden-compra-events"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put event sourcing event: %w", err)
	}

	return nil
}

// CreatePurchaseOrderCommand creates a new purchase order
type CreatePurchaseOrderCommand struct {
	PurchaseOrder *models.PurchaseOrder
	DynamoDB      *dynamodb.DynamoDB
	Logger        *log.Logger
	CorrelationID *string
	CausationID   *string
}

// NewCreatePurchaseOrderCommand creates a new CreatePurchaseOrderCommand
func NewCreatePurchaseOrderCommand(purchaseOrder *models.PurchaseOrder, dynamoDB *dynamodb.DynamoDB, logger *log.Logger, correlationID, causationID *string) *CreatePurchaseOrderCommand {
	return &CreatePurchaseOrderCommand{
		PurchaseOrder: purchaseOrder,
		DynamoDB:      dynamoDB,
		Logger:        logger,
		CorrelationID: correlationID,
		CausationID:   causationID,
	}
}

// Execute creates a new purchase order
func (c *CreatePurchaseOrderCommand) Execute(ctx context.Context) (map[string]interface{}, error) {
	c.Logger.Printf("Creating purchase order - purchase_order_id: %s, product_id: %s, supplier_id: %s, quantity: %d", c.PurchaseOrder.ID, c.PurchaseOrder.ProductID, c.PurchaseOrder.SupplierID, c.PurchaseOrder.Quantity)

	// Store purchase order in read model
	if err := c.storePurchaseOrder(ctx, c.PurchaseOrder); err != nil {
		c.Logger.Printf("Failed to store purchase order: %v", err)
		return nil, fmt.Errorf("failed to store purchase order: %w", err)
	}

	// Store event sourcing event
	if err := c.storeEventSourcingEvent(ctx, c.PurchaseOrder); err != nil {
		c.Logger.Printf("Failed to store event sourcing event: %v", err)
		return nil, fmt.Errorf("failed to store event sourcing event: %w", err)
	}

	c.Logger.Printf("Purchase order created successfully - purchase_order_id: %s, product_id: %s", c.PurchaseOrder.ID, c.PurchaseOrder.ProductID)

	return map[string]interface{}{
		"success":           true,
		"purchase_order_id": c.PurchaseOrder.ID,
		"correlation_id":    c.CorrelationID,
	}, nil
}

// storePurchaseOrder stores the purchase order in the read model
func (c *CreatePurchaseOrderCommand) storePurchaseOrder(ctx context.Context, purchaseOrder *models.PurchaseOrder) error {
	item, err := dynamodbattribute.MarshalMap(purchaseOrder)
	if err != nil {
		return fmt.Errorf("failed to marshal purchase order: %w", err)
	}

	_, err = c.DynamoDB.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("orden-compra-read"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

// storeEventSourcingEvent stores the event sourcing event
func (c *CreatePurchaseOrderCommand) storeEventSourcingEvent(ctx context.Context, purchaseOrder *models.PurchaseOrder) error {
	eventData := map[string]interface{}{
		"purchase_order": purchaseOrder,
	}

	event := models.NewEventSourcingEvent(
		purchaseOrder.ID,
		"PurchaseOrderCreated",
		eventData,
		c.CorrelationID,
		c.CausationID,
	)

	item, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event sourcing event: %w", err)
	}

	_, err = c.DynamoDB.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("orden-compra-events"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put event sourcing event: %w", err)
	}

	return nil
}

// UpdatePurchaseOrderStatusCommand updates the status of a purchase order
type UpdatePurchaseOrderStatusCommand struct {
	PurchaseOrderID string
	Status          string
	DynamoDB        *dynamodb.DynamoDB
	Logger          *log.Logger
	CorrelationID   *string
	CausationID     *string
}

// NewUpdatePurchaseOrderStatusCommand creates a new UpdatePurchaseOrderStatusCommand
func NewUpdatePurchaseOrderStatusCommand(purchaseOrderID, status string, dynamoDB *dynamodb.DynamoDB, logger *log.Logger, correlationID, causationID *string) *UpdatePurchaseOrderStatusCommand {
	return &UpdatePurchaseOrderStatusCommand{
		PurchaseOrderID: purchaseOrderID,
		Status:          status,
		DynamoDB:        dynamoDB,
		Logger:          logger,
		CorrelationID:   correlationID,
		CausationID:     causationID,
	}
}

// Execute updates the purchase order status
func (c *UpdatePurchaseOrderStatusCommand) Execute(ctx context.Context) (map[string]interface{}, error) {
	c.Logger.Printf("Updating purchase order status - purchase_order_id: %s, status: %s, correlation_id: %v", c.PurchaseOrderID, c.Status, c.CorrelationID)

	// Get current purchase order
	purchaseOrder, err := c.getPurchaseOrder(ctx)
	if err != nil {
		c.Logger.Printf("Failed to get purchase order: %v", err)
		return nil, fmt.Errorf("failed to get purchase order: %w", err)
	}

	// Update status
	purchaseOrder.UpdateStatus(c.Status)

	// Store updated purchase order
	if err := c.storePurchaseOrder(ctx, purchaseOrder); err != nil {
		c.Logger.Printf("Failed to store updated purchase order: %v", err)
		return nil, fmt.Errorf("failed to store updated purchase order: %w", err)
	}

	// Store event sourcing event
	if err := c.storeEventSourcingEvent(ctx, purchaseOrder); err != nil {
		c.Logger.Printf("Failed to store event sourcing event: %v", err)
		return nil, fmt.Errorf("failed to store event sourcing event: %w", err)
	}

	c.Logger.Printf("Purchase order status updated successfully - purchase_order_id: %s, status: %s", c.PurchaseOrderID, c.Status)

	return map[string]interface{}{
		"success":           true,
		"purchase_order_id": c.PurchaseOrderID,
		"status":            c.Status,
		"correlation_id":    c.CorrelationID,
	}, nil
}

// getPurchaseOrder retrieves the purchase order from the database
func (c *UpdatePurchaseOrderStatusCommand) getPurchaseOrder(ctx context.Context) (*models.PurchaseOrder, error) {
	result, err := c.DynamoDB.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("orden-compra-read"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(c.PurchaseOrderID),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("purchase order not found")
	}

	var purchaseOrder models.PurchaseOrder
	err = dynamodbattribute.UnmarshalMap(result.Item, &purchaseOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal purchase order: %w", err)
	}

	return &purchaseOrder, nil
}

// storePurchaseOrder stores the purchase order in the read model
func (c *UpdatePurchaseOrderStatusCommand) storePurchaseOrder(ctx context.Context, purchaseOrder *models.PurchaseOrder) error {
	item, err := dynamodbattribute.MarshalMap(purchaseOrder)
	if err != nil {
		return fmt.Errorf("failed to marshal purchase order: %w", err)
	}

	_, err = c.DynamoDB.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("orden-compra-read"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

// storeEventSourcingEvent stores the event sourcing event
func (c *UpdatePurchaseOrderStatusCommand) storeEventSourcingEvent(ctx context.Context, purchaseOrder *models.PurchaseOrder) error {
	eventData := map[string]interface{}{
		"purchase_order": purchaseOrder,
		"status_change": map[string]interface{}{
			"old_status": "unknown", // In a real implementation, we'd track the previous status
			"new_status": c.Status,
		},
	}

	event := models.NewEventSourcingEvent(
		purchaseOrder.ID,
		"PurchaseOrderStatusUpdated",
		eventData,
		c.CorrelationID,
		c.CausationID,
	)

	item, err := dynamodbattribute.MarshalMap(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event sourcing event: %w", err)
	}

	_, err = c.DynamoDB.PutItemWithContext(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("orden-compra-events"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to put event sourcing event: %w", err)
	}

	return nil
}
