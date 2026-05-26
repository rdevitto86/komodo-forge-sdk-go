package sqs

import (
	"context"
	"testing"
)

func TestNew_missingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for missing region, got nil")
	}
}

func TestNew_validRegion(t *testing.T) {
	c, err := New(context.Background(), Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}
