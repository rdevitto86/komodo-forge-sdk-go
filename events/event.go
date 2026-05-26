package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	httpcontext "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

// Represents the canonical event name in <entity>.<verb> format (e.g., "order.created").
type EventType string

// Identifies the service that emitted the event (e.g., "komodo-order-api").
type Source string

// Identifies the domain entity at the centre of the event; owned centrally to keep the set bounded.
type EntityType string

const (
	EntityOrder     EntityType = "order"
	EntityUser      EntityType = "user"
	EntityPayment   EntityType = "payment"
	EntityCart      EntityType = "cart"
	EntityInventory EntityType = "inventory"
	EntityProduct   EntityType = "product"
	EntityShipment  EntityType = "shipment"
	EntityReview    EntityType = "review"
)

// Represents the canonical business event envelope published to SNS FIFO topics and consumed via SQS FIFO queues.
type Event struct {
	ID            string         `json:"id"`
	Type          EventType      `json:"type"`
	Source        Source         `json:"source"`
	EntityID      string         `json:"entity_id"`
	EntityType    EntityType     `json:"entity_type"`
	OccurredAt    time.Time      `json:"occurred_at"`
	Version       string         `json:"version"`
	Payload       map[string]any `json:"payload"`
	CorrelationID string         `json:"correlation_id,omitempty"`
}

// Constructs an Event with a generated ID, current UTC timestamp, and version "1".
func New(
	eventType EventType,
	source Source,
	entityID string,
	entityType EntityType,
	payload map[string]any,
) Event {
	return Event{
		ID:         uuid.NewString(),
		Type:       eventType,
		Source:     source,
		EntityID:   entityID,
		EntityType: entityType,
		OccurredAt: time.Now().UTC(),
		Version:    "1",
		Payload:    payload,
	}
}

// Returns a copy of the event with CorrelationID set.
func (e Event) WithCorrelation(correlationID string) Event {
	e.CorrelationID = correlationID
	return e
}

// Returns a copy of the event with CorrelationID read from ctx; returns the event unchanged if no ID is present.
func (e Event) WithCorrelationFromContext(ctx context.Context) Event {
	if id := httpcontext.GetCorrelationID(ctx); id != "" {
		e.CorrelationID = id
	}
	return e
}
