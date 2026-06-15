package headers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func statusForContentLength(mw func(http.Handler) http.Handler, contentLength int64) int {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.ContentLength = contentLength
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec.Code
}

func TestMaxContentLength_ExplicitWinsOverEnv(t *testing.T) {
	t.Setenv("MAX_CONTENT_LENGTH", "10")
	mw := MaxContentLengthMiddleware(2048)

	if code := statusForContentLength(mw, 100); code != http.StatusOK {
		t.Errorf("explicit 2048 should allow CL=100 regardless of env=10, got %d", code)
	}
	if code := statusForContentLength(mw, 5000); code != http.StatusRequestEntityTooLarge {
		t.Errorf("explicit 2048 should reject CL=5000, got %d", code)
	}
}

func TestMaxContentLength_EnvWhenNoExplicit(t *testing.T) {
	t.Setenv("MAX_CONTENT_LENGTH", "16")
	mw := MaxContentLengthMiddleware(0)

	if code := statusForContentLength(mw, 8); code != http.StatusOK {
		t.Errorf("env=16 should allow CL=8, got %d", code)
	}
	if code := statusForContentLength(mw, 50); code != http.StatusRequestEntityTooLarge {
		t.Errorf("env=16 should reject CL=50, got %d", code)
	}
}

func TestMaxContentLength_DefaultWhenEnvUnset(t *testing.T) {
	t.Setenv("MAX_CONTENT_LENGTH", "")
	mw := MaxContentLengthMiddleware(0)

	if code := statusForContentLength(mw, DEFAULT_MAX_CONTENT_LENGTH-1); code != http.StatusOK {
		t.Errorf("default should allow CL just under %d, got %d", DEFAULT_MAX_CONTENT_LENGTH, code)
	}
	if code := statusForContentLength(mw, DEFAULT_MAX_CONTENT_LENGTH+1); code != http.StatusRequestEntityTooLarge {
		t.Errorf("default should reject CL just over %d, got %d", DEFAULT_MAX_CONTENT_LENGTH, code)
	}
}

func TestMaxContentLength_DefaultWhenEnvInvalid(t *testing.T) {
	t.Setenv("MAX_CONTENT_LENGTH", "not-a-number")
	mw := MaxContentLengthMiddleware(0)

	if code := statusForContentLength(mw, 100); code != http.StatusOK {
		t.Errorf("invalid env should fall back to default and allow CL=100, got %d", code)
	}
	if code := statusForContentLength(mw, DEFAULT_MAX_CONTENT_LENGTH+1); code != http.StatusRequestEntityTooLarge {
		t.Errorf("invalid env should fall back to default and reject oversize, got %d", code)
	}
}

func TestMaxContentLengthMiddleware_UnderLimit_CallsNext(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()

	MaxContentLengthMiddleware(1024)(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called for a body under the limit")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMaxContentLengthMiddleware_DeclaredOverLimit_Returns413(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called when Content-Length exceeds the limit")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 100)))
	rec := httptest.NewRecorder()

	MaxContentLengthMiddleware(16)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rec.Code)
	}
}

func TestMaxContentLengthMiddleware_UnderstatedLength_BodyCappedByMaxBytesReader(t *testing.T) {
	var readErr error
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 100)))
	req.ContentLength = 4

	rec := httptest.NewRecorder()
	MaxContentLengthMiddleware(16)(next).ServeHTTP(rec, req)

	if readErr == nil {
		t.Error("expected MaxBytesReader to error when the actual body exceeds the limit despite an understated Content-Length")
	}
}
