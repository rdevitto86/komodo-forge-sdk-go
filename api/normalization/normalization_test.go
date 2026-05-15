package normalization

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runNormalization(req *http.Request) *http.Request {
	var seen *http.Request
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.WriteHeader(http.StatusOK)
	})
	NormalizationMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)
	return seen
}

func TestNormalizationMiddleware_TrimsHeaderWhitespace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "  value with spaces  ")

	seen := runNormalization(req)

	if got := seen.Header.Get("X-Custom"); got != "value with spaces" {
		t.Errorf("expected trimmed header, got %q", got)
	}
}

func TestNormalizationMiddleware_LowercasesContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "APPLICATION/JSON")

	seen := runNormalization(req)

	if got := seen.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected lowercase content-type, got %q", got)
	}
}

func TestNormalizationMiddleware_LowercasesAccept(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "APPLICATION/JSON")

	seen := runNormalization(req)

	if got := seen.Header.Get("Accept"); !strings.HasPrefix(got, "application") {
		t.Errorf("expected lowercase accept, got %q", got)
	}
}

func TestNormalizationMiddleware_RemovesTrailingSlash(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items/", nil)

	seen := runNormalization(req)

	if got := seen.URL.Path; got != "/api/v1/items" {
		t.Errorf("expected trailing slash removed, got %q", got)
	}
}

func TestNormalizationMiddleware_PreservesRootSlash(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	seen := runNormalization(req)

	if got := seen.URL.Path; got != "/" {
		t.Errorf("expected root slash preserved, got %q", got)
	}
}

func TestNormalizationMiddleware_CollapseDoubleSlashes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api//items", nil)

	seen := runNormalization(req)

	if got := seen.URL.Path; strings.Contains(got, "//") {
		t.Errorf("expected double slashes removed, got %q", got)
	}
}

func TestNormalizationMiddleware_NormalizesQueryBooleans(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api?active=TRUE&deleted=FALSE", nil)

	seen := runNormalization(req)

	q := seen.URL.Query()
	if q.Get("active") != "true" {
		t.Errorf("expected 'true', got %q", q.Get("active"))
	}
	if q.Get("deleted") != "false" {
		t.Errorf("expected 'false', got %q", q.Get("deleted"))
	}
}

func TestNormalizationMiddleware_NormalizesQuerySortOrder(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api?order=ASC&dir=DESC", nil)

	seen := runNormalization(req)

	q := seen.URL.Query()
	if q.Get("order") != "asc" {
		t.Errorf("expected 'asc', got %q", q.Get("order"))
	}
	if q.Get("dir") != "desc" {
		t.Errorf("expected 'desc', got %q", q.Get("dir"))
	}
}

func TestNormalizationMiddleware_UppercasesMethod(t *testing.T) {
	req := httptest.NewRequest("get", "/", nil)

	seen := runNormalization(req)

	if seen.Method != http.MethodGet {
		t.Errorf("expected GET, got %q", seen.Method)
	}
}

// TestNormalizationMiddleware_TrimsUserAgent covers the User-Agent normalizeHeaders branch.
func TestNormalizationMiddleware_TrimsUserAgent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "  Mozilla/5.0  ")

	seen := runNormalization(req)

	if got := seen.Header.Get("User-Agent"); got != "Mozilla/5.0" {
		t.Errorf("expected trimmed User-Agent, got %q", got)
	}
}

// TestNormalizationMiddleware_NilURLSkipsURLNormalization covers the nil URL guard in normalizeURL.
func TestNormalizationMiddleware_NilURLSkipsURLNormalization(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/items/", nil)
	// Setting URL to nil exercises the nil-guard early-return branches in
	// normalizeURL and normalizeQueryParams.
	req.URL = nil

	// Must not panic.
	var seen *http.Request
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		w.WriteHeader(http.StatusOK)
	})
	NormalizationMiddleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if seen == nil {
		t.Error("expected next to be called even with nil URL")
	}
}

// TestNormalizationMiddleware_NormalizesSortQueryParam covers "Sort"/"SORT"/"sort" cases.
func TestNormalizationMiddleware_NormalizesSortQueryParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api?a=SORT&b=Sort&c=sort", nil)

	seen := runNormalization(req)

	q := seen.URL.Query()
	for _, key := range []string{"a", "b", "c"} {
		if got := q.Get(key); got != "sort" {
			t.Errorf("expected %q='sort', got %q", key, got)
		}
	}
}

// TestNormalizationMiddleware_DefaultQueryParamPassesThrough covers the default case.
func TestNormalizationMiddleware_DefaultQueryParamPassesThrough(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api?name=alice", nil)

	seen := runNormalization(req)

	if got := seen.URL.Query().Get("name"); got != "alice" {
		t.Errorf("expected name='alice', got %q", got)
	}
}
