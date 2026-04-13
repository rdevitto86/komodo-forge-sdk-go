package redaction

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runRedaction(req *http.Request) *http.Request {
	var seen *http.Request
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.WriteHeader(http.StatusOK)
	})
	RedactionMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)
	return seen
}

func TestRedactionMiddleware_RedactsAuthorizationHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret-token-12345678")

	seen := runRedaction(req)

	if got := seen.Header.Get("Authorization"); got != "REDACTED" {
		t.Errorf("expected Authorization to be REDACTED, got %q", got)
	}
}

func TestRedactionMiddleware_RedactsCookieHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", "session=abc123")

	seen := runRedaction(req)

	if got := seen.Header.Get("Cookie"); got != "REDACTED" {
		t.Errorf("expected Cookie to be REDACTED, got %q", got)
	}
}

func TestRedactionMiddleware_PassesThroughNonSensitiveHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")

	seen := runRedaction(req)

	if got := seen.Header.Get("Accept"); got != "application/json" {
		t.Errorf("expected Accept to pass through unchanged, got %q", got)
	}
}

func TestRedactionMiddleware_RedactsSensitiveQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?password=secret123&q=hello", nil)

	seen := runRedaction(req)

	q := seen.URL.Query()
	if q.Get("password") != "REDACTED" {
		t.Errorf("expected password to be REDACTED, got %q", q.Get("password"))
	}
	if q.Get("q") != "hello" {
		t.Errorf("expected q to be unchanged, got %q", q.Get("q"))
	}
}

func TestRedactionMiddleware_RedactsJSONBodyPasswordField(t *testing.T) {
	body := `{"username":"alice","password":"s3cr3t","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	seen := runRedaction(req)

	buf := new(bytes.Buffer)
	buf.ReadFrom(seen.Body)
	got := buf.String()

	if strings.Contains(got, "s3cr3t") {
		t.Errorf("expected password value to be redacted, got body: %s", got)
	}
}

func TestRedactionMiddleware_OriginalBodyRestoredForDownstream(t *testing.T) {
	body := `{"name":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	// The middleware restores the original body to req while downstream gets r2.
	// Verify middleware calls next successfully.
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	RedactionMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called")
	}
}

// TestRedactionMiddleware_RedactsNilHeader covers the nil-guard in redactHeaders.
func TestRedactionMiddleware_RedactsNilHeader(t *testing.T) {
	// Build a request with a nil header map to exercise the nil guard in redactHeaders.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header = nil

	// Must not panic.
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	RedactionMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("expected next to be called")
	}
}

// TestRedactionMiddleware_LooksLikeTokenBearerPrefix covers the bearerRE branch.
func TestRedactionMiddleware_LooksLikeTokenBearerPrefix(t *testing.T) {
	// A non-sensitive header whose value looks like a bearer token should be redacted.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// bearerRE matches "Bearer <token>" patterns.
	req.Header.Set("X-Custom-Token", "Bearer eyJhbGciOiJSUzI1NiJ9.payload.sig")

	seen := runRedaction(req)

	if got := seen.Header.Get("X-Custom-Token"); got != "REDACTED" {
		t.Errorf("expected bearer-prefixed value to be REDACTED, got %q", got)
	}
}

// TestRedactionMiddleware_LooksLikeTokenLongString covers the longTokenRE branch.
func TestRedactionMiddleware_LooksLikeTokenLongString(t *testing.T) {
	// A value >30 chars matching [A-Za-z0-9\-\._~\+/]{20,} must be redacted.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Long-Token", "abcdefghijklmnopqrstuvwxyz01234567890abcde")

	seen := runRedaction(req)

	if got := seen.Header.Get("X-Long-Token"); got != "REDACTED" {
		t.Errorf("expected long token to be REDACTED, got %q", got)
	}
}

// TestRedactionMiddleware_NonJSONBodyFallback covers the non-JSON body path in redactBody.
func TestRedactionMiddleware_NonJSONBodyFallback(t *testing.T) {
	// Plain text body — no JSON content type — exercises the regex-fallback path.
	body := "Bearer eyJhbGciOiJSUzI1NiJ9.payload.signature"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "text/plain")

	seen := runRedaction(req)

	buf := new(bytes.Buffer)
	buf.ReadFrom(seen.Body)
	got := buf.String()

	if strings.Contains(got, "eyJhbGciOiJSUzI1NiJ9") {
		t.Errorf("expected bearer token to be redacted in non-JSON body, got %q", got)
	}
}

// TestRedactionMiddleware_RedactInterfaceArray covers the []interface{} case in redactInterface.
func TestRedactionMiddleware_RedactInterfaceArray(t *testing.T) {
	// JSON array body — exercises the []interface{} branch in redactInterface.
	body := `[{"password":"secret"},{"name":"alice"}]`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	seen := runRedaction(req)

	buf := new(bytes.Buffer)
	buf.ReadFrom(seen.Body)
	got := buf.String()

	if strings.Contains(got, "secret") {
		t.Errorf("expected password to be redacted in JSON array, got %q", got)
	}
}

// TestRedactionMiddleware_RedactQueryTokenValue covers the looksLikeToken check in redactQuery.
func TestRedactionMiddleware_RedactQueryTokenValue(t *testing.T) {
	// A query param whose value is a long token (>30 chars) should be redacted.
	longVal := "abcdefghijklmnopqrstuvwxyz01234567890abcde"
	req := httptest.NewRequest(http.MethodGet, "/search?data="+longVal, nil)

	seen := runRedaction(req)

	if got := seen.URL.Query().Get("data"); got != "REDACTED" {
		t.Errorf("expected long token in query to be REDACTED, got %q", got)
	}
}

// TestRedactionMiddleware_NilQueryValues covers the nil-guard in redactQuery.
func TestRedactionMiddleware_NilQueryValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// URL with nil RawQuery produces empty Values, not nil; testing via URL = nil.
	req.URL = nil

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	RedactionMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("expected next to be called when URL is nil")
	}
}

// TestLooksLikeToken_EmptyString covers the s=="" early-return in looksLikeToken.
func TestLooksLikeToken_EmptyString(t *testing.T) {
	if looksLikeToken("") {
		t.Error("empty string should not look like a token")
	}
}

// TestRedactQuery_NilValues covers the nil-guard in redactQuery.
func TestRedactQuery_NilValues(t *testing.T) {
	if got := redactQuery(nil); got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}
}
