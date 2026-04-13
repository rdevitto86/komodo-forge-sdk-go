package idempotency

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rdevitto86/komodo-forge-sdk-go/config"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// clearStore removes all keys from the default store between tests.
func clearStore() {
	// Create a fresh local store for each test to ensure isolation
	testStore := NewStore("local", 0)
	SetStore(testStore)
}

// Safe HTTP methods skip the idempotency check entirely.
func TestIdempotencyMiddleware_SafeMethodsPassThrough(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()
			IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", method, rec.Code)
			}
		})
	}
}

// API clients (identified by X-API-Key) bypass the idempotency check.
func TestIdempotencyMiddleware_APIClientPassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "service-key")
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for API client, got %d", rec.Code)
	}
}

// Browser client with a valid idempotency key (8-64 alphanum chars) passes.
func TestIdempotencyMiddleware_BrowserValidKey(t *testing.T) {
	clearStore()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Idempotency-Key", "validkey") // exactly 8 chars
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid idempotency key, got %d", rec.Code)
	}
}

// Browser client with a key shorter than 8 characters is rejected.
func TestIdempotencyMiddleware_BrowserShortKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Idempotency-Key", "short") // 5 chars — below minimum
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short key, got %d", rec.Code)
	}
}

// Browser client with no idempotency key is rejected.
func TestIdempotencyMiddleware_BrowserMissingKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing key, got %d", rec.Code)
	}
}

// A duplicate (already-stored, non-expired) key is rejected with 409.
func TestIdempotencyMiddleware_BrowserDuplicateKey(t *testing.T) {
	clearStore()

	const key = "dup-key-abcd"
	// Pre-store with a future expiry to simulate a previous request.
	testStore := NewStore("local", 0)
	testStore.cache.(*LocalCache).Store(key, time.Now().Add(time.Hour).Unix(), 3600)
	SetStore(testStore)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate key, got %d", rec.Code)
	}
	if rec.Header().Get("Idempotency-Replayed") != "true" {
		t.Error("expected Idempotency-Replayed: true header on duplicate")
	}
}

// An expired store entry is evicted and the request proceeds normally.
func TestIdempotencyMiddleware_ExpiredKeyEvictedAndAllowed(t *testing.T) {
	clearStore()

	const key = "expired-key-x"
	// Pre-store with a past expiry.
	testStore := NewStore("local", 0)
	testStore.cache.(*LocalCache).Store(key, time.Now().Add(-time.Hour).Unix(), 3600)
	SetStore(testStore)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Idempotency-Key", key)
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for expired key (evicted), got %d", rec.Code)
	}
}

// TestGetIdemTTL_DefaultWhenNoEnv verifies getIdemTTL returns the default when
// IDEMPOTENCY_TTL_SEC is not set.
func TestGetIdemTTL_DefaultWhenNoEnv(t *testing.T) {
	config.DeleteConfigValue("IDEMPOTENCY_TTL_SEC")
	if got := getIdemTTL(); got != DEFAULT_IDEM_TTL_SEC {
		t.Errorf("expected default TTL %d, got %d", DEFAULT_IDEM_TTL_SEC, got)
	}
}

// TestGetIdemTTL_ValidPositiveValue verifies getIdemTTL parses a valid positive duration.
func TestGetIdemTTL_ValidPositiveValue(t *testing.T) {
	config.SetConfigValue("IDEMPOTENCY_TTL_SEC", "600")
	defer config.DeleteConfigValue("IDEMPOTENCY_TTL_SEC")

	if got := getIdemTTL(); got != 600 {
		t.Errorf("expected TTL 600, got %d", got)
	}
}

// TestGetIdemTTL_ZeroOrNegativeReturnsDefault verifies that a non-positive value
// falls back to the 300-second default guard.
func TestGetIdemTTL_ZeroOrNegativeReturnsDefault(t *testing.T) {
	// "0s" parses to 0 duration which triggers the <= 0 guard.
	config.SetConfigValue("IDEMPOTENCY_TTL_SEC", "0")
	defer config.DeleteConfigValue("IDEMPOTENCY_TTL_SEC")

	if got := getIdemTTL(); got != 300 {
		t.Errorf("expected 300 for zero/negative TTL, got %d", got)
	}
}
