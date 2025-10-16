package cqrs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/sirupsen/logrus"

	"orden-compra/internal/models"
)

// Query represents a query in the CQRS pattern
type Query interface {
	Execute(ctx context.Context) (map[string]interface{}, error)
}

// GetPurchaseOrderQuery retrieves a single purchase order by ID
type GetPurchaseOrderQuery struct {
	PurchaseOrderID string
	DynamoDB        *dynamodb.DynamoDB
	Logger          *logrus.Logger
}

// NewGetPurchaseOrderQuery creates a new GetPurchaseOrderQuery
func NewGetPurchaseOrderQuery(purchaseOrderID string, dynamoDB *dynamodb.DynamoDB, logger *logrus.Logger) *GetPurchaseOrderQuery {
	return &GetPurchaseOrderQuery{
		PurchaseOrderID: purchaseOrderID,
		DynamoDB:        dynamoDB,
		Logger:          logger,
	}
}

// Execute retrieves the purchase order
func (q *GetPurchaseOrderQuery) Execute(ctx context.Context) (map[string]interface{}, error) {
	q.Logger.WithFields(logrus.Fields{
		"purchase_order_id": q.PurchaseOrderID,
	}).Debug("Getting purchase order")

	result, err := q.DynamoDB.GetItemWithContext(ctx, &dynamodb.GetItemInput{
		TableName: aws.String("orden-compra-read"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(q.PurchaseOrderID),
			},
		},
	})

	if err != nil {
		q.Logger.WithError(err).Error("Failed to get purchase order")
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Purchase order not found",
		}, nil
	}

	var purchaseOrder models.PurchaseOrder
	err = dynamodbattribute.UnmarshalMap(result.Item, &purchaseOrder)
	if err != nil {
		q.Logger.WithError(err).Error("Failed to unmarshal purchase order")
		return nil, fmt.Errorf("failed to unmarshal purchase order: %w", err)
	}

	return map[string]interface{}{
		"success":        true,
		"purchase_order": purchaseOrder,
	}, nil
}

// ListPurchaseOrdersQuery lists purchase orders with optional filtering
type ListPurchaseOrdersQuery struct {
	ProductID    *string
	SupplierID   *string
	Status       *string
	UrgencyLevel *string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int64
	DynamoDB     *dynamodb.DynamoDB
	Logger       *logrus.Logger
}

// NewListPurchaseOrdersQuery creates a new ListPurchaseOrdersQuery
func NewListPurchaseOrdersQuery(dynamoDB *dynamodb.DynamoDB, logger *logrus.Logger) *ListPurchaseOrdersQuery {
	return &ListPurchaseOrdersQuery{
		DynamoDB: dynamoDB,
		Logger:   logger,
		Limit:    100,
	}
}

// WithProductID sets the product ID filter
func (q *ListPurchaseOrdersQuery) WithProductID(productID string) *ListPurchaseOrdersQuery {
	q.ProductID = &productID
	return q
}

// WithSupplierID sets the supplier ID filter
func (q *ListPurchaseOrdersQuery) WithSupplierID(supplierID string) *ListPurchaseOrdersQuery {
	q.SupplierID = &supplierID
	return q
}

// WithStatus sets the status filter
func (q *ListPurchaseOrdersQuery) WithStatus(status string) *ListPurchaseOrdersQuery {
	q.Status = &status
	return q
}

// WithUrgencyLevel sets the urgency level filter
func (q *ListPurchaseOrdersQuery) WithUrgencyLevel(urgencyLevel string) *ListPurchaseOrdersQuery {
	q.UrgencyLevel = &urgencyLevel
	return q
}

// WithDateRange sets the date range filter
func (q *ListPurchaseOrdersQuery) WithDateRange(startDate, endDate time.Time) *ListPurchaseOrdersQuery {
	q.StartDate = &startDate
	q.EndDate = &endDate
	return q
}

// WithLimit sets the limit
func (q *ListPurchaseOrdersQuery) WithLimit(limit int64) *ListPurchaseOrdersQuery {
	q.Limit = limit
	return q
}

