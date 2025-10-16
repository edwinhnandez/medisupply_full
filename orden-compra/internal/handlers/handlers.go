package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/rabbitmq/amqp091-go"

	"orden-compra/internal/cqrs"
	"orden-compra/internal/models"
)

// RabbitMQHandler handles RabbitMQ message consumption and production
type RabbitMQHandler struct {
	Connection   *amqp091.Connection
	Channel      *amqp091.Channel
	QueueName    string
	ExchangeName string
	RoutingKey   string
	DynamoDB     *dynamodb.DynamoDB
	Logger       *log.Logger
	Running      bool
}

// NewRabbitMQHandler creates a new RabbitMQ handler
func NewRabbitMQHandler(connection *amqp091.Connection, queueName, exchangeName, routingKey string, dynamoDB *dynamodb.DynamoDB, logger *log.Logger) (*RabbitMQHandler, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	err = channel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	queue, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		queue.Name,   // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &RabbitMQHandler{
		Connection:   connection,
		Channel:      channel,
		QueueName:    queue.Name,
		ExchangeName: exchangeName,
		RoutingKey:   routingKey,
		DynamoDB:     dynamoDB,
		Logger:       logger,
		Running:      false,
	}, nil
}

// StartConsuming starts consuming messages from RabbitMQ
func (h *RabbitMQHandler) StartConsuming() error {
	h.Running = true
	h.Logger.Printf("Starting RabbitMQ consumer - queue: %s, exchange: %s, routing_key: %s", h.QueueName, h.ExchangeName, h.RoutingKey)

	// Set QoS
	err := h.Channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming
	msgs, err := h.Channel.Consume(
		h.QueueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// Process messages
	go func() {
		for msg := range msgs {
			if !h.Running {
				break
			}
			h.processMessage(msg)
		}
	}()

	return nil
}

// StopConsuming stops consuming messages
func (h *RabbitMQHandler) StopConsuming() {
	h.Running = false
	if h.Channel != nil {
		h.Channel.Close()
	}
	if h.Connection != nil {
		h.Connection.Close()
	}
	h.Logger.Println("RabbitMQ consumer stopped")
}

// processMessage processes a single RabbitMQ message
func (h *RabbitMQHandler) processMessage(msg amqp091.Delivery) {
	startTime := time.Now()
	ctx := context.Background()

	// Extract correlation information from headers
	correlationID := extractHeader(msg.Headers, "correlation-id")
	causationID := extractHeader(msg.Headers, "causation-id")

	// Set correlation context
	// TODO: Implement correlation tracking
	_ = correlationID
	_ = causationID

	h.Logger.Printf("Processing message - routing_key: %s, correlation_id: %s, causation_id: %s, message_id: %s", msg.RoutingKey, correlationID, causationID, msg.MessageId)

	// Parse message
	var stockLowEvent models.StockLowEvent
	err := json.Unmarshal(msg.Body, &stockLowEvent)
	if err != nil {
		h.Logger.Printf("Failed to parse message: %v", err)
		// TODO: Record metrics
		msg.Nack(false, false) // Reject message
		return
	}

	// Process the stock low event
	result, err := h.processStockLowEvent(ctx, &stockLowEvent)
	if err != nil {
		h.Logger.Printf("Failed to process stock low event: %v", err)
		// TODO: Record metrics
		msg.Nack(false, true) // Reject and requeue
		return
	}

	// Record metrics
	processingTime := time.Since(startTime)
	// TODO: Record metrics
	_ = processingTime
	_ = result

	// Produce output event if needed
	if result["success"].(bool) && result["reception_event"] != nil {
		receptionEvent := result["reception_event"].(*models.RecepcionProveedorEvent)
		err = h.produceReceptionEvent(ctx, receptionEvent)
		if err != nil {
			h.Logger.Printf("Failed to produce reception event: %v", err)
			// TODO: Record metrics
		}
	}

	// Acknowledge message
	msg.Ack(false)

	h.Logger.Printf("Message processed successfully - event_id: %s, product_id: %s, processing_time: %v, success: %v", stockLowEvent.ID, stockLowEvent.ProductID, processingTime, result["success"])
}

// processStockLowEvent processes a stock low event and creates a purchase order
func (h *RabbitMQHandler) processStockLowEvent(ctx context.Context, event *models.StockLowEvent) (map[string]interface{}, error) {
	// Create and execute command
	command := cqrs.NewProcessStockLowCommand(
		event,
		h.DynamoDB,
		h.Logger,
		nil, // TODO: correlation ID
		nil, // TODO: causation ID
	)

	result, err := command.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %w", err)
	}

	// Record purchase order created
	if result["success"].(bool) {
		receptionEvent := result["reception_event"].(*models.RecepcionProveedorEvent)
		// TODO: Record metrics
		_ = receptionEvent
	}

	return result, nil
}

// produceReceptionEvent produces a reception event to the output exchange
func (h *RabbitMQHandler) produceReceptionEvent(ctx context.Context, event *models.RecepcionProveedorEvent) error {
	// Marshal event to JSON
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Prepare headers
	headers := make(amqp091.Table)
	// TODO: Add correlation headers
	headers["event-type"] = "RecepcionProveedor"
	headers["content-type"] = "application/json"

	// Publish message
	err = h.Channel.PublishWithContext(
		ctx,
		"recepcion-proveedor-exchange", // exchange
		"recepcion.proveedor",          // routing key
		false,                          // mandatory
		false,                          // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Headers:      headers,
			MessageId:    event.ID,
			Timestamp:    event.Timestamp,
			DeliveryMode: amqp091.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	h.Logger.Printf("Reception event produced - event_id: %s, product_id: %s, supplier_id: %s, routing_key: recepcion.proveedor", event.ID, event.ProductID, event.SupplierID)

	return nil
}

// extractHeader extracts a header value from AMQP headers
func extractHeader(headers amqp091.Table, key string) string {
	if headers == nil {
		return ""
	}
	if value, ok := headers[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// HealthCheckHandler handles health check requests
type HealthCheckHandler struct {
	DynamoDB *dynamodb.DynamoDB
	Logger   *log.Logger
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(dynamoDB *dynamodb.DynamoDB, logger *log.Logger) *HealthCheckHandler {
	return &HealthCheckHandler{
		DynamoDB: dynamoDB,
		Logger:   logger,
	}
}

// CheckHealth checks the service health
func (h *HealthCheckHandler) CheckHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"checks":    make(map[string]string),
	}

	// Check DynamoDB connection
	_, err := h.DynamoDB.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String("orden-compra-read"),
	})
	if err != nil {
		h.Logger.Printf("Health check failed - DynamoDB: %v", err)
		health["status"] = "unhealthy"
		health["checks"].(map[string]string)["dynamodb"] = "error"
		health["error"] = err.Error()
	} else {
		health["checks"].(map[string]string)["dynamodb"] = "ok"
	}

	return health
}
