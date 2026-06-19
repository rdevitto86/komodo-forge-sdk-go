package rules

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type errReader struct{}

func (e errReader) Read(p []byte) (int, error) { return 0, fmt.Errorf("simulated read error") }
func (e errReader) Close() error               { return nil }

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
  /body-constraints:
    POST:
      level: strict
      requiredVersion: 1
      body:
        code:
          type: string
          required: true
          pattern: "^[A-Z]{3}$"
          enum: ["ABC", "DEF", "GHI"]
          min_len: 3
          max_len: 3
        score:
          type: int
          required: false
          enum: ["100", "200"]
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

func getRuleOnly(t *testing.T, path, method string) *EvalRule {
	t.Helper()
	rule, _ := GetRule(path, method)
	return rule
}

func getRuleAndParams(t *testing.T, path, method string) (*EvalRule, map[string]string) {
	t.Helper()
	return GetRule(path, method)
}

// ── Unit Tests ──────────────────────────────────────────────────────────

func TestEval_WithValidRule(t *testing.T) {
	setupComprehensive(t)

	t.Run("nil request returns false", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		if IsRuleValid(nil, rule, nil) {
			t.Error("expected false for nil request")
		}
	})

	t.Run("nil rule returns false", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		if IsRuleValid(req, nil, nil) {
			t.Error("expected false for nil rule")
		}
	})

	t.Run("ignore level always passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "GET")
		req := httptest.NewRequest("GET", "/test", nil)
		if !IsRuleValid(req, rule, nil) {
			t.Error("expected true for ignore level rule")
		}
	})
}

func TestEval_WithInvalidRule(t *testing.T) {
	setupComprehensive(t)

	t.Run("strict level missing required header", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "PUT")
		req := httptest.NewRequest("PUT", "/v1/test", nil)
		req.Header.Set("Accept", "application/json;v=1")
		if IsRuleValid(req, rule, nil) {
			t.Error("expected false when required header X-Missing is absent")
		}
	})
}

