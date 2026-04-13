package mwchain

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain_NoMiddleware(t *testing.T) {
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Chain(handler).ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestChain_SingleMiddleware(t *testing.T) {
	var order []string

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw")
			next.ServeHTTP(w, r)
		})
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Chain(handler, mw).ServeHTTP(rec, req)

	if len(order) != 2 || order[0] != "mw" || order[1] != "handler" {
		t.Errorf("unexpected call order: %v", order)
	}
}

func TestChain_MultipleMiddlewareOrder(t *testing.T) {
	// Chain(h, mw1, mw2) → mw1 is outermost, executes before mw2 before handler.
	var order []string

	make := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-in")
				next.ServeHTTP(w, r)
				order = append(order, name+"-out")
			})
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Chain(handler, make("mw1"), make("mw2")).ServeHTTP(rec, req)

	expected := []string{"mw1-in", "mw2-in", "handler", "mw2-out", "mw1-out"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("step %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestChain_ResponsePropagates(t *testing.T) {
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-MW", "applied")
			next.ServeHTTP(w, r)
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Chain(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}, mw).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if rec.Header().Get("X-MW") != "applied" {
		t.Error("expected X-MW header from middleware")
	}
}
