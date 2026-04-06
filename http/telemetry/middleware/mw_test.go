package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelemetryMiddleware_PassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	TelemetryMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestTelemetryMiddleware_CapturesErrorStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	TelemetryMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 to pass through, got %d", rec.Code)
	}
}

func TestTelemetryMiddleware_RecoversPanic(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected error in handler")
	})

	// Must not propagate the panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic escaped telemetry middleware: %v", r)
		}
	}()

	TelemetryMiddleware(next).ServeHTTP(rec, req)

	// Telemetry recovery sends 500 when no status was written before the panic.
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic recovery, got %d", rec.Code)
	}
}

// TestTelemetryMiddleware_RecoversPanicWithStatusAlreadyWritten covers the
// panic-recovery branch where a status was already written before the panic
// (status != 0), so no additional 500 is sent.
// TestTelemetryMiddleware_HandlerWritesNoStatus covers the non-panic branch where
// next does not call WriteHeader or Write, leaving resWtr.Status == 0 and causing
// the middleware to default to http.StatusOK.
func TestTelemetryMiddleware_HandlerWritesNoStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// next writes nothing — status remains 0 until the defer sets it to 200.
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	TelemetryMiddleware(next).ServeHTTP(rec, req)

	// httptest.ResponseRecorder defaults to 200 when nothing is written.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when no status written, got %d", rec.Code)
	}
}

func TestTelemetryMiddleware_RecoversPanicWithStatusAlreadyWritten(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write a status before panicking so resWtr.Status != 0 at recovery time.
		w.WriteHeader(http.StatusOK)
		panic("panic after status written")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic escaped telemetry middleware: %v", r)
		}
	}()

	TelemetryMiddleware(next).ServeHTTP(rec, req)

	// Status 200 was written before the panic; the recovery must not overwrite it.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (written before panic), got %d", rec.Code)
	}
}

func TestTelemetryMiddleware_MultipleHTTPMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			TelemetryMiddleware(next).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", method, rec.Code)
			}
		})
	}
}