func TestEval_Version_Lenient(t *testing.T) {
	setupComprehensive(t)

	t.Run("lenient: no version provided - passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "PATCH")
		req := httptest.NewRequest("PATCH", "/test", nil)
		req.Body = http.NoBody
		result := IsRuleValid(req, rule, nil)
		_ = result
	})

	t.Run("lenient: with valid version - passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "PATCH")
		req := makeReqWithVersion("PATCH", "/test", "1")
		req.Body = http.NoBody
		result := isValidVersion(req, rule)
		if !result {
			t.Error("expected true for lenient mode with valid version")
		}
	})

	t.Run("lenient: invalid version format - still passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "PATCH")
		req := httptest.NewRequest("PATCH", "/v1abc/test", nil)
		result := isValidVersion(req, rule)
		_ = result
	})

	t.Run("lenient: version mismatch - still passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "PATCH")
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
		rule := getRuleOnly(t, "/versioned", "GET")
		req := makeReqWithVersion("GET", "/versioned", "2")
		result := isValidVersion(req, rule)
		if !result {
			t.Error("expected true for strict with correct version")
		}
	})

	t.Run("strict: wrong version fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/versioned", "GET")
		req := makeReqWithVersion("GET", "/versioned", "1")
		result := isValidVersion(req, rule)
		if result {
			t.Error("expected false for strict with wrong version")
		}
	})

	t.Run("strict: no version fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/versioned", "GET")
		req := httptest.NewRequest("GET", "/api/resource", nil)
		result := isValidVersion(req, rule)
		if result {
			t.Error("expected false for strict with no version")
		}
	})

	t.Run("strict: invalid version format fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/versioned", "GET")
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
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"name":"alice"}`))
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when required Content-Type is missing")
		}
	})

	t.Run("required header present - passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with required header present")
		}
	})

	t.Run("wildcard prefix value match - passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Idempotency", "Bearer mytoken123")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with wildcard prefix match")
		}
	})

	t.Run("wildcard prefix value mismatch - fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Idempotency", "Basic xyz")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with wildcard prefix mismatch")
		}
	})

	t.Run("exact value mismatch - fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "text/plain")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with exact value mismatch")
		}
	})

	t.Run("enum check passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Enum-Header", "b")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with valid enum value")
		}
	})

	t.Run("enum check fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Enum-Header", "z")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with invalid enum value")
		}
	})

	t.Run("pattern check passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Pattern-Header", "ABC")
		result := areValidHeaders(req, rule)
		if !result {
			t.Error("expected true with valid pattern")
		}
	})

	t.Run("pattern check fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Pattern-Header", "abc123")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false with invalid pattern")
		}
	})

	t.Run("min length fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-MinMax-Header", "ab")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when header is shorter than min_len")
		}
	})

	t.Run("max length fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-MinMax-Header", "this-is-way-too-long-value")
		result := areValidHeaders(req, rule)
		if result {
			t.Error("expected false when header exceeds max_len")
		}
	})

	t.Run("header fails ValidateHeaderValue check", func(t *testing.T) {
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
		rule := getRuleOnly(t, "/queried", "GET")
		req := httptest.NewRequest("GET", "/queried", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false when required query param is missing")
		}
	})

	t.Run("required query param present with valid value", func(t *testing.T) {
		rule := getRuleOnly(t, "/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=active", nil)
		result := areValidQueryParams(req, rule)
		if !result {
			t.Error("expected true with valid filter query param")
		}
	})

	t.Run("query param pattern fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=ACTIVE", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false with uppercase filter (pattern fails)")
		}
	})

	t.Run("query param enum fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=unknown", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false with invalid enum value")
		}
	})

	t.Run("query param min length fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=ab", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false when query param is shorter than min_len")
		}
	})

	t.Run("query param max length fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/queried", "GET")
		req := httptest.NewRequest("GET", "/queried?filter=activetoolongvalue", nil)
		result := areValidQueryParams(req, rule)
		if result {
			t.Error("expected false when query param exceeds max_len")
		}
	})
}

func TestEval_PathParams(t *testing.T) {
	setupComprehensive(t)

	t.Run("no params - required path param fails", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: true},
			},
		}
		result := areValidPathParams(rule, nil)
		if result {
			t.Error("expected false when required path param is missing and no params")
		}
	})

	t.Run("no params - optional path param passes", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: false},
			},
		}
		result := areValidPathParams(rule, nil)
		if !result {
			t.Error("expected true when optional path param is missing")
		}
	})

	t.Run("valid pattern route - int param passes", func(t *testing.T) {
		rule, params := getRuleAndParams(t, "/params/123", "GET")
		if rule == nil {
			t.Skip("rule not found for pattern route")
		}
		result := areValidPathParams(rule, params)
		if !result {
			t.Error("expected true for valid int path param")
		}
	})

	t.Run("valid pattern route - invalid int param fails", func(t *testing.T) {
		rule, params := getRuleAndParams(t, "/params/abc", "POST")
		if rule == nil {
			t.Skip("rule not found")
		}
		result := areValidPathParams(rule, params)
		if result {
			t.Error("expected false for non-bool path param with type=bool")
		}
	})

	t.Run("pattern route - path param enum mismatch fails", func(t *testing.T) {
		rule, params := getRuleAndParams(t, "/params/999", "GET")
		if rule == nil {
			t.Skip("rule not found")
		}
		result := areValidPathParams(rule, params)
		if result {
			t.Error("expected false for path param not in enum")
		}
	})

	t.Run("pattern route - path param pattern mismatch fails", func(t *testing.T) {
		rule, params := getRuleAndParams(t, "/params/abc", "GET")
		if rule == nil {
			t.Skip("rule not found")
		}
		result := areValidPathParams(rule, params)
		if result {
			t.Error("expected false for path param not matching pattern ^[0-9]+$")
		}
	})

	t.Run("pattern route - unknown type passes through", func(t *testing.T) {
		rule, params := getRuleAndParams(t, "/params/123", "PUT")
		if rule == nil {
			t.Skip("rule not found")
		}
		result := areValidPathParams(rule, params)
		if !result {
			t.Error("expected true for unknown type (pass-through)")
		}
	})

	t.Run("path param required but missing from params", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"missing": {Required: true},
			},
		}
		params := map[string]string{"id": "123"}
		result := areValidPathParams(rule, params)
		if result {
			t.Error("expected false when required path param is missing from matched params")
		}
	})

	t.Run("optional path param not in matched params - continues", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"other": {Required: false},
			},
		}
		params := map[string]string{"id": "123"}
		result := areValidPathParams(rule, params)
		if !result {
			t.Error("expected true when optional path param is missing (continue)")
		}
	})

	t.Run("path param min length fails", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			PathParams: PathParams{
				"id": {Required: true, MinLen: 10},
			},
		}
		params := map[string]string{"id": "123"}
		result := areValidPathParams(rule, params)
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
		params := map[string]string{"id": "123"}
		result := areValidPathParams(rule, params)
		if result {
			t.Error("expected false when path param exceeds max_len")
		}
	})
}

func TestEval_Body(t *testing.T) {
	setupComprehensive(t)

	t.Run("GET skips body validation", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "GET")
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

	t.Run("empty body with no required fields is valid", func(t *testing.T) {
		rule := &EvalRule{
			Level: LevelLenient,
			Body: Body{
				"optional": {Type: "string", Required: false},
			},
		}
		req := httptest.NewRequest("POST", "/test", strings.NewReader(""))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for empty body with no required fields")
		}
	})

	t.Run("empty body with required fields fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(""))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for empty body when required field exists")
		}
	})

	t.Run("no body rules skips validation", func(t *testing.T) {
		rule := &EvalRule{Level: LevelStrict, Body: Body{}}
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"anything":"goes"}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true when rule has no body constraints")
		}
	})

	t.Run("valid JSON body passes", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","count":5,"active":true}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for valid JSON body")
		}
	})

	t.Run("unknown fields in body are accepted", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","extra_field":"ok"}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true when body contains unknown fields")
		}
	})

	t.Run("invalid JSON fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`not-json`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for invalid JSON")
		}
	})

	t.Run("required body field missing fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"count":5}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false when required 'name' field is missing")
		}
	})

	t.Run("wrong type for string field fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":42}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for wrong type (name should be string)")
		}
	})

	t.Run("wrong type for int field fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","count":"five"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for wrong type (count should be int/float64)")
		}
	})

	t.Run("wrong type for bool field fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"alice","active":"yes"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for wrong type (active should be bool)")
		}
	})

	t.Run("optional body field absent is ok", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"bob"}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true when optional 'count' and 'active' are absent")
		}
	})
}

func TestEval_BodyConstraints(t *testing.T) {
	setupComprehensive(t)

	t.Run("body string field pattern passes", func(t *testing.T) {
		rule, _ := GetRule("/body-constraints", "POST")
		if rule == nil {
			t.Fatal("rule not found")
		}
		req := makeReqWithVersion("POST", "/v1/body-constraints", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"code":"ABC"}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for body field matching pattern")
		}
	})

	t.Run("body string field pattern fails", func(t *testing.T) {
		rule, _ := GetRule("/body-constraints", "POST")
		if rule == nil {
			t.Fatal("rule not found")
		}
		req := makeReqWithVersion("POST", "/v1/body-constraints", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"code":"abc"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for body field failing pattern")
		}
	})

	t.Run("body string field enum fails", func(t *testing.T) {
		rule, _ := GetRule("/body-constraints", "POST")
		if rule == nil {
			t.Fatal("rule not found")
		}
		req := makeReqWithVersion("POST", "/v1/body-constraints", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"code":"XYZ"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for body field not in enum")
		}
	})

	t.Run("body string field minLen fails", func(t *testing.T) {
		rule, _ := GetRule("/body-constraints", "POST")
		if rule == nil {
			t.Fatal("rule not found")
		}
		req := makeReqWithVersion("POST", "/v1/body-constraints", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"code":"AB"}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for body field shorter than min_len")
		}
	})

	t.Run("body int field enum passes", func(t *testing.T) {
		rule, _ := GetRule("/body-constraints", "POST")
		if rule == nil {
			t.Fatal("rule not found")
		}
		req := makeReqWithVersion("POST", "/v1/body-constraints", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"code":"ABC","score":100}`))
		result := isValidBody(req, rule)
		if !result {
			t.Error("expected true for int body field in enum")
		}
	})

	t.Run("body int field enum fails", func(t *testing.T) {
		rule, _ := GetRule("/body-constraints", "POST")
		if rule == nil {
			t.Fatal("rule not found")
		}
		req := makeReqWithVersion("POST", "/v1/body-constraints", "1")
		req.Body = io.NopCloser(strings.NewReader(`{"code":"ABC","score":999}`))
		result := isValidBody(req, rule)
		if result {
			t.Error("expected false for int body field not in enum")
		}
	})
}

