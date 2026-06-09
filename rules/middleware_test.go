package rules

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var testConfig = []byte(`
rules:
  /items:
    GET:
      toggle: true
      level: strict
      requiredVersion: 1
      headers:
        X-Required-Header:
          required: true
    POST:
      toggle: true
      level: lenient
`)

func TestMain(m *testing.M) {
	LoadConfigWithData(testConfig)
	os.Exit(m.Run())
}

// loadItemsConfig re-establishes the /items test rules. The eval tests reset global rule
// state via ResetForTesting, so each middleware test must reload its own config to remain
// independent of execution order.
func loadItemsConfig(t *testing.T) {
	t.Helper()
	ResetForTesting()
	LoadConfigWithData(testConfig)
	if !IsConfigLoaded() {
		t.Fatal("failed to load middleware test rules")
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRuleValidationMiddleware_ValidRequest(t *testing.T) {
	loadItemsConfig(t)
	// GET /v1/items with required header → passes
	req := httptest.NewRequest(http.MethodGet, "/v1/items", nil)
	req.Header.Set("X-Required-Header", "present")
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	RuleValidationMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called for valid request")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRuleValidationMiddleware_MissingRequiredHeader(t *testing.T) {
	loadItemsConfig(t)
	// GET /v1/items without X-Required-Header → 400
	req := httptest.NewRequest(http.MethodGet, "/v1/items", nil)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	RuleValidationMiddleware(next).ServeHTTP(rec, req)

	if called {
		t.Error("expected next NOT to be called when header is missing")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRuleValidationMiddleware_NoMatchingRule(t *testing.T) {
	loadItemsConfig(t)
	// /v1/unknown has no rule configured → 400
	req := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	RuleValidationMiddleware(next).ServeHTTP(rec, req)

	if called {
		t.Error("expected next NOT to be called for unmatched route")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRuleValidationMiddleware_LenientLevelPassesWithoutVersion(t *testing.T) {
	loadItemsConfig(t)
	// POST /items (lenient level, no requiredVersion) — no version prefix needed
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	rec := httptest.NewRecorder()
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	RuleValidationMiddleware(next).ServeHTTP(rec, req)

	if !called {
		t.Error("expected next to be called for lenient rule without version")
	}
}

func TestRuleValidationMiddleware_LoadConfigReturnsFalse(t *testing.T) {
	// Reset the rules package state so LoadConfig() will return false (no path,
	// no EVAL_RULES_PATH env var, no embedded data).
	ResetForTesting()
	defer func() {
		// Restore the test config for subsequent tests.
		ResetForTesting()
		LoadConfigWithData(testConfig)
	}()

	req := httptest.NewRequest(http.MethodGet, "/v1/items", nil)
	req.Header.Set("X-Required-Header", "present")
	rec := httptest.NewRecorder()

	// Building the middleware triggers LoadConfig() → false (no config).
	handler := RuleValidationMiddleware(okHandler())
	handler.ServeHTTP(rec, req)

	// With no config loaded, GetRule returns nil → 400.
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when config not loaded, got %d", rec.Code)
	}
}
