package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func withClientType(req *http.Request, clientType string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), ctxKeys.CLIENT_TYPE_KEY, clientType))
}

// Installs a fresh local store so each test is isolated.
func clearStore() {
	testStore := NewStore("local", 0)
	SetStore(testStore)
}

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

func TestIdempotencyMiddleware_APIClientPassesThrough(t *testing.T) {
	req := withClientType(httptest.NewRequest(http.MethodPost, "/", nil), "api")
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for API client, got %d", rec.Code)
	}
}

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

func TestIdempotencyMiddleware_BrowserShortKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Idempotency-Key", "short") // 5 chars — below minimum
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short key, got %d", rec.Code)
	}
}

func TestIdempotencyMiddleware_BrowserMissingKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	IdempotencyMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing key, got %d", rec.Code)
	}
}

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

func TestGetIdemTTL_DefaultWhenNoEnv(t *testing.T) {
	os.Unsetenv("IDEMPOTENCY_TTL_SEC")
	if got := getIdemTTL(); got != DEFAULT_IDEM_TTL_SEC {
		t.Errorf("expected default TTL %d, got %d", DEFAULT_IDEM_TTL_SEC, got)
	}
}

func TestGetIdemTTL_ValidPositiveValue(t *testing.T) {
	os.Setenv("IDEMPOTENCY_TTL_SEC", "600")
	defer os.Unsetenv("IDEMPOTENCY_TTL_SEC")

	if got := getIdemTTL(); got != 600 {
		t.Errorf("expected TTL 600, got %d", got)
	}
}

func TestGetIdemTTL_ZeroOrNegativeReturnsDefault(t *testing.T) {
	// "0s" parses to 0 duration which triggers the <= 0 guard.
	os.Setenv("IDEMPOTENCY_TTL_SEC", "0")
	defer os.Unsetenv("IDEMPOTENCY_TTL_SEC")

	if got := getIdemTTL(); got != 300 {
		t.Errorf("expected 300 for zero/negative TTL, got %d", got)
	}
}
