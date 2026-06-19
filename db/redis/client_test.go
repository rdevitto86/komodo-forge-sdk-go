package redis

import (
	"context"
	"testing"
	"time"
)

func TestMGet_EmptyKeys_ReturnsNil(t *testing.T) {
	c := &Client{}
	vals, err := c.MGet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals != nil {
		t.Fatalf("expected nil for empty keys, got %v", vals)
	}
}

func TestNew_MissingAddr(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for missing addr, got nil")
	}
	if err.Error() != "missing addr" {
		t.Fatalf("unexpected error %q", err.Error())
	}
}

func TestNewFromDBString_MalformedDB(t *testing.T) {
	_, err := NewFromDBString("localhost:6379", "", "notanumber")
	if err == nil {
		t.Fatal("expected error for malformed db string, got nil")
	}
	const wantPrefix = `invalid db string "notanumber"`
	if len(err.Error()) < len(wantPrefix) || err.Error()[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("unexpected error %q; want prefix %q", err.Error(), wantPrefix)
	}
}

func TestWithTimeout_DefaultsToTwoSeconds(t *testing.T) {
	c := &Client{opTimeout: 0}
	ctx, cancel := c.withTimeout(context.Background())
	defer cancel()

	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}
	if d := time.Until(dl); d <= 0 || d > 2*time.Second+50*time.Millisecond {
		t.Fatalf("expected ~2s deadline, got %v", d)
	}
}

func TestWithTimeout_CustomOpTimeout(t *testing.T) {
	c := &Client{opTimeout: 500 * time.Millisecond}
	ctx, cancel := c.withTimeout(context.Background())
	defer cancel()

	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}
	if d := time.Until(dl); d <= 0 || d > 500*time.Millisecond+50*time.Millisecond {
		t.Fatalf("expected ~500ms deadline, got %v", d)
	}
}

func TestWithTimeout_ExistingDeadlinePreserved(t *testing.T) {
	c := &Client{opTimeout: 10 * time.Second}

	parent, parentCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer parentCancel()

	ctx, cancel := c.withTimeout(parent)
	defer cancel()

	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if d := time.Until(dl); d > 100*time.Millisecond+10*time.Millisecond {
		t.Fatalf("expected parent deadline preserved at ~100ms, got %v", d)
	}
}
