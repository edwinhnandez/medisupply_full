package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proveedor/internal/handlers"
	"proveedor/internal/observability"

	"github.com/rabbitmq/amqp091-go"
)

func main() {
	log.Println("Starting Proveedor service...")

	// Initialize observability
	tp, err := observability.InitTracing("proveedor-service", "http://jaeger:14268/api/traces")
	if err != nil {
		log.Printf("Failed to initialize tracing: %v", err)
	} else {
		defer observability.Shutdown(tp, nil)
	}

	mp, err := observability.InitMetrics("proveedor-service")
	if err != nil {
		log.Printf("Failed to initialize metrics: %v", err)
	} else {
		defer observability.Shutdown(nil, mp)
	}

	// Connect to RabbitMQ
	conn, err := amqp091.Dial("amqp://guest:guest@rabbitmq-service:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare queue
	q, err := ch.QueueDeclare(
		"recepcion-proveedor", // name
		true,                  // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Consume messages
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	// Create event handler
	eventHandler := handlers.NewEventHandler()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	log.Println("Proveedor service started. Waiting for messages...")

	// Process messages
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, shutting down...")
			return
		case msg := <-msgs:
			if err := eventHandler.HandleRecepcionProveedorEvent(ctx, msg); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		case <-time.After(1 * time.Second):
			// Continue loop
		}
	}
}
