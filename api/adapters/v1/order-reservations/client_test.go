package orderreservations

import "testing"

func TestNewClient_Validates(t *testing.T) {
	if _, err := NewClient("", 1); err == nil {
		t.Fatal("expected error for empty baseURL")
	}
	if _, err := NewClient("http://x", 0); err == nil {
		t.Fatal("expected error for unsupported version 0")
	}
	if _, err := NewClient("http://x", 99); err == nil {
		t.Fatal("expected error for unsupported version 99")
	}
	if _, err := NewClient("http://x", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
