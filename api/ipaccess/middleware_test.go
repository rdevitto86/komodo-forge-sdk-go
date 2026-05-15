package ipaccess

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// resetIPState resets the package-level sync.Once and list so each test
// can inject a fresh config via os.Setenv.
func resetIPState() {
	ipOnce = sync.Once{}
	lists = Lists{}
	os.Unsetenv("IP_WHITELIST")
	os.Unsetenv("IP_BLACKLIST")
}

func TestIPAccessMiddleware_AllowsUnlistedIP(t *testing.T) {
	resetIPState()
	// No whitelist, no blacklist — all IPs are allowed.

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.5")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for unlisted IP, got %d", rec.Code)
	}
}

func TestIPAccessMiddleware_BlocksBlacklistedIP(t *testing.T) {
	resetIPState()
	os.Setenv("IP_BLACKLIST", "10.0.0.1")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for blacklisted IP, got %d", rec.Code)
	}
}

func TestIPAccessMiddleware_AllowsNonBlacklistedIP(t *testing.T) {
	resetIPState()
	os.Setenv("IP_BLACKLIST", "10.0.0.1")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.2")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for non-blacklisted IP, got %d", rec.Code)
	}
}

func TestIPAccessMiddleware_AllowsWhitelistedIP(t *testing.T) {
	resetIPState()
	os.Setenv("IP_WHITELIST", "192.168.1.10")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.10")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for whitelisted IP, got %d", rec.Code)
	}
}

func TestIPAccessMiddleware_BlocksIPNotInWhitelist(t *testing.T) {
	resetIPState()
	os.Setenv("IP_WHITELIST", "192.168.1.10")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.99")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for IP not in whitelist, got %d", rec.Code)
	}
}

func TestIPAccessMiddleware_BlocksCIDRBlacklist(t *testing.T) {
	resetIPState()
	os.Setenv("IP_BLACKLIST", "10.0.0.0/24")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.50")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for IP in blacklisted CIDR, got %d", rec.Code)
	}
}

// TestIPAccessMiddleware_EmptyClientKey covers the branch where GetClientKey
// returns an empty string (RemoteAddr is empty, no X-Forwarded-For header).
func TestIPAccessMiddleware_EmptyClientKey(t *testing.T) {
	resetIPState()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Clear RemoteAddr so GetClientKey returns "".
	req.RemoteAddr = ""
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for empty client key, got %d", rec.Code)
	}
}

// TestIPAccessMiddleware_InvalidIPWithPort covers the path where the raw client
// value is not a bare IP but parses as "hostname:port" with a non-IP hostname,
// triggering the second ip == nil guard.
func TestIPAccessMiddleware_InvalidIPWithPort(t *testing.T) {
	resetIPState()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// "not-an-ip:8080" – SplitHostPort succeeds (host="not-an-ip") but
	// net.ParseIP("not-an-ip") returns nil → second ip==nil branch fires.
	req.Header.Set("X-Forwarded-For", "not-an-ip:8080")
	rec := httptest.NewRecorder()

	IPAccessMiddleware(okHandler()).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid IP, got %d", rec.Code)
	}
}
