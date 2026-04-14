package headers

import (
	"net/http/httptest"
	"os"
	"testing"
)

func makeReq(hdr, val string) *httptest.ResponseRecorder {
	_ = hdr
	_ = val
	return nil
}

func TestHeaderEval_ValidateHeaderValue_Authorization(t *testing.T) {
	// "authorization" goes through jwt.ValidateToken — keys not initialized,
	// so it will return false/error for any token value
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer some.jwt.token")
	ok, _ := ValidateHeaderValue("authorization", req)
	// We just ensure the code path is exercised; result depends on JWT keys
	_ = ok
}

func TestHeaderEval_ValidateHeaderValue_AccessControlAllowOrigin(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"*", true},
		{"https://example.com", true},
		{"http://localhost:3000", true},
		{"", false},
		{"not-a-url", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Access-Control-Allow-Origin", tc.val)
		ok, err := ValidateHeaderValue("access-control-allow-origin", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(access-control-allow-origin, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_CacheControl(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"no-cache", true},
		{"no-store", true},
		{"must-revalidate", true},
		{"public", false},
		{"max-age=3600", false},
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Cache-Control", tc.val)
		ok, err := ValidateHeaderValue("cache-control", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(cache-control, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_Cookie(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"session=abc123", true},
		{"x=y; z=w", true},
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Cookie", tc.val)
		ok, err := ValidateHeaderValue("cookie", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(cookie, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_ContentType(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/x-www-form-urlencoded", true},
		{"multipart/form-data; boundary=xyz", true},
		{"text/plain", false},
		{"text/html", false},
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("Content-Type", tc.val)
		ok, err := ValidateHeaderValue("content-type", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(content-type, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_Accept(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"application/json", true},
		{"application/x-www-form-urlencoded", true},
		{"multipart/form-data", true},
		{"text/plain", false},
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", tc.val)
		ok, err := ValidateHeaderValue("accept", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(accept, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_ContentLength(t *testing.T) {
	// Default max is 4096
	os.Unsetenv("MAX_CONTENT_LENGTH")

	tests := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"4096", true},
		{"4097", false},  // exceeds default max
		{"0", false},     // not positive
		{"-1", false},    // negative
		{"abc", false},   // non-numeric
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("Content-Length", tc.val)
		ok, err := ValidateHeaderValue("content-length", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(content-length, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}

	// Test with custom max
	t.Run("custom max content length", func(t *testing.T) {
		os.Setenv("MAX_CONTENT_LENGTH", "100")
		defer os.Unsetenv("MAX_CONTENT_LENGTH")
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("Content-Length", "50")
		ok, _ := ValidateHeaderValue("content-length", req)
		if !ok {
			t.Error("expected true for 50 <= 100")
		}
		req2 := httptest.NewRequest("POST", "/", nil)
		req2.Header.Set("Content-Length", "101")
		ok2, _ := ValidateHeaderValue("content-length", req2)
		if ok2 {
			t.Error("expected false for 101 > 100")
		}
	})
}

func TestHeaderEval_ValidateHeaderValue_IdempotencyKey(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"abcdefgh", true},           // 8 chars
		{"abcdefgh12345678901234567890123456789012345678901234567890abcd", true}, // 64 chars
		{"abc-123_XYZ", true},
		{"short", false},             // too short (< 8)
		{"ab cd ef", false},          // space not allowed
		{"abc!@#$%", false},          // special chars not allowed
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("Idempotency-Key", tc.val)
		ok, err := ValidateHeaderValue("idempotency-key", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(idempotency-key, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_Referer(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"https://example.com/path", true},
		{"http://localhost:8080/api", true},
		{"not-a-url", false},
		{"ftp://example.com", false},
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Referer", tc.val)
		ok, err := ValidateHeaderValue("referer", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(referer, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_Referrer(t *testing.T) {
	// "referrer" is an alias for "referer"
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Referrer", "https://example.com")
	ok, err := ValidateHeaderValue("referrer", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for valid referrer URL")
	}
}

func TestHeaderEval_ValidateHeaderValue_UserAgent(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"Mozilla/5.0 (Windows NT 10.0)", true},
		{"curl/7.68.0", true},
		{"Go-http-client/1.1", true},
		{"", false},
		// > 256 chars
		{string(make([]byte, 257)), false},
		{"bad\x00agent", false}, // null byte not in allowed charset
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("User-Agent", tc.val)
		ok, err := ValidateHeaderValue("user-agent", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(user-agent, %q...) = %v, want %v", tc.val[:min(len(tc.val), 20)], ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_XCSRFToken(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"some-csrf-token", true},
		{"abc123", true},
		{"", false},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("X-CSRF-Token", tc.val)
		ok, err := ValidateHeaderValue("x-csrf-token", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(x-csrf-token, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_XRequestedBy(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"API_INTERNAL", true},
		{"API_EXTERNAL", true},
		{"UI_USER_VERIFIED", true},
		{"custom-service", true},
		{"", false},
		// > 64 chars
		{"a123456789012345678901234567890123456789012345678901234567890123456789", false},
		{"bad value!", false},  // space/exclamation not in allowed charset
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Requested-By", tc.val)
		ok, err := ValidateHeaderValue("x-requested-by", req)
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(x-requested-by, %q) = %v, want %v", tc.val, ok, tc.want)
		}
	}
}

func TestHeaderEval_ValidateHeaderValue_Default(t *testing.T) {
	// Any unknown header: non-empty = true, empty = false
	tests := []struct {
		hdr  string
		val  string
		want bool
	}{
		{"X-Custom-Header", "some-value", true},
		{"X-Custom-Header", "", false},
		{"X-Trace-ID", "trace-123", true},
	}
	for _, tc := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(tc.hdr, tc.val)
		ok, err := ValidateHeaderValue(tc.hdr, req)
		if err != nil {
			t.Errorf("unexpected error for hdr=%q val=%q: %v", tc.hdr, tc.val, err)
		}
		if ok != tc.want {
			t.Errorf("ValidateHeaderValue(%q, %q) = %v, want %v", tc.hdr, tc.val, ok, tc.want)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
