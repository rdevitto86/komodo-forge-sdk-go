package httperrors

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestError_SendError(t *testing.T) {
	t.Run("no overrides", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		SendError(w, req, Global.BadRequest)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
		body, _ := io.ReadAll(resp.Body)
		var er ErrorResponse
		if err := json.Unmarshal(body, &er); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if er.Code != "10001" {
			t.Errorf("code = %q, want 10001", er.Code)
		}
		if er.Message != "Bad request" {
			t.Errorf("message = %q, want 'Bad request'", er.Message)
		}
	})

	t.Run("with message override", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		SendError(w, req, Global.NotFound, WithMessage("custom message"))

		body, _ := io.ReadAll(w.Result().Body)
		var er ErrorResponse
		json.Unmarshal(body, &er)
		if er.Message != "custom message" {
			t.Errorf("message = %q, want 'custom message'", er.Message)
		}
	})

	t.Run("with detail override", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		SendError(w, req, Global.Internal, WithDetail("some detail"))

		body, _ := io.ReadAll(w.Result().Body)
		var er ErrorResponse
		json.Unmarshal(body, &er)
		if er.Detail != "some detail" {
			t.Errorf("detail = %q, want 'some detail'", er.Detail)
		}
	})

	t.Run("with status override", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		SendError(w, req, Global.Internal, WithStatus(503))

		if w.Code != 503 {
			t.Errorf("status = %d, want 503", w.Code)
		}
	})

	t.Run("with all overrides via WithOverrides", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", "req-123")
		SendError(w, req, Global.Internal, WithOverrides("override msg", "override detail", 422))

		body, _ := io.ReadAll(w.Result().Body)
		var er ErrorResponse
		json.Unmarshal(body, &er)
		if er.Message != "override msg" {
			t.Errorf("message = %q, want 'override msg'", er.Message)
		}
		if er.Detail != "override detail" {
			t.Errorf("detail = %q, want 'override detail'", er.Detail)
		}
		if er.Status != 422 {
			t.Errorf("status = %d, want 422", er.Status)
		}
		if er.RequestId != "req-123" {
			t.Errorf("request_id = %q, want 'req-123'", er.RequestId)
		}
	})
}

func TestError_SendCustomError(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/custom", nil)
	SendCustomError(w, req, 418, "I'm a teapot", "brewing tea", "99001")

	if w.Code != 418 {
		t.Errorf("status = %d, want 418", w.Code)
	}
	body, _ := io.ReadAll(w.Result().Body)
	var er ErrorResponse
	json.Unmarshal(body, &er)
	if er.Code != "99001" {
		t.Errorf("code = %q, want 99001", er.Code)
	}
	if er.Message != "I'm a teapot" {
		t.Errorf("message = %q, want \"I'm a teapot\"", er.Message)
	}
	if er.Detail != "brewing tea" {
		t.Errorf("detail = %q, want 'brewing tea'", er.Detail)
	}
}

func TestError_WithMessage(t *testing.T) {
	o := WithMessage("hello")
	if o.Message == nil || *o.Message != "hello" {
		t.Errorf("WithMessage: got %v", o.Message)
	}
}

func TestError_WithDetail(t *testing.T) {
	o := WithDetail("detail text")
	if o.Detail == nil || *o.Detail != "detail text" {
		t.Errorf("WithDetail: got %v", o.Detail)
	}
}

func TestError_WithStatus(t *testing.T) {
	o := WithStatus(201)
	if o.Status == nil || *o.Status != 201 {
		t.Errorf("WithStatus: got %v", o.Status)
	}
}

func TestError_WithOverrides(t *testing.T) {
	o := WithOverrides("msg", "dtl", 200)
	if o.Message == nil || *o.Message != "msg" {
		t.Errorf("Message: got %v", o.Message)
	}
	if o.Detail == nil || *o.Detail != "dtl" {
		t.Errorf("Detail: got %v", o.Detail)
	}
	if o.Status == nil || *o.Status != 200 {
		t.Errorf("Status: got %v", o.Status)
	}
}

func TestAPIError_WithDetail(t *testing.T) {
	e := &APIError{Service: "komodo-auth-api", Code: "20001", Message: "unauthorized", Detail: "token expired"}
	got := e.Error()
	want := "komodo-auth-api [20001] unauthorized: token expired"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIError_WithoutDetail(t *testing.T) {
	e := &APIError{Service: "komodo-auth-api", Code: "20001", Message: "unauthorized"}
	got := e.Error()
	want := "komodo-auth-api [20001] unauthorized"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestCodeID(t *testing.T) {
	tests := []struct {
		rangeRoot int
		offset    int
		want      string
	}{
		{RangeUser, 1, "30001"},
		{RangeGlobal, 10, "10010"},
		{RangeAuth, 1, "20001"},
		{RangeDB, 5, "11005"},
		{RangeOrder, 999, "40999"},
		{RangeAnalytics, 1, "80001"},
	}

	for _, tc := range tests {
		got := CodeID(tc.rangeRoot, tc.offset)
		if got != tc.want {
			t.Errorf("CodeID(%d, %d) = %q, want %q", tc.rangeRoot, tc.offset, got, tc.want)
		}
	}
}
