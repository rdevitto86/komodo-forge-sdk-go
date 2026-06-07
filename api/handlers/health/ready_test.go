package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func countingChecker(name string, err error, calls *int32) Checker {
	return CheckerFunc(name, func(ctx context.Context) error {
		atomic.AddInt32(calls, 1)
		return err
	})
}

func doReady(t *testing.T, handler http.HandlerFunc) (*http.Response, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	resp := rec.Result()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return resp, body
}

func TestNewReadyHandler_AllHealthy(t *testing.T) {
	var calls int32
	handler := NewReadyHandler([]Checker{
		countingChecker("dep-a", nil, &calls),
		countingChecker("dep-b", nil, &calls),
	})

	resp, body := doReady(t, handler)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if body["status"] != "OK" {
		t.Fatalf("expected status OK in body, got %v", body)
	}
}

func TestNewReadyHandler_OneFailingMakesAllUnhealthy(t *testing.T) {
	var calls int32
	handler := NewReadyHandler([]Checker{
		countingChecker("dep-a", nil, &calls),
		countingChecker("dep-b", errors.New("connection refused"), &calls),
	})

	resp, body := doReady(t, handler)

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", resp.StatusCode)
	}

	failing, ok := body["failing"].([]any)
	if !ok || len(failing) != 1 {
		t.Fatalf("expected exactly one failing dep, got %v", body)
	}
	entry := failing[0].(map[string]any)
	if entry["dep"] != "dep-b" {
		t.Fatalf("expected failing dep 'dep-b', got %v", entry["dep"])
	}
	if entry["error"] != "connection refused" {
		t.Fatalf("expected verbatim error message, got %v", entry["error"])
	}
}

func TestNewReadyHandler_CachesResultsWithinTTL(t *testing.T) {
	var calls int32
	handler := NewReadyHandler(
		[]Checker{countingChecker("dep-a", nil, &calls)},
		WithCacheTTL(time.Hour),
	)

	doReady(t, handler)
	doReady(t, handler)
	doReady(t, handler)

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected checker to be probed once and reused from cache, got %d calls", got)
	}
}

func TestNewReadyHandler_ReprobesAfterTTLExpires(t *testing.T) {
	var calls int32
	handler := NewReadyHandler(
		[]Checker{countingChecker("dep-a", nil, &calls)},
		WithCacheTTL(time.Millisecond),
	)

	doReady(t, handler)
	time.Sleep(5 * time.Millisecond)
	doReady(t, handler)

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected checker to be re-probed after TTL expiry, got %d calls", got)
	}
}

func TestNewReadyHandler_AppliesCheckTimeout(t *testing.T) {
	blocked := CheckerFunc("slow-dep", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	handler := NewReadyHandler([]Checker{blocked}, WithCheckTimeout(10*time.Millisecond))

	start := time.Now()
	resp, body := doReady(t, handler)
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", resp.StatusCode)
	}
	if elapsed > time.Second {
		t.Fatalf("expected handler to respect the per-check timeout, took %s", elapsed)
	}

	failing := body["failing"].([]any)
	if len(failing) != 1 || failing[0].(map[string]any)["dep"] != "slow-dep" {
		t.Fatalf("expected the slow dependency reported as failing, got %v", body)
	}
}

func TestCheckerFunc_AdaptsPlainFunction(t *testing.T) {
	c := CheckerFunc("custom", func(ctx context.Context) error { return nil })

	if c.Name() != "custom" {
		t.Fatalf("expected name 'custom', got %q", c.Name())
	}
	if err := c.Check(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
