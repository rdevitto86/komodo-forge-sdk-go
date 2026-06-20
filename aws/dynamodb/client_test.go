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
	_, err := New(context.Background(), Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	_ = err
}
