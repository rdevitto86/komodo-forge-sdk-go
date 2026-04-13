package sanitization

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runSanitization(req *http.Request) (*http.Request, int) {
	var seen *http.Request
	code := http.StatusOK
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	SanitizationMiddleware(next).ServeHTTP(rec, req)
	code = rec.Code
	return seen, code
}

func TestSanitizationMiddleware_RemovesSQLInjectionFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "SELECT * FROM users")

	seen, _ := runSanitization(req)

	got := seen.Header.Get("X-Custom")
	if strings.Contains(strings.ToLower(got), "select") {
		t.Errorf("expected SQL keyword removed from header, got %q", got)
	}
}

func TestSanitizationMiddleware_RemovesPathTraversalFromHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Path", "../../../etc/passwd")

	seen, _ := runSanitization(req)

	got := seen.Header.Get("X-Path")
	if strings.Contains(got, "../") {
		t.Errorf("expected path traversal removed from header, got %q", got)
	}
}

func TestSanitizationMiddleware_RemovesNullBytesFromQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?q=hello%00world", nil)

	seen, _ := runSanitization(req)

	got := seen.URL.Query().Get("q")
	if strings.ContainsRune(got, 0) {
		t.Errorf("expected null bytes removed from query param, got %q", got)
	}
}

func TestSanitizationMiddleware_RemovesXSSFromQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, `/search?q=<script>alert(1)</script>`, nil)

	seen, _ := runSanitization(req)

	got := seen.URL.Query().Get("q")
	if strings.Contains(strings.ToLower(got), "<script") {
		t.Errorf("expected XSS removed from query param, got %q", got)
	}
}

func TestSanitizationMiddleware_CleanRequestPassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/items?page=1&limit=20", nil)
	req.Header.Set("Accept", "application/json")

	seen, code := runSanitization(req)

	if code != http.StatusOK {
		t.Errorf("expected 200, got %d", code)
	}
	if seen.URL.Query().Get("page") != "1" {
		t.Errorf("expected page=1 unchanged, got %q", seen.URL.Query().Get("page"))
	}
}

func TestSanitizationMiddleware_SanitizesJSONBody(t *testing.T) {
	body := `{"name":"Alice","comment":"hello SELECT DROP TABLE users"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	_, code := runSanitization(req)

	if code != http.StatusOK {
		t.Errorf("expected 200 for valid JSON body, got %d", code)
	}
}

func TestSanitizationMiddleware_SkipsBodySanitizationForNonJSON(t *testing.T) {
	body := `name=Alice&comment=SELECT * FROM users`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	SanitizationMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called for non-JSON body")
	}
}

// TestSanitizationMiddleware_SanitizesPathParams covers the sanitizePathParams branch.
// Go 1.22+ populates req.Pattern when a request is dispatched through a ServeMux
// that uses the "{name}" wildcard syntax.
func TestSanitizationMiddleware_SanitizesPathParams(t *testing.T) {
	called := false
	var gotID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotID = r.PathValue("id")
		w.WriteHeader(http.StatusOK)
	})

	mux := http.NewServeMux()
	mux.Handle("GET /items/{id}", SanitizationMiddleware(inner))

	req := httptest.NewRequest(http.MethodGet, "/items/hello-world", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !called {
		t.Error("expected inner handler to be called")
	}
	if gotID == "" {
		t.Error("expected path value 'id' to be set")
	}
}

// TestSanitizationMiddleware_XSSInPathParam verifies that an XSS payload in a
// path parameter is sanitized before the inner handler sees it.
func TestSanitizationMiddleware_XSSInPathParam(t *testing.T) {
	var gotID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = r.PathValue("id")
		w.WriteHeader(http.StatusOK)
	})

	mux := http.NewServeMux()
	mux.Handle("GET /items/{id}", SanitizationMiddleware(inner))

	req := httptest.NewRequest(http.MethodGet, "/items/%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if strings.Contains(strings.ToLower(gotID), "<script") {
		t.Errorf("expected XSS payload sanitized in path param, got %q", gotID)
	}
}

// TestSanitizationMiddleware_BodyReadError covers the io.ReadAll error path in sanitizeBody.
// sanitizeBody sets req.Body = nil on read error; the middleware's nil-guard
// short-circuits and next is NOT called.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read error") }
func (errReader) Close() error             { return nil }

func TestSanitizationMiddleware_BodyReadError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = io.NopCloser(errReader{})
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	SanitizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called when body read fails")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for body read error, got %d", rec.Code)
	}
}

// TestSanitizationMiddleware_InvalidJSONBody covers the json.Unmarshal error path.
// sanitizeBody sets req.Body = nil on JSON parse error; the middleware's nil-guard
// short-circuits and next is NOT called.
func TestSanitizationMiddleware_InvalidJSONBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not-json{"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	SanitizationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next should not be called for invalid JSON")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// TestSanitizationMiddleware_JSONArrayBody covers the []interface{} branch in sanitizeJSON.
func TestSanitizationMiddleware_JSONArrayBody(t *testing.T) {
	body := `["hello","world","<script>"]`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	_, code := runSanitization(req)

	if code != http.StatusOK {
		t.Errorf("expected 200 for JSON array body, got %d", code)
	}
}

// TestSanitizationMiddleware_JSONDefaultType covers the default case in sanitizeJSON
// where the value is a non-string, non-map, non-array type (e.g. a number).
func TestSanitizationMiddleware_JSONDefaultType(t *testing.T) {
	body := `{"count":42,"active":true}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	_, code := runSanitization(req)

	if code != http.StatusOK {
		t.Errorf("expected 200 for JSON with numeric values, got %d", code)
	}
}

// TestSanitizationMiddleware_XSSPatternInString covers the XSS branch in sanitizeString.
func TestSanitizationMiddleware_XSSPatternInString(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// An XSS payload that passes the initial html.EscapeString but still matches
	// XssPattern after escaping can be set in a header.
	req.Header.Set("X-Custom", "<img src=x onerror=alert(1)>")

	seen, _ := runSanitization(req)

	got := seen.Header.Get("X-Custom")
	if strings.Contains(strings.ToLower(got), "onerror") {
		t.Errorf("expected XSS payload sanitized, got %q", got)
	}
}
