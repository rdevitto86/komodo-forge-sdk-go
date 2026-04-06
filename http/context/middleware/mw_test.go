package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func TestContextMiddleware_GeneratesRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/items", nil)
	rec := httptest.NewRecorder()
	var reqID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID, _ = r.Context().Value(ctxKeys.REQUEST_ID_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	ContextMiddleware(next).ServeHTTP(rec, req)

	if reqID == "" {
		t.Error("expected non-empty request ID in context")
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID in response header")
	}
}

func TestContextMiddleware_UsesExistingRequestID(t *testing.T) {
	const existing = "my-request-id"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", existing)
	var reqID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID, _ = r.Context().Value(ctxKeys.REQUEST_ID_KEY).(string)
	})

	ContextMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if reqID != existing {
		t.Errorf("expected %q, got %q", existing, reqID)
	}
}

func TestContextMiddleware_SetsStartTime(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/foo", nil)
	var startTime time.Time

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime, _ = r.Context().Value(ctxKeys.START_TIME_KEY).(time.Time)
	})

	ContextMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if startTime.IsZero() {
		t.Error("expected non-zero start time in context")
	}
}

func TestContextMiddleware_SetsMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/v1/resource/1", nil)
	var method string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, _ = r.Context().Value(ctxKeys.METHOD_KEY).(string)
	})

	ContextMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if method != http.MethodDelete {
		t.Errorf("expected DELETE, got %q", method)
	}
}

func TestContextMiddleware_SetsVersionFromURLPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v2/items", nil)
	var version string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version, _ = r.Context().Value(ctxKeys.VERSION_KEY).(string)
	})

	ContextMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if version != "/v2" {
		t.Errorf("expected \"/v2\", got %q", version)
	}
}

func TestContextMiddleware_CallsNext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	})

	ContextMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called")
	}
	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}

// TestContextMiddleware_UsesContextRequestID covers the branch where no X-Request-ID header
// is present but the context already carries a REQUEST_ID_KEY value.
func TestContextMiddleware_UsesContextRequestID(t *testing.T) {
	const existingID = "ctx-request-id-abc"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Inject the ID into context directly (no header).
	ctx := req.Context()
	ctx = context.WithValue(ctx, ctxKeys.REQUEST_ID_KEY, existingID)
	req = req.WithContext(ctx)

	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = r.Context().Value(ctxKeys.REQUEST_ID_KEY).(string)
	})

	ContextMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if gotID != existingID {
		t.Errorf("expected request ID %q from context, got %q", existingID, gotID)
	}
}
