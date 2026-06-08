package request

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetAPIVersion_Nil(t *testing.T) {
	got := GetAPIVersion(nil)
	if got != "" {
		t.Errorf("GetAPIVersion(nil) = %q, want empty", got)
	}
}

func TestGetAPIVersion_AcceptHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json;v=1")
	got := GetAPIVersion(req)
	if got != "/v1" {
		t.Errorf("GetAPIVersion = %q, want /v1", got)
	}
}

func TestGetAPIVersion_AcceptHeaderVersionParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", "application/json; version=2")
	got := GetAPIVersion(req)
	if got != "/v2" {
		t.Errorf("GetAPIVersion = %q, want /v2", got)
	}
}

func TestGetAPIVersion_ContentTypeHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Content-Type", "application/json;v=3")
	got := GetAPIVersion(req)
	if got != "/v3" {
		t.Errorf("GetAPIVersion = %q, want /v3", got)
	}
}

func TestGetAPIVersion_URLPath(t *testing.T) {
	req := httptest.NewRequest("GET", "/v2/users", nil)
	got := GetAPIVersion(req)
	if got != "/v2" {
		t.Errorf("GetAPIVersion = %q, want /v2", got)
	}
}

func TestGetAPIVersion_NoVersion(t *testing.T) {
	req := httptest.NewRequest("GET", "/users", nil)
	got := GetAPIVersion(req)
	if got != "" {
		t.Errorf("GetAPIVersion = %q, want empty", got)
	}
}

func TestGetAPIVersion_AcceptNoParts(t *testing.T) {
	// Accept with no semicolon, no version in Content-Type, no version in URL
	req := httptest.NewRequest("GET", "/api/resource", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	got := GetAPIVersion(req)
	if got != "" {
		t.Errorf("GetAPIVersion = %q, want empty for unversioned request", got)
	}
}

func TestGetAPIVersion_AcceptWithUnknownParam(t *testing.T) {
	// Has semicolon but no v= or version= param → falls through to URL path
	req := httptest.NewRequest("GET", "/api/resource", nil)
	req.Header.Set("Accept", "application/json; charset=utf-8")
	got := GetAPIVersion(req)
	if got != "" {
		t.Errorf("GetAPIVersion = %q, want empty", got)
	}
}

func TestGetAPIRoute_Versioned(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/users/123", nil)
	got := GetAPIRoute(req)
	if got != "/users/123" {
		t.Errorf("GetAPIRoute = %q, want /users/123", got)
	}
}

func TestGetAPIRoute_Unversioned(t *testing.T) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	got := GetAPIRoute(req)
	if got != "/users/123" {
		t.Errorf("GetAPIRoute = %q, want /users/123", got)
	}
}

func TestGetAPIRoute_Nil(t *testing.T) {
	got := GetAPIRoute(nil)
	if got != "" {
		t.Errorf("GetAPIRoute(nil) = %q, want empty", got)
	}
}

func TestGetAPIRoute_WithQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	got := GetAPIRoute(req)
	if got != "/search" {
		t.Errorf("GetAPIRoute = %q, want /search", got)
	}
}

func TestGetAPIRoute_RootVersion(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1", nil)
	got := GetAPIRoute(req)
	// After stripping version, path becomes "/"
	if got != "/" {
		t.Errorf("GetAPIRoute = %q, want /", got)
	}
}

func TestGetAPIRoute_DoubleSlashCollapse(t *testing.T) {
	// /v1// → pathSegments=["",""] → route="//" → normalized to "/"
	req := httptest.NewRequest("GET", "/v1//", nil)
	got := GetAPIRoute(req)
	if got != "/" {
		t.Errorf("GetAPIRoute(/v1//) = %q, want /", got)
	}
}

func TestGetQueryParams_Multiple(t *testing.T) {
	req := httptest.NewRequest("GET", "/path?foo=bar&baz=qux&num=42", nil)
	params := GetQueryParams(req)
	if params["foo"] != "bar" {
		t.Errorf("foo = %q, want bar", params["foo"])
	}
	if params["baz"] != "qux" {
		t.Errorf("baz = %q, want qux", params["baz"])
	}
	if params["num"] != "42" {
		t.Errorf("num = %q, want 42", params["num"])
	}
}

func TestGetQueryParams_Empty(t *testing.T) {
	req := httptest.NewRequest("GET", "/path", nil)
	params := GetQueryParams(req)
	if len(params) != 0 {
		t.Errorf("expected empty params, got %v", params)
	}
}

func TestGetQueryParams_Nil(t *testing.T) {
	params := GetQueryParams(nil)
	if params == nil {
		t.Error("expected non-nil map for nil request")
	}
}