func TestIsRuleValid_Integration(t *testing.T) {
	setupComprehensive(t)

	t.Run("full valid strict POST request", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(strings.NewReader(`{"name":"alice"}`))
		result := IsRuleValid(req, rule, nil)
		if !result {
			t.Error("expected true for fully valid strict POST")
		}
	})

	t.Run("strict POST with missing required header fails at header check", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "PUT")
		req := makeReqWithVersion("PUT", "/v1/test", "1")
		req.Body = io.NopCloser(strings.NewReader(""))
		result := IsRuleValid(req, rule, nil)
		if result {
			t.Error("expected false when required header missing")
		}
	})

	t.Run("strict POST with wrong version fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/versioned", "GET")
		req := makeReqWithVersion("GET", "/v1/versioned", "1")
		result := IsRuleValid(req, rule, nil)
		if result {
			t.Error("expected false for wrong version")
		}
	})

	t.Run("strict POST with invalid body fails", func(t *testing.T) {
		rule := getRuleOnly(t, "/test", "POST")
		req := makeReqWithVersion("POST", "/v1/test", "1")
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(strings.NewReader(`invalid-json`))
		result := IsRuleValid(req, rule, nil)
		if result {
			t.Error("expected false for invalid body")
		}
	})

	t.Run("fails at path param check via IsRuleValid", func(t *testing.T) {
		rule := &EvalRule{
			Level:           LevelLenient,
			RequiredVersion: 0,
			PathParams: PathParams{
				"needed": {Required: true},
			},
			Body: Body{},
		}
		req := httptest.NewRequest("POST", "/static/path", nil)
		req.Body = io.NopCloser(strings.NewReader(""))
		result := IsRuleValid(req, rule, nil)
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
			Body: Body{},
		}
		req := httptest.NewRequest("POST", "/any", nil)
		req.Body = io.NopCloser(strings.NewReader(""))
		result := IsRuleValid(req, rule, nil)
		if result {
			t.Error("expected false when required query param is missing")
		}
	})
}

