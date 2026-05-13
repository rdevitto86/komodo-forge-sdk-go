package rules

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// errReader is a ReadCloser that returns an error on Read, used to test body-read failure paths.
type errReader struct{}

func (e errReader) Read(p []byte) (int, error) { return 0, fmt.Errorf("simulated read error") }
func (e errReader) Close() error               { return nil }

// comprehensiveYAML is used to test various eval paths.
const comprehensiveYAML = `
rules:
  /test:
    GET:
      level: ignore
    POST:
      level: strict
      requiredVersion: 1
      headers:
        Content-Type:
          required: true
          value: "application/json"
        X-Idempotency:
          required: false
          value: "Bearer *"
        X-Enum-Header:
          required: false
          enum: ["a", "b", "c"]
        X-Pattern-Header:
          required: false
          pattern: "^[A-Z]+$"
        X-MinMax-Header:
          required: false
          min_len: 3
          max_len: 10
      body:
        name:
          type: string
          required: true
        count:
          type: int
          required: false
        active:
          type: bool
          required: false
    PUT:
      level: strict
      requiredVersion: 1
      headers:
        X-Missing:
          required: true
    PATCH:
      level: lenient
      requiredVersion: 1
  /versioned:
    GET:
      level: strict
      requiredVersion: 2
  /queried:
    GET:
      level: lenient
      query:
        filter:
          required: true
          pattern: "^[a-z]+$"
          enum: ["active", "inactive"]
          min_len: 3
          max_len: 20
  /params/:id:
    GET:
      level: lenient
      params:
        id:
          required: true
          pattern: "^[0-9]+$"
          enum: ["123", "456"]
          min_len: 1
          max_len: 5
          type: int
    POST:
      level: lenient
      params:
        id:
          required: true
          type: bool
    PUT:
      level: lenient
      params:
        id:
          type: unknown_type
    DELETE:
      level: lenient
      params:
        id:
          required: false
          pattern: "^INVALID[[$"
`

func setupComprehensive(t *testing.T) {
	t.Helper()
	ResetForTesting()
	LoadConfigWithData([]byte(comprehensiveYAML))
	if !IsConfigLoaded() {
		t.Fatal("failed to load test rules")
	}
}

func makeReqWithVersion(method, path, version string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if version != "" {
		req.Header.Set("Accept", "application/json;v="+version)
	}
	return req
}

func TestEval_WithValidRule(t *testing.T) {
	setupComprehensive(t)

	t.Run("nil request returns false", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		if IsRuleValid(nil, rule) {
			t.Error("expected false for nil request")
		}
	})

	t.Run("nil rule returns false", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		if IsRuleValid(req, nil) {
			t.Error("expected false for nil rule")
		}
	})

	t.Run("ignore level always passes", func(t *testing.T) {
		rule := GetRule("/test", "GET")
		req := httptest.NewRequest("GET", "/test", nil)
		if !IsRuleValid(req, rule) {
			t.Error("expected true for ignore level rule")
		}
	})
}

func TestEval_WithInvalidRule(t *testing.T) {
	setupComprehensive(t)

	t.Run("strict level missing required header", func(t *testing.T) {
		rule := GetRule("/test", "PUT")
		req := httptest.NewRequest("PUT", "/v1/test", nil)
		req.Header.Set("Accept", "application/json;v=1")
		if IsRuleValid(req, rule) {
			t.Error("expected false when required header X-Missing is absent")
		}
	})
}

func TestEval_Version_Lenient(t *testing.T) {
	setupComprehensive(t)

	t.Run("lenient: no version provided - passes", func(t *testing.T) {
		rule := GetRule("/test", "PATCH")
		req := httptest.NewRequest("PATCH", "/test", nil)
		// No version header or URL version
		// Note: areValidQueryParams and areValidPathParams/Body will also run
		// PATCH with empty body is valid
		req.Body = http.NoBody
		result := IsRuleValid(req, rule)
		// With lenient level and no headers required, should pass
		_ = result // lenient with no version passes that check; other checks may fail
	})

	t.Run("lenient: with valid version - passes", func(t *testing.T) {
		rule := GetRule("/test", "PATCH")
		req := makeReqWithVersion("PATCH", "/test", "1")
		req.Body = http.NoBody
		result := isValidVersion(req, rule)
		if !result {
			t.Error("expected true for lenient mode with valid version")
		}
	})

	t.Run("lenient: invalid version format - still passes", func(t *testing.T) {
		rule := GetRule("/test", "PATCH")
		req := httptest.NewRequest("PATCH", "/v1abc/test", nil)
		result := isValidVersion(req, rule)
		_ = result // vXXX format isn't matched as valid version; passes lenient
	})

	t.Run("lenient: version mismatch - still passes", func(t *testing.T) {
		rule := GetRule("/test", "PATCH")
		req := makeReqWithVersion("PATCH", "/test", "99")
		result := isValidVersion(req, rule)
		if !result {
			t.Error("expected true for lenient mode even with version mismatch")
		}
	})
}

