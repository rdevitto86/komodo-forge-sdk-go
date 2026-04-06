package events

import (
	"testing"
	"time"
)

// --- New ---

func TestEvents_Envelope_New_Success(t *testing.T) {
	before := time.Now().UTC()

	evt := New(
		EventOrderCreated,
		SourceOrderAPI,
		"order-123",
		EntityOrder,
		map[string]any{"amount": 99.99},
	)

	after := time.Now().UTC()

	if evt.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if evt.Type != EventOrderCreated {
		t.Errorf("Type = %q, want %q", evt.Type, EventOrderCreated)
	}
	if evt.Source != SourceOrderAPI {
		t.Errorf("Source = %q, want %q", evt.Source, SourceOrderAPI)
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
	evt := New(EventUserCreated, SourceUserAPI, "user-1", EntityUser, nil)

	if evt.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if evt.Payload != nil {
		t.Errorf("expected nil payload, got %v", evt.Payload)
	}
}

func TestEvents_Envelope_New_UniqueIDs_Success(t *testing.T) {
	evt1 := New(EventOrderCreated, SourceOrderAPI, "e1", EntityOrder, nil)
	evt2 := New(EventOrderCreated, SourceOrderAPI, "e2", EntityOrder, nil)

	if evt1.ID == evt2.ID {
		t.Error("expected unique IDs for each event")
	}
}

// --- WithCorrelation ---

func TestEvents_Envelope_WithCorrelation_Success(t *testing.T) {
	evt := New(EventPaymentSucceeded, SourcePaymentsAPI, "pay-1", EntityPayment, nil)

	correlated := evt.WithCorrelation("corr-abc-123")

	if correlated.CorrelationID != "corr-abc-123" {
		t.Errorf("CorrelationID = %q, want corr-abc-123", correlated.CorrelationID)
	}
	// Original must be immutable (value copy).
	if evt.CorrelationID != "" {
		t.Errorf("original CorrelationID mutated to %q", evt.CorrelationID)
	}
	// Other fields must be preserved.
	if correlated.ID != evt.ID {
		t.Errorf("ID changed after WithCorrelation: %q vs %q", correlated.ID, evt.ID)
	}
	if correlated.Type != evt.Type {
		t.Errorf("Type changed after WithCorrelation")
	}
}

func TestEvents_Envelope_WithCorrelation_Empty_Success(t *testing.T) {
	evt := New(EventOrderCancelled, SourceOrderAPI, "o-1", EntityOrder, nil)
	correlated := evt.WithCorrelation("")

	if correlated.CorrelationID != "" {
		t.Errorf("CorrelationID = %q, want empty", correlated.CorrelationID)
	}
}

// --- EventType constants ---

func TestEvents_EventType_Constants_Success(t *testing.T) {
	all := []EventType{
		EventOrderCreated, EventOrderStatusUpdated, EventOrderCancelled, EventOrderFulfilled,
		EventUserCreated, EventUserProfileUpdated, EventUserDeleted,
		EventPaymentInitiated, EventPaymentSucceeded, EventPaymentFailed, EventPaymentRefunded,
		EventCartCheckedOut, EventInventoryReserved, EventInventoryReleased,
	}
	seen := make(map[EventType]bool)
	for _, c := range all {
		if string(c) == "" {
			t.Errorf("EventType constant is empty: %v", c)
		}
		if seen[c] {
			t.Errorf("duplicate EventType constant: %q", c)
		}
		seen[c] = true
	}
}

// --- Source constants ---

func TestEvents_Source_Constants_Success(t *testing.T) {
	all := []Source{
		SourceAuthAPI, SourceUserAPI, SourceOrderAPI, SourceCartAPI,
		SourceInventoryAPI, SourcePaymentsAPI, SourceShopItemsAPI, SourceCommunicationsAPI,
	}
	seen := make(map[Source]bool)
	for _, s := range all {
		if string(s) == "" {
			t.Errorf("Source constant is empty: %v", s)
		}
		if seen[s] {
			t.Errorf("duplicate Source constant: %q", s)
		}
		seen[s] = true
	}
}

// --- EntityType constants ---

func TestEvents_EntityType_Constants_Success(t *testing.T) {
	all := []EntityType{
		EntityOrder, EntityUser, EntityPayment, EntityCart, EntityInventory, EntityProduct,
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
