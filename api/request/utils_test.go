package request

import (
	"context"
	"net/http/httptest"
	"testing"

	ctxKeys "github.com/rdevitto86/komodo-forge-sdk-go/http/context"
)

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

func TestGenerateRequestId(t *testing.T) {
	id1 := GenerateRequestId()
	id2 := GenerateRequestId()
	if id1 == "" {
		t.Error("expected non-empty request ID")
	}
	if id1 == id2 {
		t.Error("expected unique request IDs")
	}
	// Should be hex encoded (24 hex chars for 12 bytes)
	if len(id1) != 24 {
		t.Errorf("expected 24-char hex ID, got %d chars: %q", len(id1), id1)
	}
}