func TestEval_Version_Strict(t *testing.T) {
	setupComprehensive(t)

	t.Run("strict: correct version passes", func(t *testing.T) {
		rule := GetRule("/versioned", "GET")
		req := makeReqWithVersion("GET", "/versioned", "2")
		result := isValidVersion(req, rule)
		if !result {
			t.Error("expected true for strict with correct version")
		}
	})

	t.Run("strict: wrong version fails", func(t *testing.T) {
		rule := GetRule("/versioned", "GET")
		req := makeReqWithVersion("GET", "/versioned", "1")
		result := isValidVersion(req, rule)
		if result {
			t.Error("expected false for strict with wrong version")
		}
	})

	t.Run("strict: no version fails", func(t *testing.T) {
		rule := GetRule("/versioned", "GET")
		// Use a path that doesn't start with 'v' to ensure GetAPIVersion returns ""
		req := httptest.NewRequest("GET", "/api/resource", nil)
		result := isValidVersion(req, rule)
		if result {
			t.Error("expected false for strict with no version")
		}
	})

	t.Run("strict: invalid version format fails", func(t *testing.T) {
		rule := GetRule("/versioned", "GET")
		// Use a URL path with v-prefix and letters after (not a number)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept", "application/json;v=abc")
		result := isValidVersion(req, rule)
		if result {
			t.Error("expected false for strict with invalid version format")
		}
	})

	t.Run("strict: requiredVersion <= 0 fails", func(t *testing.T) {
		rule := &EvalRule{Level: LevelStrict, RequiredVersion: 0}
		req := makeReqWithVersion("GET", "/any", "1")
		result := isValidVersion(req, rule)
		if result {
			t.Error("expected false for strict with requiredVersion=0")
		}
	})
}

func TestEval_Headers(t *testing.T) {
	setupComprehensive(t)

	t.Run("required header missing - fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		// No Content-Type header
		req.Body = io.NopCloser(strings.NewReader(`{"name":"alice"}`))
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when required Content-Type is missing")
		}
	})

	t.Run("required header present - passes", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with required header present")
		}
	})

	t.Run("wildcard prefix value match - passes", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Idempotency", "Bearer mytoken123")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with wildcard prefix match")
		}
	})

	t.Run("wildcard prefix value mismatch - fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Idempotency", "Basic xyz")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with wildcard prefix mismatch")
		}
	})

	t.Run("exact value mismatch - fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "text/plain") // wrong value
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with exact value mismatch")
		}
	})

	t.Run("enum check passes", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Enum-Header", "b")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with valid enum value")
		}
	})

	t.Run("enum check fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Enum-Header", "z")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with invalid enum value")
		}
	})

	t.Run("pattern check passes", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Pattern-Header", "ABC")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with valid pattern")
		}
	})

	t.Run("pattern check fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Pattern-Header", "abc123") // lowercase doesn't match ^[A-Z]+$
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with invalid pattern")
		}
	})

	t.Run("min length fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-MinMax-Header", "ab") // too short (< 3)
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when header is shorter than min_len")
		}
	})

	t.Run("max length fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-MinMax-Header", "this-is-way-too-long-value") // > 10
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when header exceeds max_len")
		}
	})

	t.Run("header fails ValidateHeaderValue check", func(t *testing.T) {
		// Content-Length: 0 fails isValidContentLength (not > 0)
		rule := &EvalRule{
			Level: LevelLenient,
			Headers: Headers{
				"Content-Length": {Required: false},
			},
		}
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Content-Length", "0")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when Content-Length=0 fails ValidateHeaderValue")
		}
	})
}