// Execute lists purchase orders with filtering
func (q *ListPurchaseOrdersQuery) Execute(ctx context.Context) (map[string]interface{}, error) {
	q.Logger.Debug("Listing purchase orders")

	// Build scan parameters
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("orden-compra-read"),
		Limit:     aws.Int64(q.Limit),
	}

	// Add filter expressions
	var filterExpressions []string
	expressionAttributeNames := make(map[string]*string)
	expressionAttributeValues := make(map[string]*dynamodb.AttributeValue)

	if q.ProductID != nil {
		filterExpressions = append(filterExpressions, "product_id = :product_id")
		expressionAttributeValues[":product_id"] = &dynamodb.AttributeValue{
			S: q.ProductID,
		}
	}

	if q.SupplierID != nil {
		filterExpressions = append(filterExpressions, "supplier_id = :supplier_id")
		expressionAttributeValues[":supplier_id"] = &dynamodb.AttributeValue{
			S: q.SupplierID,
		}
	}

	if q.Status != nil {
		filterExpressions = append(filterExpressions, "#status = :status")
		expressionAttributeNames["#status"] = aws.String("status")
		expressionAttributeValues[":status"] = &dynamodb.AttributeValue{
			S: q.Status,
		}
	}

	if q.UrgencyLevel != nil {
		filterExpressions = append(filterExpressions, "urgency_level = :urgency_level")
		expressionAttributeValues[":urgency_level"] = &dynamodb.AttributeValue{
			S: q.UrgencyLevel,
		}
	}

	if q.StartDate != nil {
		filterExpressions = append(filterExpressions, "created_at >= :start_date")
		expressionAttributeValues[":start_date"] = &dynamodb.AttributeValue{
			S: aws.String(q.StartDate.Format(time.RFC3339)),
		}
	}

	if q.EndDate != nil {
		filterExpressions = append(filterExpressions, "created_at <= :end_date")
		expressionAttributeValues[":end_date"] = &dynamodb.AttributeValue{
			S: aws.String(q.EndDate.Format(time.RFC3339)),
		}
	}

	if len(filterExpressions) > 0 {
		scanInput.FilterExpression = aws.String(fmt.Sprintf("%s", filterExpressions[0]))
		for i := 1; i < len(filterExpressions); i++ {
			scanInput.FilterExpression = aws.String(fmt.Sprintf("%s AND %s", *scanInput.FilterExpression, filterExpressions[i]))
		}
	}

	if len(expressionAttributeNames) > 0 {
		scanInput.ExpressionAttributeNames = expressionAttributeNames
	}

	if len(expressionAttributeValues) > 0 {
		scanInput.ExpressionAttributeValues = expressionAttributeValues
	}

	result, err := q.DynamoDB.ScanWithContext(ctx, scanInput)
	if err != nil {
		q.Logger.WithError(err).Error("Failed to scan purchase orders")
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	var purchaseOrders []models.PurchaseOrder
	for _, item := range result.Items {
		var purchaseOrder models.PurchaseOrder
		err := dynamodbattribute.UnmarshalMap(item, &purchaseOrder)
		if err != nil {
			q.Logger.WithError(err).Error("Failed to unmarshal purchase order")
			continue
		}
		purchaseOrders = append(purchaseOrders, purchaseOrder)
	}

	return map[string]interface{}{
		"success":         true,
		"purchase_orders": purchaseOrders,
		"count":           len(purchaseOrders),
	}, nil
}

// GetPurchaseOrderEventsQuery retrieves events for a purchase order
type GetPurchaseOrderEventsQuery struct {
	PurchaseOrderID string
	EventType       *string
	StartDate       *time.Time
	EndDate         *time.Time
	Limit           int64
	DynamoDB        *dynamodb.DynamoDB
	Logger          *logrus.Logger
}

// NewGetPurchaseOrderEventsQuery creates a new GetPurchaseOrderEventsQuery
func NewGetPurchaseOrderEventsQuery(purchaseOrderID string, dynamoDB *dynamodb.DynamoDB, logger *logrus.Logger) *GetPurchaseOrderEventsQuery {
	return &GetPurchaseOrderEventsQuery{
		PurchaseOrderID: purchaseOrderID,
		DynamoDB:        dynamoDB,
		Logger:          logger,
		Limit:           100,
	}
}

// WithEventType sets the event type filter
func (q *GetPurchaseOrderEventsQuery) WithEventType(eventType string) *GetPurchaseOrderEventsQuery {
	q.EventType = &eventType
	return q
}

// WithDateRange sets the date range filter
func (q *GetPurchaseOrderEventsQuery) WithDateRange(startDate, endDate time.Time) *GetPurchaseOrderEventsQuery {
	q.StartDate = &startDate
	q.EndDate = &endDate
	return q
}

// WithLimit sets the limit
func (q *GetPurchaseOrderEventsQuery) WithLimit(limit int64) *GetPurchaseOrderEventsQuery {
	q.Limit = limit
	return q
}

