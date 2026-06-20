package s3

import (
	"context"
	"testing"
)

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error for empty region, got nil")
	}
}

func TestNew_ValidRegion(t *testing.T) {
	_, err := New(context.Background(), Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
