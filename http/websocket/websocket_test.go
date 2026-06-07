package websocket

import (
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin_NoOriginHeader(t *testing.T) {
	SetAllowedOrigins(nil)
	req := httptest.NewRequest("GET", "/ws", nil)
	if !checkOrigin(req) {
		t.Error("expected a request with no Origin header to pass")
	}
}

func TestCheckOrigin_AllowlistedOrigin(t *testing.T) {
	SetAllowedOrigins([]string{"https://app.komodo.io"})
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://app.komodo.io")
	if !checkOrigin(req) {
		t.Error("expected an allowlisted origin to pass")
	}
}

func TestCheckOrigin_NonAllowlistedOriginRejected(t *testing.T) {
	SetAllowedOrigins([]string{"https://app.komodo.io"})
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	if checkOrigin(req) {
		t.Error("expected a non-allowlisted origin to be rejected")
	}
}

func TestCheckOrigin_EmptyAllowlistRejectsAllCrossOrigin(t *testing.T) {
	SetAllowedOrigins(nil)
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://app.komodo.io")
	if checkOrigin(req) {
		t.Error("expected a cross-origin request to be rejected when the allowlist is empty")
	}
}