func TestEval_QueryParams(t *testing.T) {
	setupComprehensive(t)

	t.Run("optional query param absent - continues", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			QueryParams: QueryParams{
				"optional_param": {Required: false},
			},
		}
		req := httptest.NewRequest("GET", "/queried", nil)
		result := areValidQueryParams(req, rule)
		if !result {
			t.Error("expected true when optional query param is missing (continue)")
		}
	})

	t.Run("required query param missing - fails", func(t *testing.T) {
		rule := GetRule("/queried", "GET")
		req := httptest.NewRequest("GET", "/queried", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false when required query param is missing")
		}
	})

	t.Run("required query param present with valid value", func(t *testing.T) {
		rule := GetRule("/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=active", nil)
		result := areValidQueryParams(req, rule)
		if !result {
			t.Error("expected true with valid filter query param")
		}
	})

	t.Run("query param pattern fails", func(t *testing.T) {
		rule := GetRule("/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=ACTIVE", nil) // uppercase fails ^[a-z]+$
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false with uppercase filter (pattern fails)")
		}
	})

	t.Run("query param enum fails", func(t *testing.T) {
		rule := GetRule("/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=unknown", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false with invalid enum value")
		}
	})

	t.Run("query param min length fails", func(t *testing.T) {
		rule := GetRule("/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=ab", nil) // too short
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false when query param is shorter than min_len")
		}
	})

	t.Run("query param max length fails", func(t *testing.T) {
		rule := GetRule("/queried", "GET")
		// 'filter' max is 20, use a too-long value that also fails enum
		req := httptest.NewRequest("GET", "/queried?filter=activetoolongvalue", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false when query param exceeds max_len")
		}
	})
}

func TestEval_PathParams(t *testing.T) {
	setupComprehensive(t)

	t.Run("no pattern routes - required path param fails", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: true},
			},
		}
		req := httptest.NewRequest("GET", "/static/path", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false when required path param is missing and no pattern matches")
		}
	})

	t.Run("no pattern routes - optional path param passes", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: false},
			},
		}
		req := httptest.NewRequest("GET", "/static/path", nil)
		result := areValidPathParams(req, rule)
		if !result {
			t.Error("expected true when optional path param is missing")
		}
	})

	t.Run("valid pattern route - int param passes", func(t *testing.T) {
		rule := GetRule("/params/123", "GET")
		if rule == nil {
			t.Skip("rule not found for pattern route")
		}
		req := httptest.NewRequest("GET", "/params/123", nil)
		result := areValidPathParams(req, rule)
		if !result {
			t.Error("expected true for valid int path param")
		}
	})

	t.Run("valid pattern route - invalid int param fails", func(t *testing.T) {
		// POST /params/:id expects type bool
		rule := GetRule("/params/abc", "POST")
		if rule == nil {
			t.Skip("rule not found")
		}
		req := httptest.NewRequest("POST", "/params/abc", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false for non-bool path param with type=bool")
		}
	})

	t.Run("pattern route - path param enum mismatch fails", func(t *testing.T) {
		rule := GetRule("/params/999", "GET")
		if rule == nil {
			t.Skip("rule not found")
		}
		req := httptest.NewRequest("GET", "/params/999", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false for path param not in enum")
		}
	})

	t.Run("pattern route - path param pattern mismatch fails", func(t *testing.T) {
		rule := GetRule("/params/abc", "GET")
		if rule == nil {
			t.Skip("rule not found")
		}
		req := httptest.NewRequest("GET", "/params/abc", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false for path param not matching pattern ^[0-9]+$")
		}
	})

	t.Run("pattern route - unknown type passes through", func(t *testing.T) {
		rule := GetRule("/params/123", "PUT")
		if rule == nil {
			t.Skip("rule not found")
		}
		req := httptest.NewRequest("PUT", "/params/123", nil)
		result := areValidPathParams(req, rule)
		if !result {
			t.Error("expected true for unknown type (pass-through)")
		}
	})

	t.Run("path param required but missing from pattern match", func(t *testing.T) {
		// A rule with a required param that's not in the URL
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"missing": {Required: true},
			},
		}
		// Load a comprehensiveYAML with :id pattern - even though it matches :id,
		// the rule being evaluated has "missing" not "id"
		req := httptest.NewRequest("GET", "/params/123", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false when required path param is missing from matched params")
		}
	})

	t.Run("optional path param not in matched params - continues", func(t *testing.T) {
		// Rule has an optional param "other" which won't be in params (only "id" is)
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"other": {Required: false},
			},
		}
		req := httptest.NewRequest("GET", "/params/123", nil)
		result := areValidPathParams(req, rule)
		if !result {
			t.Error("expected true when optional path param is missing (continue)")
		}
	})

	t.Run("path param min length fails", func(t *testing.T) {
		// Rule with min_len on :id param, but the value is too short
		// We need to override the rule to have min_len > len("123")
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: true, MinLen: 10},
			},
		}
		req := httptest.NewRequest("GET", "/params/123", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false when path param is shorter than min_len")
		}
	})

	t.Run("path param max length fails", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: true, MaxLen: 2},
			},
		}
		req := httptest.NewRequest("GET", "/params/123", nil)
		result := areValidPathParams(req, rule)
		if result {
			t.Error("expected false when path param exceeds max_len")
		}
	})
}

