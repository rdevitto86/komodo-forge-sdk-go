package clienttype

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func captureClientType(req *http.Request) string {
	var captured string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v, ok := r.Context().Value(ctxKeys.CLIENT_TYPE_KEY).(string); ok {
			captured = v
		}
		w.WriteHeader(http.StatusOK)
	})
	ClientTypeMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)
	return captured
}

func TestClientTypeMiddleware_AuthOnlyIsAPI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	// No Referer or Cookie → API client
	if got := captureClientType(req); got != ClientTypeAPI {
		t.Errorf("expected %q, got %q", ClientTypeAPI, got)
	}
}

func TestClientTypeMiddleware_AuthWithRefererIsBrowser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Referer", "https://example.com")
	if got := captureClientType(req); got != ClientTypeBrowser {
		t.Errorf("expected %q, got %q", ClientTypeBrowser, got)
	}
}

func TestClientTypeMiddleware_AuthWithCookieIsBrowser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Cookie", "session=abc123")
	if got := captureClientType(req); got != ClientTypeBrowser {
		t.Errorf("expected %q, got %q", ClientTypeBrowser, got)
	}
}

func TestClientTypeMiddleware_NoHeadersIsBrowser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := captureClientType(req); got != ClientTypeBrowser {
		t.Errorf("expected %q, got %q", ClientTypeBrowser, got)
	}
}

func TestClientTypeMiddleware_RefererOnlyIsBrowser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Referer", "https://example.com")
	if got := captureClientType(req); got != ClientTypeBrowser {
		t.Errorf("expected %q, got %q", ClientTypeBrowser, got)
	}
}

func TestClientTypeMiddleware_SetsContextAndCallsNext(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	ClientTypeMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}
