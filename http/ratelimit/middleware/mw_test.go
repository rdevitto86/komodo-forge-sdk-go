package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rdevitto86/komodo-forge-sdk-go/config"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimiterMiddleware_AllowsFirstRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	rec := httptest.NewRecorder()

	RateLimiterMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for first request, got %d", rec.Code)
	}
}

func TestRateLimiterMiddleware_DeniesAfterBurstExhausted(t *testing.T) {
	// Each client key gets its own token bucket (default burst = 20).
	// Use a unique IP so this test has an isolated bucket.
	const testIP = "198.51.100.200"
	handler := RateLimiterMiddleware(okHandler())

	// Exhaust the burst window (21 requests: requests 0-19 are allowed, 20 is denied).
	var lastCode int
	for i := 0; i <= 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", testIP)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		lastCode = rec.Code
	}

	if lastCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after burst exhaustion, got %d", lastCode)
	}
}

func TestRateLimiterMiddleware_SetsRetryAfterOnDeny(t *testing.T) {
	const testIP = "198.51.100.201"
	handler := RateLimiterMiddleware(okHandler())

	var rec *httptest.ResponseRecorder
	for i := 0; i <= 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", testIP)
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	if rec.Code == http.StatusTooManyRequests {
		if rec.Header().Get("Retry-After") == "" {
			t.Error("expected Retry-After header on 429 response")
		}
	}
}

func TestRateLimiterMiddleware_IsolatesBucketsByKey(t *testing.T) {
	// Two distinct IPs must have independent buckets; first request from each is allowed.
	for _, ip := range []string{"198.51.100.10", "198.51.100.11"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()

		RateLimiterMiddleware(okHandler()).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", ip, rec.Code)
		}
	}
}

// TestRateLimiterMiddleware_FailOpen covers the err!=nil + ShouldFailOpen()==true branch.
// Setting ENV=prod causes ratelimit.Allow to call AllowDistributed which errors because
// no Elasticache client is configured; with RATE_LIMIT_FAIL_OPEN=true the request passes.
func TestRateLimiterMiddleware_FailOpen(t *testing.T) {
	config.SetConfigValue("ENV", "prod")
	config.SetConfigValue("RATE_LIMIT_FAIL_OPEN", "true")
	defer func() {
		config.DeleteConfigValue("ENV")
		config.DeleteConfigValue("RATE_LIMIT_FAIL_OPEN")
	}()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.77")
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	RateLimiterMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called in fail-open mode")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 in fail-open mode, got %d", rec.Code)
	}
}

// TestRateLimiterMiddleware_FailClosed covers the err!=nil + ShouldFailOpen()==false branch.
// Setting ENV=prod causes ratelimit.Allow to error; with RATE_LIMIT_FAIL_OPEN=false the
// middleware returns 500.
func TestRateLimiterMiddleware_FailClosed(t *testing.T) {
	config.SetConfigValue("ENV", "prod")
	config.SetConfigValue("RATE_LIMIT_FAIL_OPEN", "false")
	defer func() {
		config.DeleteConfigValue("ENV")
		config.DeleteConfigValue("RATE_LIMIT_FAIL_OPEN")
	}()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.88")
	rec := httptest.NewRecorder()

	RateLimiterMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 in fail-closed mode, got %d", rec.Code)
	}
}