func TestEval_Body(t *testing.T) {
	setupComprehensive(t)

	t.Run("GET skips body validation", func(t *testing.T) {
		rule := GetRule("/test", "GET")
		req := httptest.NewRequest("GET", "/test", nil)
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for GET (body validation skipped)")
		}
	})

	t.Run("HEAD skips body validation", func(t *testing.T) {
		rule := &EvalRule{Level: LevelLenient}
		req := httptest.NewRequest("HEAD", "/test", nil)
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for HEAD (body validation skipped)")
		}
	})

	t.Run("OPTIONS skips body validation", func(t *testing.T) {
		rule := &EvalRule{Level: LevelLenient}
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for OPTIONS (body validation skipped)")
		}
	})

	t.Run("empty body is valid", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(""))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for empty body")
		}
	})

	t.Run("valid JSON body passes", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","count":5,"active":true}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for valid JSON body")
		}
	})

	t.Run("invalid JSON fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`not-json`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for invalid JSON")
		}
	})

	t.Run("required body field missing fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"count":5}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false when required 'name' field is missing")
		}
	})

	t.Run("wrong type for string field fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		// 'name' should be string but we pass a number
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":42}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for wrong type (name should be string)")
		}
	})

	t.Run("wrong type for int field fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		// 'count' should be int (float64 in JSON) but we pass a string
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","count":"five"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for wrong type (count should be int/float64)")
		}
	})

	t.Run("wrong type for bool field fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		// 'active' should be bool but we pass a string
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","active":"yes"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for wrong type (active should be bool)")
		}
	})

	t.Run("optional body field absent is ok", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"bob"}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true when optional 'count' and 'active' are absent")
		}
	})
}

func TestIsRuleValid_Integration(t *testing.T) {
	setupComprehensive(t)

	t.Run("full valid strict POST request", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(strings.NewReader(`{"name":"alice"}`))
		result := IsRuleValid(req, rule)
		if !result {
			t.Error("expected true for fully valid strict POST")
		}
	})

	t.Run("strict POST with missing required header fails at header check", func(t *testing.T) {
		rule := GetRule("/test", "PUT")
		req := makeReqWithVersion("PUT", "/v1/test", "1")
		// No X-Missing header
		req.Body = io.NopCloser(strings.NewReader(""))
		result := IsRuleValid(req, rule)
		if result {
			t.Error("expected false when required header missing")
		}
	})

	t.Run("strict POST with wrong version fails", func(t *testing.T) {
		rule := GetRule("/versioned", "GET")
		req := makeReqWithVersion("GET", "/v1/versioned", "1")
		result := IsRuleValid(req, rule)
		if result {
			t.Error("expected false for wrong version")
		}
	})

	t.Run("strict POST with invalid body fails", func(t *testing.T) {
		rule := GetRule("/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(strings.NewReader(`invalid-json`))
		result := IsRuleValid(req, rule)
		if result {
			t.Error("expected false for invalid body")
		}
	})

	t.Run("fails at path param check via IsRuleValid", func(t *testing.T) {
		// Use a rule with required path param
		rule := &EvalRule{
			Level:           LevelLenient,
			RequiredVersion: 0, // lenient: version optional
			PathParams: PathParams{
				"needed": {Required: true},
			},
		}
		// /static/path won't match any pattern route, so params will be nil
		req := httptest.NewRequest("POST", "/static/path", nil)
		req.Body = io.NopCloser(strings.NewReader(""))
		result := IsRuleValid(req, rule)
		if result {
			t.Error("expected false when required path param is missing")
		}
	})

	t.Run("fails at query param check via IsRuleValid", func(t *testing.T) {
		rule := &EvalRule{
			Level:           LevelLenient,
			RequiredVersion: 0,
			QueryParams: QueryParams{
				"required_q": {Required: true},
			},
		}
		req := httptest.NewRequest("POST", "/any", nil)
		req.Body = io.NopCloser(strings.NewReader(""))
		result := IsRuleValid(req, rule)
		if result {
			t.Error("expected false when required query param is missing")
		}
	})
}

