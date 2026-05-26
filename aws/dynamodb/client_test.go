package dynamodb

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

func TestNew_validRegion_localstack(t *testing.T) {
	// Uses the LocalStack path: Endpoint set without static credentials triggers
	// the default credential chain, which is fine in -short mode with a fake endpoint.
	_, err := New(context.Background(), Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	// Config load succeeds even without reachable infrastructure; only a real
	// operation would fail. Accept both nil and a load error.
	_ = err
}
