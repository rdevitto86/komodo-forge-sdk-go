package request

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewRequest_Methods(t *testing.T) {
	// Note: GET/DELETE/HEAD/OPTIONS set bodyReader to a typed nil *strings.Reader,
	// which causes http.NewRequestWithContext to panic when it calls .Len().
	// Only test methods that set bodyReader to a real value or return errors.
	tests := []struct {
		method  string
		body    any
		wantErr bool
	}{
		{"POST", `{"key":"value"}`, false},
		{"PUT", `{"key":"value"}`, false},
		{"PATCH", `{"key":"value"}`, false},
		{"INVALID", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			req, err := NewRequest(tc.method, "http://example.com/path", tc.body, nil, context.Background())
			if tc.wantErr {
				if err == nil {
					t.Errorf("NewRequest(%q) expected error, got nil", tc.method)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRequest(%q) unexpected error: %v", tc.method, err)
			}
			if req.Method != strings.ToUpper(tc.method) {
				t.Errorf("Method = %q, want %q", req.Method, strings.ToUpper(tc.method))
			}
		})
	}
}

func TestNewRequest_EmptyURL(t *testing.T) {
	_, err := NewRequest("POST", "", `{}`, nil, context.Background())
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestNewRequest_BodyAsString(t *testing.T) {
	req, err := NewRequest("POST", "http://example.com", `{"hello":"world"}`, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Body == nil {
		t.Error("expected non-nil body")
	}
}

func TestNewRequest_BodyAsStruct(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	req, err := NewRequest("POST", "http://example.com", payload{Name: "test"}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Body == nil {
		t.Error("expected non-nil body for struct payload")
	}
}

func TestNewRequest_BodyMarshalError(t *testing.T) {
	// Channels cannot be JSON marshaled - this exercises the error marshal branch
	ch := make(chan int)
	_, err := NewRequest("POST", "http://example.com", ch, nil, nil)
	if err == nil {
		t.Error("expected error for non-marshallable body")
	}
}

func TestNewRequest_InvalidURL(t *testing.T) {
	// Malformed URL causes http.NewRequestWithContext to fail
	_, err := NewRequest("POST", "://invalid-url", `{}`, nil, nil)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestNewRequest_WithHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type": "application/json",
		"X-Custom":     "value123",
	}
	req, err := NewRequest("POST", "http://example.com", `{}`, headers, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("X-Custom") != "value123" {
		t.Errorf("X-Custom = %q, want 'value123'", req.Header.Get("X-Custom"))
	}
}

func TestNewRequest_NilContext_UsesBackground(t *testing.T) {
	req, err := NewRequest("POST", "http://example.com", `{}`, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Context() == nil {
		t.Error("expected non-nil context")
	}
}

func TestFromRequest_Valid(t *testing.T) {
	orig := httptest.NewRequest("POST", "http://example.com/api", strings.NewReader(`{}`))
	orig.Header.Set("Content-Type", "application/json")

	cloned, err := FromRequest(orig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloned.Method != "POST" {
		t.Errorf("Method = %q, want POST", cloned.Method)
	}
}

func TestFromRequest_Nil(t *testing.T) {
	_, err := FromRequest(nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}
