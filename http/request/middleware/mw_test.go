package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	var ctxID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID, _ = r.Context().Value(ctxKeys.REQUEST_ID_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if ctxID == "" {
		t.Error("expected non-empty request ID in context")
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID in response header")
	}
	if rec.Header().Get("X-Request-ID") != ctxID {
		t.Errorf("response header %q does not match context ID %q",
			rec.Header().Get("X-Request-ID"), ctxID)
	}
}

func TestRequestIDMiddleware_UsesClientSuppliedID(t *testing.T) {
	const supplied = "client-request-id-123"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", supplied)
	rec := httptest.NewRecorder()
	var ctxID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID, _ = r.Context().Value(ctxKeys.REQUEST_ID_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if ctxID != supplied {
		t.Errorf("expected context ID %q, got %q", supplied, ctxID)
	}
	if rec.Header().Get("X-Request-ID") != supplied {
		t.Errorf("expected response header %q, got %q", supplied, rec.Header().Get("X-Request-ID"))
	}
}

func TestRequestIDMiddleware_PropagatesCorrelationID(t *testing.T) {
	const corrID = "browser-session-abc"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Correlation-ID", corrID)
	rec := httptest.NewRecorder()
	var ctxCorrID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxCorrID, _ = r.Context().Value(ctxKeys.CORRELATION_ID_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if ctxCorrID != corrID {
		t.Errorf("expected correlation ID %q in context, got %q", corrID, ctxCorrID)
	}
	if rec.Header().Get("X-Correlation-ID") != corrID {
		t.Errorf("expected X-Correlation-ID %q in response, got %q", corrID, rec.Header().Get("X-Correlation-ID"))
	}
}

func TestRequestIDMiddleware_NoCorrelationIDWhenAbsent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Correlation-ID"); got != "" {
		t.Errorf("expected no X-Correlation-ID, got %q", got)
	}
}

// TestRequestIDMiddleware_UsesContextRequestID covers the branch where no
// X-Request-ID header is present but the context carries REQUEST_ID_KEY.
func TestRequestIDMiddleware_UsesContextRequestID(t *testing.T) {
	const ctxID = "ctx-id-12345"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ctxKeys.REQUEST_ID_KEY, ctxID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	var gotID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = r.Context().Value(ctxKeys.REQUEST_ID_KEY).(string)
		w.WriteHeader(http.StatusOK)
	})

	RequestIDMiddleware(next).ServeHTTP(rec, req)

	if gotID != ctxID {
		t.Errorf("expected context request ID %q, got %q", ctxID, gotID)
	}
	if rec.Header().Get("X-Request-ID") != ctxID {
		t.Errorf("expected X-Request-ID %q in response, got %q", ctxID, rec.Header().Get("X-Request-ID"))
	}
}