func TestGetQueryParams_NilURL(t *testing.T) {
	req := &http.Request{URL: nil}
	params := GetQueryParams(req)
	if params == nil {
		t.Error("expected non-nil map for nil URL")
	}
}

func TestGetClientKey_XForwardedFor_IgnoredByDefault(t *testing.T) {
	SetTrustedProxyDepth(0)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.5:9999"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	got := GetClientKey(req)
	if got != "203.0.113.5" {
		t.Errorf("GetClientKey = %q, want peer 203.0.113.5 (XFF must be ignored at depth 0)", got)
	}
}

func TestGetClientKey_XForwardedFor_TrustedDepth(t *testing.T) {
	// Two trusted proxies (proxyA, proxyB=peer) each append one X-Forwarded-For entry:
	// "client, proxyA". Counting 2 from the right lands on the originating client.
	SetTrustedProxyDepth(2)
	defer SetTrustedProxyDepth(0)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.3:9999"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	got := GetClientKey(req)
	if got != "10.0.0.1" {
		t.Errorf("GetClientKey = %q, want 10.0.0.1", got)
	}
}

func TestGetClientKey_XForwardedFor_IgnoresInjectedLeftHop(t *testing.T) {
	// An attacker prepends a spoofed hop; counting depth from the right still selects
	// the real client recorded by the outermost trusted proxy.
	SetTrustedProxyDepth(2)
	defer SetTrustedProxyDepth(0)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.3:9999"
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.1, 10.0.0.2")
	got := GetClientKey(req)
	if got != "10.0.0.1" {
		t.Errorf("GetClientKey = %q, want 10.0.0.1 (spoofed 1.2.3.4 must be ignored)", got)
	}
}

func TestGetClientKey_XForwardedFor_ShorterThanDepthFallsBack(t *testing.T) {
	// A header shorter than the configured depth indicates spoofing/misconfig; use the peer.
	SetTrustedProxyDepth(3)
	defer SetTrustedProxyDepth(0)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.5:9999"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	got := GetClientKey(req)
	if got != "203.0.113.5" {
		t.Errorf("GetClientKey = %q, want peer 203.0.113.5", got)
	}
}

func TestGetClientKey_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	got := GetClientKey(req)
	if got != "192.168.1.1" {
		t.Errorf("GetClientKey = %q, want 192.168.1.1", got)
	}
}

func TestGetClientKey_RemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.99"
	got := GetClientKey(req)
	if got != "192.168.1.99" {
		t.Errorf("GetClientKey = %q, want 192.168.1.99", got)
	}
}

func TestGetClientType_APIKey(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "some-api-key")
	got := GetClientType(req)
	if got != "api" {
		t.Errorf("GetClientType = %q, want api", got)
	}
}

func TestGetClientType_BearerClaimsIgnored(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+makeTestJWT(map[string]any{"client_type": "api", "scope": "api:write"}))
	if got := GetClientType(req); got != "browser" {
		t.Errorf("GetClientType = %q, want browser — Bearer claims must not influence client type", got)
	}
}

func TestGetClientType_NoSpecialHeaders_DefaultBrowser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := GetClientType(req)
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser (default)", got)
	}
}

func TestGetPathParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/users/123", nil)
	params := GetPathParams(req)
	// Current implementation always returns empty map
	if params == nil {
		t.Error("expected non-nil map")
	}
}

func TestIsValidAPIKey(t *testing.T) {
	// Current implementation always returns true
	if !IsValidAPIKey("any-key") {
		t.Error("IsValidAPIKey should return true (placeholder)")
	}
	if !IsValidAPIKey("") {
		t.Error("IsValidAPIKey('') should return true (placeholder)")
	}
}

// Creates an unsigned test JWT (header.payload.sig).
func makeTestJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encodedPayload + ".sig"
}

// Exercises a URL.Path containing a literal '?' (manually built, not net/url-parsed).
func TestGetAPIRoute_QueryInPath_Success(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/v1/search?embedded"},
	}
	got := GetAPIRoute(req)
	// The '?' and everything after it should be stripped from the path segment.
	if got != "/search" {
		t.Errorf("GetAPIRoute with '?' in path = %q, want /search", got)
	}
}

func TestGetQueryParams_InvalidRawQuery_Failure(t *testing.T) {
	req := &http.Request{URL: &url.URL{RawQuery: "%gg"}} // invalid percent-encoding
	params := GetQueryParams(req)
	// Should return an empty map (not nil) on parse error.
	if params == nil {
		t.Error("expected non-nil map on parse error")
	}
}

// Ensure httptest is used (needed to avoid import errors)
var _ = http.MethodGet
