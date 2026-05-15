package httpresponse

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// marshalErrBody implements io.ReadCloser and json.Marshaler, returning an error on marshal.
type marshalErrBody struct{}

func (m marshalErrBody) Read(p []byte) (int, error) { return 0, io.EOF }
func (m marshalErrBody) Close() error               { return nil }
func (m marshalErrBody) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("marshal error")
}

func TestResponse_ResponseWriter_WriteHeader(t *testing.T) {
	t.Run("first call sets status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: rec}
		rw.WriteHeader(201)
		if rw.Status != 201 {
			t.Errorf("Status = %d, want 201", rw.Status)
		}
		if !rw.WroteHeader {
			t.Error("WroteHeader should be true")
		}
	})

	t.Run("second call does not change status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: rec}
		rw.WriteHeader(200)
		rw.WriteHeader(500)
		if rw.Status != 200 {
			t.Errorf("Status = %d, want 200 (first call wins)", rw.Status)
		}
	})
}

func TestResponse_ResponseWriter_Write(t *testing.T) {
	t.Run("auto-sets 200 when WriteHeader not called", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: rec}
		n, err := rw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 5 {
			t.Errorf("n = %d, want 5", n)
		}
		if rw.Status != http.StatusOK {
			t.Errorf("Status = %d, want 200", rw.Status)
		}
		if rw.BytesWritten != 5 {
			t.Errorf("BytesWritten = %d, want 5", rw.BytesWritten)
		}
	})

	t.Run("does not override already-set status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: rec}
		rw.WriteHeader(201)
		rw.Write([]byte("body"))
		if rw.Status != 201 {
			t.Errorf("Status = %d, want 201", rw.Status)
		}
	})

	t.Run("accumulates bytes written", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &ResponseWriter{ResponseWriter: rec}
		rw.Write([]byte("ab"))
		rw.Write([]byte("cde"))
		if rw.BytesWritten != 5 {
			t.Errorf("BytesWritten = %d, want 5", rw.BytesWritten)
		}
	})
}

func TestResponse_ResponseWriter_Unwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &ResponseWriter{ResponseWriter: rec}
	underlying := rw.Unwrap()
	if underlying != rec {
		t.Error("Unwrap should return the underlying ResponseWriter")
	}
}

func TestResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{199, false},
		{200, true},
		{201, true},
		{299, true},
		{300, false},
	}
	for _, tc := range tests {
		if got := IsSuccess(tc.status); got != tc.want {
			t.Errorf("IsSuccess(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestResponse_IsError(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{399, false},
		{400, true},
		{404, true},
		{500, true},
		{599, true},
		{600, false},
	}
	for _, tc := range tests {
		if got := IsError(tc.status); got != tc.want {
			t.Errorf("IsError(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestResponse_IsRedirect(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{299, false},
		{300, true},
		{301, true},
		{399, true},
		{400, false},
	}
	for _, tc := range tests {
		if got := IsRedirect(tc.status); got != tc.want {
			t.Errorf("IsRedirect(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestResponse_IsInformational(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{99, false},
		{100, true},
		{101, true},
		{199, true},
		{200, false},
	}
	for _, tc := range tests {
		if got := IsInformational(tc.status); got != tc.want {
			t.Errorf("IsInformational(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

func TestResponse_Bind(t *testing.T) {
	t.Run("nil response returns error", func(t *testing.T) {
		_, err := Bind(nil, nil)
		if err == nil {
			t.Error("expected error for nil response")
		}
	})

	t.Run("valid response", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{"X-Request-Id": []string{"abc123"}},
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}
		api, err := Bind(resp, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if api.Status != 200 {
			t.Errorf("Status = %d, want 200", api.Status)
		}
	})

	t.Run("body marshal error returns error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: 200,
			Header:     http.Header{},
			Body:       marshalErrBody{},
		}
		_, err := Bind(resp, nil)
		if err == nil {
			t.Error("expected error when body cannot be marshaled")
		}
	})
}

// Ensure httptest is used to avoid import warnings.
var _ = httptest.NewRecorder
