package s3

import (
	"context"
	"testing"
)

func TestNew_MissingRegion(t *testing.T) {
	_, err := New(context.Background(), S3Config{})
	if err == nil {
		t.Fatal("expected error for empty region, got nil")
	}
}

func TestNew_ValidRegion(t *testing.T) {
	_, err := New(context.Background(), S3Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPresignPut_ZeroTTL(t *testing.T) {
	c, err := New(context.Background(), S3Config{
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566",
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	_, err = c.PresignPut(context.Background(), "bucket", "key", 0, "", 0)
	if err == nil {
		t.Fatal("expected error for zero TTL, got nil")
	}
}