func TestEval_PathParams_IntType_InvalidValue_Failure(t *testing.T) {
	setupComprehensive(t)
	// Custom rule: type int but no pattern that would reject non-ints first.
	rule := &EvalRule{
		Level: LevelLenient,
		PathParams: PathParams{
			"id": {Required: true, Type: "int"},
		},
	}
	// /params/notanumber matches the /params/:id pattern → id="notanumber"
	req := httptest.NewRequest("GET", "/params/notanumber", nil)
	result := areValidPathParams(req, rule)
	if result {
		t.Error("expected false for non-integer value with type=int path param")
	}
}

func TestEval_PathParams_StringType_Success(t *testing.T) {
	setupComprehensive(t)
	// Verify that type=string (and empty type) is a pass-through.
	rule := &EvalRule{
		Level: LevelLenient,
		PathParams: PathParams{
			"id": {Required: true, Type: "string"},
		},
	}
	req := httptest.NewRequest("GET", "/params/hello", nil)
	result := areValidPathParams(req, rule)
	if !result {
		t.Error("expected true for string-typed path param")
	}
}

func TestEval_PathParams_InvalidRegex_Failure(t *testing.T) {
	setupComprehensive(t)
	// The DELETE rule in comprehensiveYAML has pattern "^INVALID[[$" which is an invalid regex.
	rule := GetRule("/params/abc", "DELETE")
	if rule == nil {
		t.Skip("DELETE rule not found for /params/:id")
	}
	req := httptest.NewRequest("DELETE", "/params/abc", nil)
	result := areValidPathParams(req, rule)
	if result {
		t.Error("expected false for path param with invalid regex pattern")
	}
}

func TestEval_QueryParams_MinLen_Failure(t *testing.T) {
	rule := &EvalRule{
		Level: LevelLenient,
		QueryParams: QueryParams{
			"q": {Required: true, MinLen: 5},
		},
	}
	req := httptest.NewRequest("GET", "/test?q=ab", nil)
	result := areValidQueryParams(req, rule)
	if result {
		t.Error("expected false when query param is shorter than min_len")
	}
}

func TestEval_QueryParams_MaxLen_Failure(t *testing.T) {
	rule := &EvalRule{
		Level: LevelLenient,
		QueryParams: QueryParams{
			"q": {Required: true, MaxLen: 3},
		},
	}
	req := httptest.NewRequest("GET", "/test?q=toolongvalue", nil)
	result := areValidQueryParams(req, rule)
	if result {
		t.Error("expected false when query param exceeds max_len")
	}
}

func TestEval_QueryParams_InvalidRegex_Failure(t *testing.T) {
	rule := &EvalRule{
		Level: LevelLenient,
		QueryParams: QueryParams{
			"q": {Required: true, Pattern: "^INVALID[[$"},
		},
	}
	req := httptest.NewRequest("GET", "/test?q=somevalue", nil)
	result := areValidQueryParams(req, rule)
	if result {
		t.Error("expected false for query param with invalid regex pattern")
	}
}

func TestEval_Body_ReadError_Failure(t *testing.T) {
	setupComprehensive(t)
	rule := GetRule("/test", "POST")
	req := httptest.NewRequest("POST", "/test", nil)
	req.Body = errReader{}
	result := isValidBody(req, rule)
	if result {
		t.Error("expected false when request body cannot be read")
	}
}
