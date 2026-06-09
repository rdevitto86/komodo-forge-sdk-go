package opensearch

import (
	"errors"
	"testing"
)

func TestNew_NotImplemented(t *testing.T) {
	c, err := New(Config{Endpoint: "https://example.test"})
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if c != nil {
		t.Errorf("expected nil client when not implemented, got %v", c)
	}
}
