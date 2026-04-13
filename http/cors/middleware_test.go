package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_PassesThrough(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()
			called := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			CORSMiddleware(next).ServeHTTP(rec, req)

			if !called {
				t.Errorf("expected next to be called for %s", method)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected 200 for %s, got %d", method, rec.Code)
			}
		})
	}
}
