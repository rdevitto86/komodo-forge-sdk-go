package events

import (
	"testing"
	"time"
)

// Publisher-defined constants — each service defines these locally.
// Shown here to demonstrate the intended usage pattern.
const (
	evtOrderCreated   EventType = "order.created"
	evtOrderCancelled EventType = "order.cancelled"
	evtPaymentSuccess EventType = "payment.succeeded"
	evtUserCreated    EventType = "user.created"

	srcOrderAPI    Source = "komodo-order-api"
	srcUserAPI     Source = "komodo-user-api"
	srcPaymentsAPI Source = "komodo-payments-api"
)

func TestEvents_Envelope_New_Success(t *testing.T) {
	before := time.Now().UTC()

	evt := New(evtOrderCreated, srcOrderAPI, "order-123", EntityOrder, map[string]any{"amount": 99.99})

	after := time.Now().UTC()

	if evt.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if evt.Type != evtOrderCreated {
		t.Errorf("Type = %q, want %q", evt.Type, evtOrderCreated)
	}
	if evt.Source != srcOrderAPI {
		t.Errorf("Source = %q, want %q", evt.Source, srcOrderAPI)
	}
	if evt.EntityID != "order-123" {
		t.Errorf("EntityID = %q, want order-123", evt.EntityID)
	}
	if evt.EntityType != EntityOrder {
		t.Errorf("EntityType = %q, want %q", evt.EntityType, EntityOrder)
	}
	if evt.OccurredAt.Before(before) || evt.OccurredAt.After(after) {
		t.Errorf("OccurredAt %v not in range [%v, %v]", evt.OccurredAt, before, after)
	}
	if evt.Version != "1" {
		t.Errorf("Version = %q, want 1", evt.Version)
	}
	if evt.Payload["amount"] != 99.99 {
		t.Errorf("Payload amount = %v, want 99.99", evt.Payload["amount"])
	}
	if evt.CorrelationID != "" {
		t.Errorf("CorrelationID should be empty by default, got %q", evt.CorrelationID)
	}
}

func TestEvents_Envelope_New_NilPayload_Success(t *testing.T) {
	evt := New(evtUserCreated, srcUserAPI, "user-1", EntityUser, nil)

	if evt.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if evt.Payload != nil {
		t.Errorf("expected nil payload, got %v", evt.Payload)
	}
}

func TestEvents_Envelope_New_UniqueIDs_Success(t *testing.T) {
	evt1 := New(evtOrderCreated, srcOrderAPI, "e1", EntityOrder, nil)
	evt2 := New(evtOrderCreated, srcOrderAPI, "e2", EntityOrder, nil)

	if evt1.ID == evt2.ID {
		t.Error("expected unique IDs for each event")
	}
}

func TestEvents_Envelope_WithCorrelation_Success(t *testing.T) {
	evt := New(evtPaymentSuccess, srcPaymentsAPI, "pay-1", EntityPayment, nil)

	correlated := evt.WithCorrelation("corr-abc-123")

	if correlated.CorrelationID != "corr-abc-123" {
		t.Errorf("CorrelationID = %q, want corr-abc-123", correlated.CorrelationID)
	}
	if evt.CorrelationID != "" {
		t.Errorf("original CorrelationID mutated to %q", evt.CorrelationID)
	}
	if correlated.ID != evt.ID {
		t.Errorf("ID changed after WithCorrelation: %q vs %q", correlated.ID, evt.ID)
	}
	if correlated.Type != evt.Type {
		t.Errorf("Type changed after WithCorrelation")
	}
}

func TestEvents_Envelope_WithCorrelation_Empty_Success(t *testing.T) {
	evt := New(evtOrderCancelled, srcOrderAPI, "o-1", EntityOrder, nil)
	correlated := evt.WithCorrelation("")

	if correlated.CorrelationID != "" {
		t.Errorf("CorrelationID = %q, want empty", correlated.CorrelationID)
	}
}

func TestEvents_EntityType_Constants_Success(t *testing.T) {
	all := []EntityType{
		EntityOrder, EntityUser, EntityPayment, EntityCart,
		EntityInventory, EntityProduct, EntityShipment, EntityReview,
	}
	seen := make(map[EntityType]bool)
	for _, e := range all {
		if string(e) == "" {
			t.Errorf("EntityType constant is empty: %v", e)
		}
		if seen[e] {
			t.Errorf("duplicate EntityType constant: %q", e)
		}
		seen[e] = true
	}
}
