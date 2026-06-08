package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// Returns the configured status codes in order, repeating the last once exhausted, and counts attempts served.
type sequenceTransport struct {
	mu       sync.Mutex
	attempts int
	statuses []int
	bodies   []string
}

func (s *sequenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	idx := s.attempts
	s.attempts++
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		req.Body.Close()
		s.bodies = append(s.bodies, string(data))
	}
	s.mu.Unlock()

	status := s.statuses[len(s.statuses)-1]
	if idx < len(s.statuses) {
		status = s.statuses[idx]
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}

func (s *sequenceTransport) attemptCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts
}

func TestRetry_SucceedsAfterTransientFailures(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{503, 502, 200}}
	c := NewClient(ClientConfig{
		Transport: transport,
		Retry:     &RetryConfig{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected final status 200, got %d", resp.StatusCode)
	}
	if got := transport.attemptCount(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestRetry_StopsOnNonRetryableStatus(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{400, 200}}
	c := NewClient(ClientConfig{
		Transport: transport,
		Retry:     &RetryConfig{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected the non-retryable 400 to be returned as-is, got %d", resp.StatusCode)
	}
	if got := transport.attemptCount(); got != 1 {
		t.Errorf("expected exactly 1 attempt for a non-retryable status, got %d", got)
	}
}

func TestRetry_ExhaustsMaxAttempts(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{503}}
	const maxAttempts = 3
	c := NewClient(ClientConfig{
		Transport: transport,
		Retry:     &RetryConfig{MaxAttempts: maxAttempts, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected the last attempt's 503 to be returned, got %d", resp.StatusCode)
	}
	if got := transport.attemptCount(); got != maxAttempts {
		t.Errorf("expected exactly %d attempts, got %d", maxAttempts, got)
	}
}

func TestRetry_AppliesExponentialBackoff(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{503, 503, 200}}
	base := 20 * time.Millisecond
	c := NewClient(ClientConfig{
		Transport: transport,
		Retry:     &RetryConfig{MaxAttempts: 5, BaseDelay: base, MaxDelay: time.Second},
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	start := time.Now()
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	// Two retries: delays of base and 2*base must both elapse before success.
	want := base + 2*base
	if elapsed := time.Since(start); elapsed < want {
		t.Errorf("expected backoff to take at least %v, took %v", want, elapsed)
	}
}

func TestRetry_ReplaysRequestBody(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{503, 200}}
	c := NewClient(ClientConfig{
		Transport: transport,
		Retry:     &RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
	})

	const payload = `{"hello":"world"}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.com", bytes.NewReader([]byte(payload)))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader([]byte(payload))), nil
	}

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	transport.mu.Lock()
	defer transport.mu.Unlock()
	if len(transport.bodies) != 2 {
		t.Fatalf("expected 2 attempts to have bodies recorded, got %d", len(transport.bodies))
	}
	for i, body := range transport.bodies {
		if body != payload {
			t.Errorf("attempt %d: expected replayed body %q, got %q", i+1, payload, body)
		}
	}
}

func TestRetry_StopsOnBreakerOpen(t *testing.T) {
	transport := &sequenceTransport{statuses: []int{503, 503, 200}}
	c := NewClient(ClientConfig{
		Transport:      transport,
		Retry:          &RetryConfig{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond},
		CircuitBreaker: &BreakerConfig{FailureThreshold: 1, SuccessThreshold: 1, OpenTimeout: time.Hour, MaxHalfOpenRequests: 1},
	})

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	_, err := c.Do(req)
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("expected ErrOpen once the breaker trips mid-retry, got %v", err)
	}
	// First attempt trips the breaker (failure #1 reaches the threshold of 1);
	// the second attempt is short-circuited by the now-open breaker, so retrying stops there.
	if got := transport.attemptCount(); got != 1 {
		t.Errorf("expected the breaker to short-circuit further retries after 1 upstream attempt, got %d", got)
	}
}

func BenchmarkClientDo_Retry(b *testing.B) {
	transport := &sequenceTransport{statuses: []int{200}}
	c := NewClient(ClientConfig{
		Transport: transport,
		Retry:     &RetryConfig{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond},
	})

	b.ReportAllocs()
	for b.Loop() {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
		resp, err := c.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
