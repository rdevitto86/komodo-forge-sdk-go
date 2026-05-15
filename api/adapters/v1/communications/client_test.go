package communications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_Validates(t *testing.T) {
	if _, err := NewClient("", 1); err == nil {
		t.Fatal("expected error for empty baseURL")
	}
	if _, err := NewClient("http://x", 0); err == nil {
		t.Fatal("expected error for unsupported version 0")
	}
	if _, err := NewClient("http://x", 99); err == nil {
		t.Fatal("expected error for unsupported version 99")
	}
	if _, err := NewClient("http://x", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendOTP_PostsExpectedPayload(t *testing.T) {
	var got SendEmailRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/send/email" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		_ = json.NewEncoder(w).Encode(SendResult{MessageID: "msg_1"})
	}))
	defer srv.Close()

	c, err := NewClient(srv.URL, 1)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := c.SendOTP(context.Background(), "u@example.com", "123456", 300); err != nil {
		t.Fatalf("SendOTP: %v", err)
	}
	if got.To != "u@example.com" || got.TemplateID != "otp-request" {
		t.Fatalf("bad payload: %+v", got)
	}
	if got.TemplateData["code"] != "123456" {
		t.Fatalf("bad code: %v", got.TemplateData["code"])
	}
}
