package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"proveedor/internal/cqrs"
	"proveedor/internal/models"

	"github.com/rabbitmq/amqp091-go"
)

// EventHandler handles incoming events
type EventHandler struct {
	createHandler *cqrs.CreateRecepcionProveedorHandler
	updateHandler *cqrs.UpdateRecepcionProveedorHandler
}

// NewEventHandler creates a new event handler
func NewEventHandler() *EventHandler {
	return &EventHandler{
		createHandler: cqrs.NewCreateRecepcionProveedorHandler(),
		updateHandler: cqrs.NewUpdateRecepcionProveedorHandler(),
	}
}

// HandleRecepcionProveedorEvent handles recepcion proveedor events
func (h *EventHandler) HandleRecepcionProveedorEvent(ctx context.Context, delivery amqp091.Delivery) error {
	log.Printf("Received recepcion proveedor event: %s", delivery.Body)

	var event models.RecepcionProveedorEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		log.Printf("Error unmarshaling event: %v", err)
		return err
	}

	switch event.Type {
	case "RecepcionProveedorCreated":
		cmd := cqrs.CreateRecepcionProveedorCommand{
			ProveedorID:    event.ProveedorID,
			ProductoID:     event.ProductoID,
			Cantidad:       event.Cantidad,
			FechaRecepcion: event.FechaRecepcion,
			Estado:         event.Estado,
		}

		recepcion, err := h.createHandler.Handle(ctx, cmd)
		if err != nil {
			log.Printf("Error creating recepcion proveedor: %v", err)
			return err
		}

		log.Printf("Created recepcion proveedor: %s", recepcion.ID)

		// Produce InventarioRecibido event
		return h.produceInventarioRecibidoEvent(ctx, recepcion)

	case "RecepcionProveedorUpdated":
		cmd := cqrs.UpdateRecepcionProveedorCommand{
			ID:     event.ID,
			Estado: event.Estado,
		}

		if err := h.updateHandler.Handle(ctx, cmd); err != nil {
			log.Printf("Error updating recepcion proveedor: %v", err)
			return err
		}

		log.Printf("Updated recepcion proveedor: %s", event.ID)

	default:
		log.Printf("Unknown event type: %s", event.Type)
	}

	return nil
}

// produceInventarioRecibidoEvent produces an inventario recibido event
func (h *EventHandler) produceInventarioRecibidoEvent(ctx context.Context, recepcion *models.RecepcionProveedor) error {
	// TODO: Implement RabbitMQ producer
	// This would connect to RabbitMQ and publish the InventarioRecibido event

	event := models.InventarioRecibidoEvent{
		ID:             recepcion.ID,
		ProveedorID:    recepcion.ProveedorID,
		ProductoID:     recepcion.ProductoID,
		Cantidad:       recepcion.Cantidad,
		FechaRecepcion: recepcion.FechaRecepcion,
		Estado:         recepcion.Estado,
		Timestamp:      time.Now(),
	}

	log.Printf("Would produce InventarioRecibido event: %+v", event)
	return nil
}
