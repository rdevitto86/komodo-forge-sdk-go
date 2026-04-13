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

func TestGetClientKey_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	got := GetClientKey(req)
	if got != "10.0.0.1" {
		t.Errorf("GetClientKey = %q, want 10.0.0.1", got)
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

func TestGetClientType_BearerWithGrantType(t *testing.T) {
	claims := map[string]interface{}{
		"sub":        "user123",
		"grant_type": "client_credentials",
	}
	token := makeTestJWT(claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := GetClientType(req)
	if got != "api" {
		t.Errorf("GetClientType = %q, want api (grant_type=client_credentials)", got)
	}
}

func TestGetClientType_BearerWithApiScope(t *testing.T) {
	claims := map[string]interface{}{
		"scope": "api:read api:write",
	}
	token := makeTestJWT(claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := GetClientType(req)
	if got != "api" {
		t.Errorf("GetClientType = %q, want api (scope contains api:)", got)
	}
}

func TestGetClientType_BearerWithServiceScope(t *testing.T) {
	claims := map[string]interface{}{
		"scope": "service:internal",
	}
	token := makeTestJWT(claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := GetClientType(req)
	if got != "api" {
		t.Errorf("GetClientType = %q, want api (scope contains service:)", got)
	}
}

func TestGetClientType_BearerWithClientType(t *testing.T) {
	claims := map[string]interface{}{
		"client_type": "browser",
	}
	token := makeTestJWT(claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := GetClientType(req)
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser (client_type=browser)", got)
	}
}

func TestGetClientType_BearerWithAPIClientType(t *testing.T) {
	claims := map[string]interface{}{
		"client_type": "api",
	}
	token := makeTestJWT(claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := GetClientType(req)
	if got != "api" {
		t.Errorf("GetClientType = %q, want api (client_type=api)", got)
	}
}

func TestGetClientType_BearerWithUnknownClientType(t *testing.T) {
	// client_type present but not "api" or "browser"
	claims := map[string]interface{}{
		"client_type": "other",
	}
	token := makeTestJWT(claims)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := GetClientType(req)
	// Falls through to browser default
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser for unknown client_type", got)
	}
}

func TestGetClientType_NoSpecialHeaders_DefaultBrowser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got := GetClientType(req)
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser (default)", got)
	}
}

func TestGetClientType_InvalidBearerFormat(t *testing.T) {
	// Not a valid 3-part JWT
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt.token.here")
	got := GetClientType(req)
	// Falls through to browser
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser for non-3-part JWT", got)
	}
}

func TestGetClientType_InvalidBase64Payload(t *testing.T) {
	// Valid 3-part format but invalid base64
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer header.!!!invalid!!.sig")
	got := GetClientType(req)
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser for invalid base64", got)
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

// makeTestJWT creates an unsigned test JWT (header.payload.sig).
func makeTestJWT(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + encodedPayload + ".sig"
}

// TestGetAPIRoute_QueryInPath covers the branch where URL.Path contains a literal '?'
// (e.g., a manually constructed URL rather than one parsed by net/url).
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

// TestGetQueryParams_InvalidRawQuery covers the url.ParseQuery error path.
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