// Execute retrieves events for the purchase order
func (q *GetPurchaseOrderEventsQuery) Execute(ctx context.Context) (map[string]interface{}, error) {
	q.Logger.WithFields(logrus.Fields{
		"purchase_order_id": q.PurchaseOrderID,
	}).Debug("Getting purchase order events")

	// Build scan parameters
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("orden-compra-events"),
		Limit:     aws.Int64(q.Limit),
	}

	// Add filter expressions
	var filterExpressions []string
	expressionAttributeNames := make(map[string]*string)
	expressionAttributeValues := make(map[string]*dynamodb.AttributeValue)

	// Filter by aggregate ID (purchase order ID)
	filterExpressions = append(filterExpressions, "aggregate_id = :aggregate_id")
	expressionAttributeValues[":aggregate_id"] = &dynamodb.AttributeValue{
		S: aws.String(q.PurchaseOrderID),
	}

	if q.EventType != nil {
		filterExpressions = append(filterExpressions, "event_type = :event_type")
		expressionAttributeValues[":event_type"] = &dynamodb.AttributeValue{
			S: q.EventType,
		}
	}

	if q.StartDate != nil {
		filterExpressions = append(filterExpressions, "#timestamp >= :start_date")
		expressionAttributeNames["#timestamp"] = aws.String("timestamp")
		expressionAttributeValues[":start_date"] = &dynamodb.AttributeValue{
			S: aws.String(q.StartDate.Format(time.RFC3339)),
		}
	}

	if q.EndDate != nil {
		filterExpressions = append(filterExpressions, "#timestamp <= :end_date")
		expressionAttributeNames["#timestamp"] = aws.String("timestamp")
		expressionAttributeValues[":end_date"] = &dynamodb.AttributeValue{
			S: aws.String(q.EndDate.Format(time.RFC3339)),
		}
	}

	if len(filterExpressions) > 0 {
		scanInput.FilterExpression = aws.String(fmt.Sprintf("%s", filterExpressions[0]))
		for i := 1; i < len(filterExpressions); i++ {
			scanInput.FilterExpression = aws.String(fmt.Sprintf("%s AND %s", *scanInput.FilterExpression, filterExpressions[i]))
		}
	}

	if len(expressionAttributeNames) > 0 {
		scanInput.ExpressionAttributeNames = expressionAttributeNames
	}

	if len(expressionAttributeValues) > 0 {
		scanInput.ExpressionAttributeValues = expressionAttributeValues
	}

	result, err := q.DynamoDB.ScanWithContext(ctx, scanInput)
	if err != nil {
		q.Logger.WithError(err).Error("Failed to scan purchase order events")
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	var events []models.EventSourcingEvent
	for _, item := range result.Items {
		var event models.EventSourcingEvent
		err := dynamodbattribute.UnmarshalMap(item, &event)
		if err != nil {
			q.Logger.WithError(err).Error("Failed to unmarshal event")
			continue
		}
		events = append(events, event)
	}

	return map[string]interface{}{
		"success": true,
		"events":  events,
		"count":   len(events),
	}, nil
}

// GetOverduePurchaseOrdersQuery retrieves overdue purchase orders
type GetOverduePurchaseOrdersQuery struct {
	Limit    int64
	DynamoDB *dynamodb.DynamoDB
	Logger   *logrus.Logger
}

// NewGetOverduePurchaseOrdersQuery creates a new GetOverduePurchaseOrdersQuery
func NewGetOverduePurchaseOrdersQuery(dynamoDB *dynamodb.DynamoDB, logger *logrus.Logger) *GetOverduePurchaseOrdersQuery {
	return &GetOverduePurchaseOrdersQuery{
		DynamoDB: dynamoDB,
		Logger:   logger,
		Limit:    100,
	}
}

// WithLimit sets the limit
func (q *GetOverduePurchaseOrdersQuery) WithLimit(limit int64) *GetOverduePurchaseOrdersQuery {
	q.Limit = limit
	return q
}

