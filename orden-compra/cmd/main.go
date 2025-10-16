package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"

	"orden-compra/internal/handlers"
	"orden-compra/internal/observability"
)

func main() {
	// Initialize observability
	tp, err := observability.InitTracing("orden-compra", "http://jaeger:14268/api/traces")
	if err != nil {
		log.Printf("Failed to initialize tracing: %v", err)
	} else {
		defer observability.Shutdown(tp, nil)
	}

	mp, err := observability.InitMetrics("orden-compra")
	if err != nil {
		log.Printf("Failed to initialize metrics: %v", err)
	} else {
		defer observability.Shutdown(nil, mp)
	}

	logger := log.New(os.Stdout, "[orden-compra] ", log.LstdFlags)

	// Get configuration from environment variables
	config := getConfig()

	// Initialize DynamoDB client
	dynamoDB, err := initializeDynamoDB(config)
	if err != nil {
		log.Fatalf("Failed to initialize DynamoDB: %v", err)
	}

	// Initialize RabbitMQ connection
	rabbitMQConn, err := initializeRabbitMQ(config)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ: %v", err)
	}
	defer rabbitMQConn.Close()

	// Initialize handlers
	rabbitMQHandler, err := handlers.NewRabbitMQHandler(
		rabbitMQConn,
		config.RabbitMQ.QueueName,
		config.RabbitMQ.ExchangeName,
		config.RabbitMQ.RoutingKey,
		dynamoDB,
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ handler: %v", err)
	}

	healthHandler := handlers.NewHealthCheckHandler(dynamoDB, logger)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start RabbitMQ consumer
	err = rabbitMQHandler.StartConsuming()
	if err != nil {
		log.Fatalf("Failed to start RabbitMQ consumer: %v", err)
	}

	// Start HTTP server
	router := setupRouter(healthHandler)
	go func() {
		log.Printf("Starting HTTP server on port %s", config.Server.Port)
		if err := router.Run(":" + config.Server.Port); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	log.Println("Orden Compra service started successfully")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, shutting down gracefully")

	// Stop RabbitMQ consumer
	rabbitMQHandler.StopConsuming()

	log.Println("Orden Compra service stopped")
}

// Config represents the service configuration
type Config struct {
	Server struct {
		Port string
	}
	RabbitMQ struct {
		URL          string
		QueueName    string
		ExchangeName string
		RoutingKey   string
	}
	DynamoDB struct {
		Endpoint string
		Region   string
	}
}

// getConfig gets configuration from environment variables
func getConfig() Config {
	config := Config{}

	// Server configuration
	config.Server.Port = getEnv("SERVICE_PORT", "8000")

	// RabbitMQ configuration
	config.RabbitMQ.URL = getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq-service:5672/")
	config.RabbitMQ.QueueName = getEnv("RABBITMQ_QUEUE_NAME", "stock-bajo-queue")
	config.RabbitMQ.ExchangeName = getEnv("RABBITMQ_EXCHANGE_NAME", "stock-bajo-exchange")
	config.RabbitMQ.RoutingKey = getEnv("RABBITMQ_ROUTING_KEY", "stock.bajo")

	// DynamoDB configuration
	config.DynamoDB.Endpoint = getEnv("DYNAMODB_ENDPOINT", "http://dynamodb-local:8000")
	config.DynamoDB.Region = getEnv("DYNAMODB_REGION", "us-east-1")

	return config
}

// initializeDynamoDB initializes the DynamoDB client
func initializeDynamoDB(config Config) (*dynamodb.DynamoDB, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint:    aws.String(config.DynamoDB.Endpoint),
		Region:      aws.String(config.DynamoDB.Region),
		Credentials: credentials.NewStaticCredentials("dummy", "dummy", ""),
	})
	if err != nil {
		return nil, err
	}

	return dynamodb.New(sess), nil
}

// initializeRabbitMQ initializes the RabbitMQ connection
func initializeRabbitMQ(config Config) (*amqp091.Connection, error) {
	conn, err := amqp091.Dial(config.RabbitMQ.URL)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// setupRouter sets up the HTTP router
func setupRouter(healthHandler *handlers.HealthCheckHandler) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		health := healthHandler.CheckHealth(ctx)

		if health["status"] == "healthy" {
			c.JSON(200, health)
		} else {
			c.JSON(503, health)
		}
	})

	// Metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":   "Metrics endpoint",
			"timestamp": time.Now().Unix(),
		})
	})

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service":   "Orden Compra",
			"version":   "1.0.0",
			"status":    "running",
			"timestamp": time.Now().Unix(),
		})
	})

	return router
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
