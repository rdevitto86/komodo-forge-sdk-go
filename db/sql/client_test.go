package sqldb

import (
	"context"
	"errors"
	"testing"
)

func TestNew_ReturnsErrNotImplemented(t *testing.T) {
	c, err := New(Config{DSN: "postgres://localhost/test"})
	if c != nil {
		t.Fatal("expected nil client, got non-nil")
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestQuery_ReturnsErrNotImplemented(t *testing.T) {
	c := &Client{}
	_, err := c.Query(context.Background(), "SELECT 1")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestExec_ReturnsErrNotImplemented(t *testing.T) {
	c := &Client{}
	_, err := c.Exec(context.Background(), "INSERT INTO t VALUES (1)")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
