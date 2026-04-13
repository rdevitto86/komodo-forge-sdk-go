package csrf

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// Safe methods bypass CSRF checks entirely.
func TestCSRFMiddleware_SafeMethodsPassThrough(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", method, rec.Code)
			}
		})
	}
}

// API clients (identified by X-API-Key) are exempt from CSRF.
func TestCSRFMiddleware_APIClientExempt(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			req.Header.Set("X-API-Key", "service-api-key")
			rec := httptest.NewRecorder()

			CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200 for API client, got %d", method, rec.Code)
			}
		})
	}
}

// Browser client POST with a valid (non-empty) CSRF token passes.
func TestCSRFMiddleware_BrowserWithValidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	req.Header.Set("X-CSRF-Token", "csrf-token-value-here")
	rec := httptest.NewRecorder()

	CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for browser with valid CSRF token, got %d", rec.Code)
	}
}

// Browser client POST without CSRF token is rejected.
func TestCSRFMiddleware_BrowserWithoutToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	// No X-CSRF-Token, no X-API-Key → browser client, no token
	rec := httptest.NewRecorder()

	CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for browser without CSRF token, got %d", rec.Code)
	}
}

// PUT and DELETE from browser without CSRF token are also rejected.
func TestCSRFMiddleware_BrowserMutatingMethodsRequireToken(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/resource/1", nil)
			rec := httptest.NewRecorder()

			CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d", method, rec.Code)
			}
		})
	}
}
