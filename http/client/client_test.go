package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testPayload struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestGetJSON_200(t *testing.T) {
	want := testPayload{ID: 1, Name: "widget"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient()
	got, err := GetJSON[testPayload](c, context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGetJSON_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient()
	_, err := GetJSON[testPayload](c, context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", httpErr.StatusCode)
	}
}

func TestPostJSON_201(t *testing.T) {
	want := testPayload{ID: 42, Name: "created"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewClient()
	got, err := PostJSON[testPayload](c, context.Background(), srv.URL, map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestPostJSON_409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "conflict", http.StatusConflict)
	}))
	defer srv.Close()

	c := NewClient()
	_, err := PostJSON[testPayload](c, context.Background(), srv.URL, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", httpErr.StatusCode)
	}
}

func TestGetJSON_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately so the request fails at transport layer

	c := NewClient()
	_, err := GetJSON[testPayload](c, context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		t.Fatalf("expected a transport error (not *HTTPError), got *HTTPError with status %d", httpErr.StatusCode)
	}
}

// --- Circuit breaker integration tests ---

func newBreakerClient(threshold int, timeout time.Duration) *Client {
	return NewClient(WithCircuitBreaker(Config{
		FailureThreshold:    threshold,
		SuccessThreshold:    1,
		OpenTimeout:         timeout,
		MaxHalfOpenRequests: 1,
	}))
}

func TestWithCircuitBreaker_TripsAfterNFailures(t *testing.T) {
	const threshold = 3
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newBreakerClient(threshold, time.Hour)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)

	for i := range threshold {
		resp, err := c.Do(req)
		if err != nil {
			t.Fatalf("call %d: unexpected transport error: %v", i, err)
		}
		resp.Body.Close()
	}

	// Breaker should be open now.
	_, err := c.Do(req)
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("expected ErrOpen after %d failures, got %v", threshold, err)
	}
}

func TestWithCircuitBreaker_ReturnsErrOpenWhenOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := newBreakerClient(1, time.Hour)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)

	resp, _ := c.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	_, err := c.Do(req)
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
}

func TestNoBreaker_TransportErrorPassesThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := NewClient() // no circuit breaker
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)

	_, err := c.Do(req)
	if err == nil {
		t.Fatal("expected transport error, got nil")
	}
	if errors.Is(err, ErrOpen) {
		t.Fatal("expected real transport error, not ErrOpen")
	}
}
