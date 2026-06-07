package csrf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rdevitto86/komodo-forge-sdk-go/api/headers"
	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func withClientType(req *http.Request, clientType string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), ctxKeys.CLIENT_TYPE_KEY, clientType))
}

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

// Verified API clients (CLIENT_TYPE_KEY set in context by AuthMiddleware) are exempt from CSRF.
func TestCSRFMiddleware_APIClientExempt(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := withClientType(httptest.NewRequest(method, "/", nil), "api")
			rec := httptest.NewRecorder()

			CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200 for API client, got %d", method, rec.Code)
			}
		})
	}
}

// An unverified X-API-Key header alone does not exempt a request — only a verified context value does.
func TestCSRFMiddleware_ForgedAPIKeyHeaderDoesNotExempt(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-API-Key", "service-api-key")
	rec := httptest.NewRecorder()

	CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 — an X-API-Key header must not exempt a request from CSRF, got %d", rec.Code)
	}
}

// Browser client POST with a CSRF header matching the issued cookie passes (double-submit).
func TestCSRFMiddleware_BrowserWithValidToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/", CSRFMiddleware(okHandler()))

	get := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, get)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == headers.COOKIE_CSRF_TOKEN {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("expected CSRFMiddleware to issue a csrf_token cookie")
	}

	post := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	post.Header.Set("X-CSRF-Token", token)
	post.AddCookie(&http.Cookie{Name: headers.COOKIE_CSRF_TOKEN, Value: token})
	rec := httptest.NewRecorder()

	CSRFMiddleware(okHandler()).ServeHTTP(rec, post)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for browser with matching double-submit token, got %d", rec.Code)
	}
}

// A header token that doesn't match the cookie is rejected — guards against a forged header alone.
func TestCSRFMiddleware_BrowserWithMismatchedToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	req.Header.Set("X-CSRF-Token", "forged-token")
	req.AddCookie(&http.Cookie{Name: headers.COOKIE_CSRF_TOKEN, Value: "issued-token"})
	rec := httptest.NewRecorder()

	CSRFMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for mismatched double-submit token, got %d", rec.Code)
	}
}

// Browser client POST without CSRF token is rejected.
func TestCSRFMiddleware_BrowserWithoutToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/items", nil)
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
