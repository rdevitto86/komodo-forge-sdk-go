package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	httpcontext "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

// EventType is the canonical event name in <entity>.<verb> format (e.g.
// "order.created"). Each publisher service defines its own typed constants;
// the SDK owns only the type, not the values.
type EventType string

// Source identifies the service that emitted the event (e.g.
// "komodo-order-api"). Each publisher service defines its own constant;
// the SDK owns only the type, not the values.
type Source string

// EntityType identifies the domain entity at the centre of the event.
// This set is bounded by the number of first-class domains and is therefore
// owned centrally.
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

// Event is the canonical business event envelope published to SNS FIFO topics
// and consumed via SQS FIFO queues. MessageGroupId is set to EntityID so all
// events for the same entity are ordered; events for different entities are
// processed in parallel.
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

// Constructs an Event with a generated ID, current UTC timestamp, and
// version "1". Chain WithCorrelation or WithCorrelationFromContext to attach a
// correlation ID before publishing.
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

// Returns a copy of the event with CorrelationID
// read from the X-Correlation-ID value stored in ctx by the HTTP middleware.
// If no correlation ID is present in ctx the event is returned unchanged.
func (e Event) WithCorrelationFromContext(ctx context.Context) Event {
	if id := httpcontext.GetCorrelationID(ctx); id != "" {
		e.CorrelationID = id
	}
	return e
}
