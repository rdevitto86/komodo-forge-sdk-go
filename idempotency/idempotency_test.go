package idempotency

import (
	"testing"
	"time"
)

func TestNewStore_DefaultsToLocal(t *testing.T) {
	store := NewStore("", 0)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if _, ok := store.cache.(*LocalCache); !ok {
		t.Error("expected LocalCache for empty mode")
	}
}

func TestNewStore_LocalMode(t *testing.T) {
	store := NewStore("local", 0)
	if _, ok := store.cache.(*LocalCache); !ok {
		t.Error("expected LocalCache for local mode")
	}
}

func TestNewStore_DistributedMode(t *testing.T) {
	store := NewStore("distributed", 0)
	if _, ok := store.cache.(*DistributedCache); !ok {
		t.Error("expected DistributedCache for distributed mode")
	}
}

func TestStore_Check_NewKey(t *testing.T) {
	store := NewStore("local", 300)
	allowed, err := store.Check("new-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected new key to be allowed")
	}
}

func TestStore_Check_DuplicateKey(t *testing.T) {
	store := NewStore("local", 300)
	// First check should allow
	allowed, err := store.Check("dup-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected new key to be allowed")
	}

	// Set the key
	if err := store.Set("dup-key"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	// Second check should reject
	allowed, err = store.Check("dup-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected duplicate key to be rejected")
	}
}

func TestStore_Check_ExpiredKey(t *testing.T) {
	store := NewStore("local", 1) // 1 second TTL
	// Set the key
	if err := store.Set("expire-key"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	// Wait for expiry
	time.Sleep(2 * time.Second)

	// Check should allow after expiry
	allowed, err := store.Check("expire-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected expired key to be allowed")
	}
}

func TestStore_Set(t *testing.T) {
	store := NewStore("local", 300)
	if err := store.Set("test-key"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}
}

func TestStore_Delete(t *testing.T) {
	store := NewStore("local", 300)
	// Set the key
	if err := store.Set("delete-key"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	// Delete the key
	store.Delete("delete-key")

	// Check should allow after deletion
	allowed, err := store.Check("delete-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected deleted key to be allowed")
	}
}

func TestLocalCache_BasicOperations(t *testing.T) {
	cache := &LocalCache{}

	// Store
	if err := cache.Store("key", "value", 300); err != nil {
		t.Fatalf("failed to store: %v", err)
	}

	// Load
	val, ok := cache.Load("key")
	if !ok {
		t.Error("expected key to exist")
	}
	if val != "value" {
		t.Errorf("expected value, got %v", val)
	}

	// Delete
	cache.Delete("key")

	// Load after delete
	_, ok = cache.Load("key")
	if ok {
		t.Error("expected key to be deleted")
	}
}