// Execute retrieves overdue purchase orders
func (q *GetOverduePurchaseOrdersQuery) Execute(ctx context.Context) (map[string]interface{}, error) {
	q.Logger.Debug("Getting overdue purchase orders")

	// Get all purchase orders
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("orden-compra-read"),
		Limit:     aws.Int64(q.Limit),
	}

	result, err := q.DynamoDB.ScanWithContext(ctx, scanInput)
	if err != nil {
		q.Logger.WithError(err).Error("Failed to scan purchase orders")
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	var overdueOrders []models.PurchaseOrder

	for _, item := range result.Items {
		var purchaseOrder models.PurchaseOrder
		err := dynamodbattribute.UnmarshalMap(item, &purchaseOrder)
		if err != nil {
			q.Logger.WithError(err).Error("Failed to unmarshal purchase order")
			continue
		}

		// Check if order is overdue
		if purchaseOrder.IsOverdue() {
			overdueOrders = append(overdueOrders, purchaseOrder)
		}
	}

	return map[string]interface{}{
		"success":         true,
		"purchase_orders": overdueOrders,
		"count":           len(overdueOrders),
	}, nil
}

// GetPurchaseOrderStatsQuery retrieves purchase order statistics
type GetPurchaseOrderStatsQuery struct {
	StartDate *time.Time
	EndDate   *time.Time
	DynamoDB  *dynamodb.DynamoDB
	Logger    *logrus.Logger
}

// NewGetPurchaseOrderStatsQuery creates a new GetPurchaseOrderStatsQuery
func NewGetPurchaseOrderStatsQuery(dynamoDB *dynamodb.DynamoDB, logger *logrus.Logger) *GetPurchaseOrderStatsQuery {
	return &GetPurchaseOrderStatsQuery{
		DynamoDB: dynamoDB,
		Logger:   logger,
	}
}

// WithDateRange sets the date range filter
func (q *GetPurchaseOrderStatsQuery) WithDateRange(startDate, endDate time.Time) *GetPurchaseOrderStatsQuery {
	q.StartDate = &startDate
	q.EndDate = &endDate
	return q
}

// Execute retrieves purchase order statistics
func (q *GetPurchaseOrderStatsQuery) Execute(ctx context.Context) (map[string]interface{}, error) {
	q.Logger.Debug("Getting purchase order statistics")

	// Get all purchase orders
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("orden-compra-read"),
	}

	result, err := q.DynamoDB.ScanWithContext(ctx, scanInput)
	if err != nil {
		q.Logger.WithError(err).Error("Failed to scan purchase orders")
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	stats := map[string]interface{}{
		"total_orders":     0,
		"pending_orders":   0,
		"completed_orders": 0,
		"overdue_orders":   0,
		"by_status":        make(map[string]int),
		"by_urgency":       make(map[string]int),
		"by_supplier":      make(map[string]int),
	}

	for _, item := range result.Items {
		var purchaseOrder models.PurchaseOrder
		err := dynamodbattribute.UnmarshalMap(item, &purchaseOrder)
		if err != nil {
			q.Logger.WithError(err).Error("Failed to unmarshal purchase order")
			continue
		}

		// Apply date filter if specified
		if q.StartDate != nil && purchaseOrder.CreatedAt.Before(*q.StartDate) {
			continue
		}
		if q.EndDate != nil && purchaseOrder.CreatedAt.After(*q.EndDate) {
			continue
		}

		// Update statistics
		stats["total_orders"] = stats["total_orders"].(int) + 1

		// Count by status
		if statusCount, ok := stats["by_status"].(map[string]int)[purchaseOrder.Status]; ok {
			stats["by_status"].(map[string]int)[purchaseOrder.Status] = statusCount + 1
		} else {
			stats["by_status"].(map[string]int)[purchaseOrder.Status] = 1
		}

		// Count by urgency
		if urgencyCount, ok := stats["by_urgency"].(map[string]int)[purchaseOrder.UrgencyLevel]; ok {
			stats["by_urgency"].(map[string]int)[purchaseOrder.UrgencyLevel] = urgencyCount + 1
		} else {
			stats["by_urgency"].(map[string]int)[purchaseOrder.UrgencyLevel] = 1
		}

		// Count by supplier
		if supplierCount, ok := stats["by_supplier"].(map[string]int)[purchaseOrder.SupplierID]; ok {
			stats["by_supplier"].(map[string]int)[purchaseOrder.SupplierID] = supplierCount + 1
		} else {
			stats["by_supplier"].(map[string]int)[purchaseOrder.SupplierID] = 1
		}

		// Count specific statuses
		if purchaseOrder.Status == "pending" {
			stats["pending_orders"] = stats["pending_orders"].(int) + 1
		}
		if purchaseOrder.IsCompleted() {
			stats["completed_orders"] = stats["completed_orders"].(int) + 1
		}
		if purchaseOrder.IsOverdue() {
			stats["overdue_orders"] = stats["overdue_orders"].(int) + 1
		}
	}

	return map[string]interface{}{
		"success": true,
		"stats":   stats,
	}, nil
}