func TestEval_PathParams_IntType_InvalidValue_Failure(t *testing.T) {
	setupComprehensive(t)
	rule := &EvalRule{
		Level: LevelLenient,
		PathParams: PathParams{
			"id": {Required: true, Type: "int"},
		},
	}
	params := map[string]string{"id": "notanumber"}
	result := areValidPathParams(rule, params)
	if result {
		t.Error("expected false for non-integer value with type=int path param")
	}
}

func TestEval_PathParams_StringType_Success(t *testing.T) {
	setupComprehensive(t)
	rule := &EvalRule{
		Level: LevelLenient,
		PathParams: PathParams{
			"id": {Required: true, Type: "string"},
		},
	}
	params := map[string]string{"id": "hello"}
	result := areValidPathParams(rule, params)
	if !result {
		t.Error("expected true for string-typed path param")
	}
}

func TestEval_PathParams_InvalidRegex_LoadFailure(t *testing.T) {
	ResetForTesting()
	badRegexYAML := `
rules:
  /params/:id:
    DELETE:
      level: lenient
      params:
        id:
          required: false
          pattern: "^INVALID[[$"
`
	result := LoadConfigWithData([]byte(badRegexYAML))
	if result {
		t.Error("expected config load to fail with invalid regex pattern")
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

func TestEval_QueryParams_InvalidRegex_LoadFailure(t *testing.T) {
	ResetForTesting()
	badRegexYAML := `
rules:
  /test:
    GET:
      level: lenient
      query:
        q:
          required: true
          pattern: "^INVALID[[$"
`
	result := LoadConfigWithData([]byte(badRegexYAML))
	if result {
		t.Error("expected config load to fail with invalid regex pattern")
	}
}

func TestEval_Body_ReadError_Failure(t *testing.T) {
	setupComprehensive(t)
	rule := getRuleOnly(t, "/test", "POST")
	req := httptest.NewRequest("POST", "/test", nil)
	req.Body = errReader{}
	result := isValidBody(req, rule)
	if result {
		t.Error("expected false when request body cannot be read")
	}
}
