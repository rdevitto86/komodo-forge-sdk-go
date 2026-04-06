package request

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

func TestRequest_NewRequest(t *testing.T) {
	req, err := NewRequest("POST", "http://example.com/api", `{"test":true}`, map[string]string{"X-Custom": "val"}, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.Header.Get("X-Custom") != "val" {
		t.Errorf("X-Custom header missing")
	}
}

func TestRequest_NewRequest_Error(t *testing.T) {
	_, err := NewRequest("INVALID", "http://example.com", nil, nil, nil)
	if err == nil {
		t.Error("expected error for invalid method")
	}
}

func TestRequest_FromRequest(t *testing.T) {
	orig := httptest.NewRequest("PUT", "http://example.com/resource", strings.NewReader(`{}`))
	cloned, err := FromRequest(orig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloned.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", cloned.Method)
	}
}

func TestRequest_FromRequest_Nil(t *testing.T) {
	_, err := FromRequest(nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

func TestRequest_GenerateRequestId(t *testing.T) {
	id := GenerateRequestId()
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestRequest_GetAPIVersion(t *testing.T) {
	req := httptest.NewRequest("POST", "/v2/users", nil)
	req.Header.Set("Accept", "application/json;v=2")
	got := GetAPIVersion(req)
	if got != "/v2" {
		t.Errorf("GetAPIVersion = %q, want /v2", got)
	}
}

func TestRequest_GetAPIRoute(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/users/123", nil)
	got := GetAPIRoute(req)
	if got != "/users/123" {
		t.Errorf("GetAPIRoute = %q, want /users/123", got)
	}
}

func TestRequest_GetPathParams(t *testing.T) {
	req := httptest.NewRequest("POST", "/v1/users/123", nil)
	params := GetPathParams(req)
	if params == nil {
		t.Error("expected non-nil map")
	}
}

func TestRequest_GetQueryParams(t *testing.T) {
	req := httptest.NewRequest("POST", "/path?foo=bar", nil)
	params := GetQueryParams(req)
	if params["foo"] != "bar" {
		t.Errorf("foo = %q, want bar", params["foo"])
	}
}

func TestRequest_GetClientKey(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	got := GetClientKey(req)
	if got != "1.2.3.4" {
		t.Errorf("GetClientKey = %q, want 1.2.3.4", got)
	}
}

func TestRequest_GetClientType_Default(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	got := GetClientType(req)
	if got != "browser" {
		t.Errorf("GetClientType = %q, want browser", got)
	}
}

func TestRequest_IsValidAPIKey(t *testing.T) {
	got := IsValidAPIKey("any-key")
	if !got {
		t.Error("IsValidAPIKey should return true (placeholder)")
	}
}

func TestRequest_GetRequestID(t *testing.T) {
	t.Run("with request ID in context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		ctx := context.WithValue(req.Context(), ctxKeys.REQUEST_ID_KEY, "test-req-id")
		req = req.WithContext(ctx)
		got := GetRequestID(req)
		if got != "test-req-id" {
			t.Errorf("GetRequestID = %q, want 'test-req-id'", got)
		}
	})

	t.Run("without request ID in context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		got := GetRequestID(req)
		if got != "unknown" {
			t.Errorf("GetRequestID = %q, want 'unknown'", got)
		}
	})
}
